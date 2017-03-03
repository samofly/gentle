// websocket-client is a command line utility to connect to gentle server via WebSockets.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/net/websocket"
)

var url = flag.String("url", "ws://echo.websocket.org:80", "URL to connect to")
var origin = flag.String("origin", "http://localhost/", "Origin URL to send to server")

func read(r io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(buf[:n]))
	}
}

func main() {
	flag.Parse()

	if *url == "" {
		log.Fatal("-url not specified")
	}

	ws, err := websocket.Dial(*url, "", *origin)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	go read(ws)

	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		if _, err := ws.Write([]byte(fmt.Sprintf("%s\n", s.Text()))); err != nil {
			log.Fatal(err)
		}
	}
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}
}
