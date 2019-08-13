
## Armie

#### Async-RMI and Eventing framework.

Package armie provides a framework for async, symmetric RMI and
Eventing.  Connections facilitate both Requests (with responses)
and Events (no response).  Requests are issued asynchronously,
and a single connection can have any number of outstanding requests
(a map of request-ids is kept in order to route responses).  Issuing a
request returns a Future that can be used to acquire a response.

A single Armie connection can process tens of thousands of
requests-per-second.

Example:
```
type Person struct {
	Name string
	Age  int
}

func sayHello(person *Person) int {
	fmt.Printf("%s says 'Hello'.\n", person.Name)
	return person.Age
}

var handleReq armie.RequestHandler = func(request *armie.Request, response *armie.Response) {
	switch request.Method {
	case "HELLO":
		age, err := request.CallMethod(sayHello)
		if err != nil {
			response.Error(err.Error())
		} else {
			response.Send(age)
		}
	}
}

var handleEvt armie.EventHandler = func(event *armie.Event) {
	switch event.Event {
	case "ARRIVED":
		var name string
		event.Decode(&name)
		fmt.Printf("%s has arrived.\n", name)
	}
}

func main() {
	s := armie.NewTCPServer(os.Stdout)

	// ConnectionHandler should wire up handlers on each
	// new connection to the server
	s.OnConnection(func(conn *armie.Conn) error {
		conn.OnRequest(handleReq)
		conn.OnEvent(handleEvt)
		return nil
	})


	err := s.Listen(":9999")
	if err != nil {
		fmt.Printf("Error setting up server: %s", err.Error())
	}

	// The client connection can optionally accept a connectionHandler,
	// or Request / Event Handlers can be wired up later with OnRequest
	// and OnEvent.  The former method is necessary if you expect a request
	// or event to be waiting on the wire immediately.
	conn, err := armie.NewTCPConnection(":9999", os.Stdout, nil)
	if err != nil {
		fmt.Printf("Error connecting: %s", err.Error())
	}

	joe := Person{
		Name: "Joe",
		Age: 30,
	}
	conn.SendEvent("ARRIVED", "Joe")

	res, err := conn.SendRequest("HELLO", &joe)
	var joesAge int
	res.GetResult(&joesAge)
	fmt.Printf("Joe is %d.\n", joesAge)
}
```
Output:
```
Joe has arrived.
Joe says 'Hello'.
Joe is 30.
```

