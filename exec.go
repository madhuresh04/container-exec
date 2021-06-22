package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"k8s.io/client-go/rest"
	//"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	//"time"
)

type RoundTripCallback func(c *websocket.Conn) error

type WebsocketRoundTripper struct {
	TLSConfig *tls.Config
	Callback  RoundTripCallback
}

var protocols = []string{
	"v4.channel.k8s.io",
	"v3.channel.k8s.io",
	"v2.channel.k8s.io",
	"channel.k8s.io",
}

const (
	stdin = iota
	stdout
	stderr
)
var cacheBuff bytes.Buffer
var send chan []byte
var out chan string

type ExecOptions struct {
	Namespace string
	Pod       string
	Container string
	Command   []string
	TTY       bool
	Stdin     bool
}

func WebsocketCallback(c *websocket.Conn) error {
	errChan := make(chan error, 3)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1025)
		for m := range send {
			if len(m) >= 1 {
				m = append(m, byte(10))
			}
			copy(buf[1:], m)
			n := len(m)
			cacheBuff.Write(buf[1:n+1])
			cacheBuff.Write([]byte{13, 10})
			if err := c.WriteMessage(websocket.TextMessage, buf[:n+1]); err != nil {
				errChan <- err
				fmt.Printf("Error in Write Message to connection : %+v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			_, buf, err := c.ReadMessage()
			if err != nil {
				errChan <- err
				fmt.Printf("Error in Read Message from Connection : %+v", err)
				return
			}

			if len(buf) > 1 {
				var w io.Writer
				switch buf[0] {
				case stdout:
					w = os.Stdout
				case stderr:
					w = os.Stderr
				}

				if w == nil {
					fmt.Println("Continue in the loop")
					continue
				}
				s := strings.Replace(string(buf[1:]), cacheBuff.String(), "", -1)
				out <- s
				if err != nil {
					errChan <- err
					fmt.Printf("Error in Write io.Writer : %+v", err)
					return
				}
			}
			cacheBuff.Reset()
		}
	}()

	wg.Wait()
	fmt.Println("End of Infinite Loop")
	close(errChan)
	err := <-errChan
	return err
}

func ExecRequest(config *rest.Config, opts *ExecOptions) (*http.Request, error) {
	u, err := url.Parse(config.Host)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return nil, fmt.Errorf("Unrecognised URL scheme in %v", u)
	}

	u.Path = fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", opts.Namespace, opts.Pod)

	rawQuery := "stdout=true&tty=true&stderr=true"
	for _, c := range opts.Command {
		rawQuery += "&command=" + c
	}

	if opts.Container != "" {
		rawQuery += "&container=" + opts.Container
	}

	if opts.Stdin {
		rawQuery += "&stdin=true"
	}
	u.RawQuery = rawQuery

	return &http.Request{
		Method: http.MethodGet,
		URL:    u,
	}, nil
}

func DialerFunc( config *rest.Config,r *http.Request) (*websocket.Conn, error) {
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil,err
	}

	dialer := &websocket.Dialer{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
		Subprotocols:    protocols,
	}
	fmt.Println("Dailing the connection")
	requestHeader := make(http.Header)
	requestHeader.Add("Authorization" , "Bearer "+ config.BearerToken)
	conn, _, err := dialer.Dial(r.URL.String(), requestHeader)
	fmt.Println("Out of Dailing connection")
	//fmt.Println(resp)
	if err != nil {
		fmt.Printf("Error in Dailer Dail (Pod WS connection) : %+v\n", err)
		return nil, err
	}
	return conn,nil
}

func RoundTrip(config *rest.Config,r *http.Request) error {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	conn, err := DialerFunc(config,r)
	if err != nil {
		fmt.Printf("Error in Dial connection : %+v",err)
		return err
	}
	defer conn.Close()
	err = WebsocketCallback(conn)
	if err != nil {
		fmt.Println("How dis this fuction end, it should never end", err)
	}
	return nil
}

