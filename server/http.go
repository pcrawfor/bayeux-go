package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	eventsource "github.com/antage/eventsource/http"
	"github.com/gorilla/websocket"
)

// Connection represents a websocket connection along with reader and writer state
type Connection struct {
	ws          *websocket.Conn
	es          eventsource.EventSource
	send        chan []byte
	isWebsocket bool
}

func (c *Connection) esWriter(f *Server) {
	fmt.Println("Writer started.")
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.es.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.esWrite([]byte{})
				return
			}
			if err := c.esWrite([]byte(message)); err != nil {
				return
			}
		case <-ticker.C:
			fmt.Println("tick.")
			if err := c.esWrite([]byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Connection) esWrite(payload []byte) error {
	fmt.Println("Writing to eventsource: ", string(payload))
	c.es.SendMessage(string(payload), "", "")
	return nil
}

// serverOther handle non WS requests
func serveOther(w http.ResponseWriter, r *http.Request) {
	fmt.Println("serve other: ", r.URL)

	fmt.Println("REQUEST URL: ", r.URL.Path)
	fmt.Println("REQUEST RAW QUERY: ", r.URL.RawQuery)
	fmt.Println("REQUEST HEADER ", r.Header)

	if isEventSource(r) {
		fmt.Println("Looks like event source")
		handleEventSource(w, r)
	}
	w.WriteHeader(http.StatusOK)
	return
}

func handleEventSource(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle event source: ", r.URL.Path)
	// create a new connection for the event source action
	clientID := strings.Split(r.URL.Path, "/")[2]
	fmt.Println("clientID: ", clientID)
	es := eventsource.New(nil, nil)
	c := &Connection{send: make(chan []byte, 256), es: es, isWebsocket: false}
	// TODO: NEED TO ASSOCIATED THE EXISTING FAYE CLIENT INFO/SUBSCRIPTIONS WITH THE CONNECTION CHANNEL
	// USE CLIENT ID TO UPDATE FAYE INFO WITH ES CONNETION CHANNEL
	f.UpdateClientChannel(clientID, c.send)
	go c.esWriter(f)
	c.es.ServeHTTP(w, r)
	return
}

// serverWs - provides an http handler for upgrading a connection to a websocket connection
func serveWs(w http.ResponseWriter, r *http.Request) {
	fmt.Println("serveWs")
	fmt.Println("METHOD: ", r.Method)
	fmt.Println("REQUEST URL: ", r.URL.Path)
	fmt.Println("REQUEST RAW QUERY: ", r.URL.RawQuery)
	fmt.Println("REQUEST HEADER ", r.Header)

	// handle options
	// if r.Method == "OPTIONS" || r.Header.Get("Access-Control-Request-Method") == "POST" {
	// 	handleOptions(w, r)
	// }

	// if isEventSource(r) {
	// 	fmt.Println("Is event source")
	// }

	// if r.Method != "GET" {
	// 	http.Error(w, "Method not allowed", 405)
	// 	//serveLongPolling(f, w, r)
	// 	return
	// }

	/*
	   if r.Header.Get("Origin") != "http://"+r.Host {
	           http.Error(w, "Origin not allowed", 403)
	           return
	   }
	*/

	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		//http.Error(w, "Not a websocket handshake", 400)
		fmt.Println("ERR:", err)
		fmt.Println("NOT A WEBSOCKET HANDSHAKE")
		serveJSONP(f, w, r)
		return
	} else if err != nil {
		fmt.Println(err)
		return
	}
	c := &Connection{send: make(chan []byte, 256), ws: ws, isWebsocket: true}
	go c.writer(f)
	c.reader(f)
}

// handleOptions allows for access control awesomeness
func handleOptions(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle options!")
	w.Header().Set("Access-Control-Allow-Credentials", "false")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Pragma, X-Requested-With")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusOK)
	return
}

// isEventSource
func isEventSource(r *http.Request) bool {
	fmt.Println("isEventSource? ", r.Method)
	if r.Method != "GET" {
		return false
	}

	accept := r.Header.Get("Accept")
	fmt.Println("Accept: ", accept)
	return accept == "text/event-stream"
}

var f *Server

// Start inits the http server on the address/port given in the addr param
func Start(addr string) {
	f = NewServer()
	http.HandleFunc("/bayeux", serveWs)
	http.HandleFunc("/", serveOther)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
