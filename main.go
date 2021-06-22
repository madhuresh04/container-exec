package main

import (
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)
const KubeConfigPath = "/Users/madhuresh.kumar/.kube/config"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// define a reader which will listen for
// new messages being sent to our WebSocket
// endpoint
func reader(conn *websocket.Conn ) {
	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil && len(p) != 0 {
			alltime <- (string)(p)
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Printf("error: %v", err)
			}
			log.Printf("error: %v", err)
			return
		}
		processMessage(messageType,string(p))

	}
}

func writer(conn *websocket.Conn) {
	for m := range out {
		fmt.Println(m)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(m)); err != nil {
			log.Println(err)
			return
		}
	}
}
func processMessage(a int,  s string) string{
	send <- []byte(s)
	return s
}

func homePage (w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Page Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}


func SendAndReceive (w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w,r,nil)
	if err != nil {
		log.Println(err)
	}
	log.Println("Client Connected to v2 Version")
	// Send Ping Message
	//go SendPingAtInterval(ws, 5 * time.Second)
	send = make(chan []byte)
	out = make(chan string)
	alltime = make(chan string)
	// Read Function
	go callFunc()
	go writer(ws)
	go alwaysWriteToUI(ws)
	reader(ws)

}
// Prints the same message back to UI
func alwaysWriteToUI(conn *websocket.Conn) {
	for m := range alltime {
		fmt.Println(m)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(m)); err != nil {
			log.Println(err)
			return
		}
	}
}

func callFunc() {
	var cmd []string
	cmd = append(cmd, "bin/bash" )
	opts := &ExecOptions{
		Namespace: "default",
		Pod:       "ws-test1",
		Container: "kube-attach",
		Command:   cmd,
		TTY:       true,
		Stdin:     true,
	}
	config, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	req, err := ExecRequest(config, opts)
	if err != nil {
		log.Fatalln(err)
	}

	if  err = RoundTrip(config,req); err != nil {
		log.Fatalln(err)
	}
}
func SendPingAtInterval(ws *websocket.Conn, duration time.Duration) {
	ch := time.Tick(duration)
	for range ch {
		log.Println( "Sent Ping Message")
		ws.WriteControl(websocket.PingMessage, ([]byte)("PING MESSAGE"), time.Time{} )
	}
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/v2/ws", SendAndReceive)

}

func main() {
	fmt.Println("Go Websockets")
	setupRoutes()
	log.Fatal(http.ListenAndServe(":3000", nil))
}
