package armie

import (
	"math/rand"
	"os"
	"fmt"
	"testing"
	"strconv"
	"time"
	"github.com/fred-lewis/armie/log"
)

var test_logger *log.Logger
var test_server *Server
var test_conn *Conn
var test_addr string
var msg_res_event string
var msg_res_person person

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		fmt.Printf("ERROR setting up tests: %s", err)
		os.Exit(1)
	}
	code := m.Run()
	os.Exit(code)
}

func TestInt(t *testing.T) {
	f, err := test_conn.SendRequest("INTTEST", 5, 2)
	if err != nil {
		t.Error(err)
	}
	var res int
	f.GetResult(&res)
	if res != 10 {
		t.Fail()
	}
}

func TestString(t *testing.T) {
	f, err := test_conn.SendRequest("STRINGTEST", "123456789012345")
	if err != nil {
		t.Error(err)
	}
	var res int
	f.GetResult(&res)
	if res != 15 {
		t.Fail()
	}
}

func TestObject(t *testing.T) {
	f, err := test_conn.SendRequest("OBJTEST", &person{
		Name: "bob",
		Age: 44,
	})
	if err != nil {
		t.Error(err)
	}
	var res string
	f.GetResult(&res)
	if res != "bob is 44" {
		t.Logf("Got result: %s", res)
		t.Fail()
	}
}

func TestObjectRet(t *testing.T) {
	f, err := test_conn.SendRequest("OBJRETTEST")
	if err != nil {
		t.Error(err)
	}
	var res person
	f.GetResult(&res)
	if res.Age != 45 || res.Name != "bill" {
		t.Fail()
	}
}

func TestEvents(t *testing.T) {
	err := test_conn.SendEvent("EVENT123", person{ 27, "curt"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if msg_res_event != "EVENT123" {
		t.Logf("Got wrong event: %s", msg_res_event)
		t.Fail()
	}
	if msg_res_person.Name != "curt" || msg_res_person.Age != 27 {
		t.Logf("Got wrong person: %v", msg_res_person)
		t.Fail()
	}
}

func intTest(a, b int) int {
	return a * b
}

func stringTest(s string) int {
	return len(s)
}

type person struct {
	Age int
	Name string
}

func objTest(p *person) string {
	return p.Name + " is " + strconv.Itoa(p.Age)
}

func objRetTest() person {
	return person{45, "bill"}
}

func handleRequest(req *Request, response *Response) {
	switch req.Method {
	case "INTTEST":
		res, err := req.CallMethod(intTest)
		if err != nil {
			response.Error(err.Error())
		}
		response.Send(res)
		break
	case "STRINGTEST":
		res, err := req.CallMethod(stringTest)
		if err != nil {
			response.Error(err.Error())
		}
		response.Send(res)
	case "OBJTEST":
		res, err := req.CallMethod(objTest)
		if err != nil {
			response.Error(err.Error())
		}
		response.Send(res)
	case "OBJRETTEST":
		res, err := req.CallMethod(objRetTest)
		if err != nil {
			response.Error(err.Error())
		}
		response.Send(res)
	}
}

func handleMessage(evt *Event) {
	msg_res_event= evt.Event
	evt.Decode(&msg_res_person)
}

func setup() error {
	var err error

	test_logger = log.New(os.Stdout)

	for i := 0; i < 5; i++ {
		port := 46000 + rand.Intn(1000)
		test_addr = "localhost:" + strconv.Itoa(port)
		s := NewTCPServer(os.Stdout)
		s.OnConnection(func(conn *Conn) error {
			conn.OnRequest(handleRequest)
			conn.OnEvent(handleMessage)
			return nil
		})
		err := s.Listen(test_addr)
		if err == nil {
			test_server = s
			break
		} else {
			test_logger.Warn("could not bind to %s: %s", test_addr, err)
		}
	}

	if test_server == nil {
		return fmt.Errorf("Could not bind to port")
	}

	test_logger.Info("Bound server to %s", test_addr)

	test_conn, err = NewTCPConnection(test_addr, os.Stdout, nil)

	if err != nil {
		return err
	}

	return nil
}

