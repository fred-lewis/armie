package armie

import (
	"net"
	"io"
)

type tcpTransport struct {}

func (tcpTransport) Dial(address string) (*transportConn, error) {
	sock, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return &transportConn{
		Socket: sock,
		Address: address,
	}, nil
}

func (tcpTransport) Listen(address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}

func NewTCPServer(logout io.Writer) *Server {
	return newServer(logout, &tcpTransport{})
}

func NewTCPConnection(addr string, logout io.Writer, handler ConnectionHandler) (*Conn, error) {
	conn, err := tcpTransport{}.Dial(addr)
	if err != nil {
		return nil, err
	}
	return newConn(conn, logout, handler)
}