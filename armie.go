package armie

import (
	"net"
	"io"
	"fmt"
	"bufio"
	"sync"
	"github.com/ugorji/go/codec"
	"errors"
	"github.com/fred-lewis/armie/log"
)

const bufSize = 8192

type transportConn struct {
	Socket io.ReadWriteCloser
	Address string
}

type transport interface {
	Dial(address string) (*transportConn, error)
	Listen(address string) (net.Listener, error)
}

type RequestHandler func(request *Request, response *Response)
type EventHandler func(event *Event)
type ConnectionHandler func(conn *Conn) error

//
// Server provides a Listen(addr) method for accepting new connections.
// Register a ConnectionHandler with OnConnection() and use the
// ConnectionHandler to wire up a Request and / or Event handler,
// and do any other housekeeping required before requests and
// events are accepted.
//
type Server struct {
	logger       *log.Logger
	addr         string
	shutdown     bool
	transport    transport
	connHandler  ConnectionHandler
	listener     net.Listener
	shutdownChan chan int
}

func newServer(logout io.Writer, transport transport) *Server {
	return &Server{
		logger:       log.New(logout),
		transport:    transport,
		shutdown:     false,
		shutdownChan: make(chan int),
	}
}

//
// Use the ConnectionHandler to wire up a Request and / or Event
// handler, and do any other housekeeping required before requests and
// events are accepted.
//
func (serv *Server) OnConnection(handler ConnectionHandler) {
	serv.connHandler = handler
}

//
// Listen at the given address for new connections.  Address
// will be passed unmodified down to the underlying transport.
//
func (serv *Server) Listen(addr string) error {

	serv.addr = addr

	ln, err := serv.transport.Listen(addr)
	if err != nil {
		return fmt.Errorf("[RPC] Could not bind on " + addr)
	}

	serv.logger.Info("[RPC] Listening on %s for RPC connections", addr)

	go func() {
		for !serv.shutdown {
			con, err := ln.Accept()

			if serv.shutdown {
				serv.logger.Info("[RPC] Shutting down listener on %s", addr)
				break
			}

			if err != nil {
				serv.logger.Error("[RPC] accepting connection on %s: %v", addr, err)
				break
			}

			bw := bufio.NewWriterSize(con, bufSize)
			br := bufio.NewReaderSize(con, bufSize)

			c := &Conn{
				Alive: true,
				conn: con,
				outstanding: make(map[uint64]*Future),
				logger: serv.logger,
				bw: bw,
				br: br,
				dec: codec.NewDecoder(br, &mph),
				enc: codec.NewEncoder(bw, &mph),
				addr: con.RemoteAddr().String(),
			}

			err = serv.connHandler(c)
			if err != nil {
				serv.logger.Error("[RPC] initializing connection on %s: %v", addr, err)
				con.Close()
				continue
			}

			go c.serve()
		}

		ln.Close()
		serv.shutdownChan <- 1
	}()

	return nil
}

//
// Stop listening.  Close will not return until listener
// goroutine has exited.
//
func (serv *Server) Close() error {
	serv.shutdown = true
	serv.listener.Close()
	<-serv.shutdownChan
	return nil
}

//
// Conn is Symmetric.  Both server and client can register
// RequestHandlers and EventHandlers, and both can SendEvent() and
// SendRequest().
//
type Conn struct {
	Alive bool
	conn io.ReadWriteCloser
	outstanding map[uint64]*Future
	mu sync.Mutex
	connmu sync.Mutex
	logger *log.Logger
	bw *bufio.Writer
	br *bufio.Reader
	dec *codec.Decoder
	enc *codec.Encoder
	addr string
	reqHandler RequestHandler
	evtHandler EventHandler
	shutdownChan chan int
}

func newConn(transportConn *transportConn, logout io.Writer, handler ConnectionHandler) (*Conn, error) {

	bw := bufio.NewWriterSize(transportConn.Socket, bufSize)
	br := bufio.NewReaderSize(transportConn.Socket, bufSize)

	c := &Conn{
		Alive: true,
		conn: transportConn.Socket,
		outstanding: make(map[uint64]*Future),
		logger: log.New(logout),
		bw: bw,
		br: br,
		dec: codec.NewDecoder(br, &mph),
		enc: codec.NewEncoder(bw, &mph),
		addr: transportConn.Address,
		shutdownChan: make(chan int),
	}

	if handler != nil {
		err := handler(c)
		if err != nil {
			c.Close()
			return nil, err
		}
	}

	go c.serve()

	return c, nil
}

//
// Send an asynchronous RMI Request.  Multiple arguments are supported.
// Returns a Future that can be used to await the result.
//
func (c *Conn) SendRequest(method string, args ... interface{}) (*Future, error) {
	if !c.Alive {
		return nil, fmt.Errorf("request on inactive connection")
	}

	req := &Request{
		Method: method,
		Id: genID(),
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := encodeRequest(c, req, args)
	if err != nil {
		return nil, err
	}


	f := newFuture()

	c.outstanding[req.Id] = f

	return f, nil
}

//
// Send an asynchronous Event.  Events should have an event name, and
// a single data object.  Events are one-way communications and there's
// no guarantee they arrive if the connection is lost.
//
func (c *Conn) SendEvent(method string, data interface{}) error {
	if !c.Alive {
		return fmt.Errorf("send event on inactive connection")
	}

	return encodeEvent(c, method, data)
}

//
// Register a request handler.
//
func (c *Conn) OnRequest(handler RequestHandler) {
	c.reqHandler = handler
}

//
// Register an event handler.
//
func (c *Conn) OnEvent(handler EventHandler) {
	c.evtHandler = handler
}

//
// Close the connection.  Close will not return until the
// goroutine reading events and requests exits.
//
func (c *Conn) Close() error {
	if !c.Alive {
		return fmt.Errorf("shutdown on inactive connection")
	}

	c.Alive = false
	c.conn.Close()
	<-c.shutdownChan
	return nil
}

func (c *Conn) handleResponse(frm *frame) {
	c.mu.Lock()
	f := c.outstanding[frm.Id]
	delete(c.outstanding, frm.Id)
	c.mu.Unlock()

	if frm.Error != "" {
		f.error(errors.New(frm.Error))
	} else {
		f.complete(frm)
	}
}

func (c *Conn) handleEvent(frm *frame) {
	evt := &Event{
		Event: frm.Method,
		Payload: frm.Payload,
	}

	c.evtHandler(evt)
}

func (c *Conn) handleRequest(frm *frame) {

	req := &Request{
		Method: frm.Method,
		Id: frm.Id,
		Payload: frm.Payload,
	}

	response := &Response{
		Id: frm.Id,
		conn: c,
	}

	c.reqHandler(req, response)
}

func (c *Conn) serve() {
	c.Alive = true
	for {
		frm, err := readFrame(c)
		if err != nil {
			c.Alive = false
			c.logger.Error("[RPC] error reading RPC frame: %v", err)
			return
		}
		switch frm.Type {
		case RESPONSE:
			c.logger.Trace("[RPC] Response from %v. Err: %v",
				c.addr, frm.Error)

			c.handleResponse(frm)
		case REQUEST:
			c.logger.Trace("[RPC] Request from %v. %v",
				c.addr, frm.Method)

			c.handleRequest(frm)
		case EVENT:
			c.logger.Trace("[RPC] Event from %v. %v",
				c.addr, frm.Method)

			c.handleEvent(frm)
		}
	}
	c.shutdownChan <- 1
}