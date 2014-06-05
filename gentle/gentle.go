// gentle is a simple g-code sender compatible with TinyG.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"code.google.com/p/go.net/websocket"

	"github.com/samofly/gentle/engine"
	"github.com/samofly/serial"
)

var (
	ttyDev   = flag.String("dev", "/dev/ttyUSB0", "Serial device to open")
	baudRate = flag.Int("rate", 115200, "Baud rate")
	jsonMode = flag.Bool("json", true, "Whether to use TinyG json protocol. If false, just send raw gcode")
	web      = flag.Bool("web", false, "Whether to start a web interface")
	port     = flag.Int("port", 9000, "HTTP port (only used with if -web is active)")
)

// sanitizeG handle Gnn commands. cmd is upper-case, trimmed and starts with 'G'
func sanitizeG(cmd string) (string, error) {
	return cmd, nil
}

// sanitizeCmd checks gcode command and returns a canonically-formatted gcode without comments.
func sanitizeCmd(cmd string) (string, error) {
	cmd = strings.ToUpper(strings.TrimSpace(cmd))
	if cmd == "" {
		return "", nil
	}

	// TODO(krasin): implement a proper gcode parser.
	switch cmd[0] {
	case 'G':
		return sanitizeG(cmd)
	case 'F':
		return cmd, nil
	case 'M':
		return cmd, nil
	default:
		return "", fmt.Errorf("sanitizeCmd(%q): %q command not recognized", cmd, cmd[0])
	}

	return cmd, nil
}

type server struct {
	m engine.Machine
}

func downstream(w io.Writer, ch <-chan *engine.Message) {
	for msg := range ch {
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error: failed to marshal json for %+v, err: %v", msg, err)
			return
		}
		if _, err := w.Write(data); err != nil {
			log.Print("Error: failed to deliver message, err: ", err)
			return
		}
	}
}

type webRequest struct {
	Raw string `json:"raw"`
}

func (s *server) Serve(ws *websocket.Conn) {
	defer log.Printf("Connection closed.")
	defer ws.Close()

	go downstream(ws, s.m.Sub())

	in := bufio.NewScanner(ws)
	var js jsonSplitter
	in.Split(js.Split)
	for in.Scan() {
		log.Printf("incoming json message: %s", in.Text())
		var req webRequest
		if err := json.Unmarshal(in.Bytes(), &req); err != nil {
			log.Printf("Failed to unmarshal incoming request: %v, err: %v", in.Bytes(), err)
			return
		}
		if req.Raw == "" {
			log.Printf("Only raw messages are currently supported")
			continue
		}
		s.m.Send(req.Raw)
	}
	if err := in.Err(); err != nil {
		log.Printf("Error while reading from connection with %v: %v", ws.RemoteAddr(), err)
	}
}

func handleEmbed(w http.ResponseWriter, req *http.Request) {
	p := path.Clean(req.URL.Path)

	// go-bindata generates relative paths.
	if strings.HasPrefix(p, "/") {
		p = p[1:]
	}
	if p == "" {
		p = "index.html"
	}
	data, err := Asset(p)
	if err != nil {
		log.Printf("Error: Asset(%q): %v", p, err)
		http.NotFound(w, req)
		return
	}
	http.ServeContent(w, req, p, time.Time{}, bytes.NewReader(data))
}

func runWeb(port int, m engine.Machine) {
	s := &server{m: m}
	http.Handle("/ws", websocket.Handler(s.Serve))
	http.HandleFunc("/", handleEmbed)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

func print(w io.Writer, ch <-chan *engine.Message) {
	for msg := range ch {
		str := msg.Raw
		if str == "" {
			data, err := json.Marshal(msg)
			if err != nil {
				log.Print("Error: failed to marshal a message to json, err: ", err)
				return
			}
			str = string(data)
		}
		if !strings.HasSuffix(str, "\n") {
			str = str + "\n"
		}
		if _, err := fmt.Fprint(w, str); err != nil {
			log.Print("Error: failed to deliver message, err: ", err)
			return
		}
	}
}

func main() {
	flag.Parse()

	if *ttyDev == "" {
		log.Fatal("-dev (serial device) is not specified.")
	}
	s, err := serial.Open(*ttyDev, *baudRate)
	if err != nil {
		log.Fatalf("Could not open serial port at %s: %v", *ttyDev, err)
	}
	defer s.Close()
	log.Print("Port opened at ", *ttyDev)

	m := engine.New(s, *jsonMode)

	go print(os.Stdout, m.Sub())

	if *web {
		go runWeb(*port, m)
	}

	// init
	m.Send(`{"sr":""}`)

	fmt.Fprintln(os.Stderr, "Please, enter g-code lines below:")
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		if !*jsonMode {
			m.Send(strings.TrimSpace(in.Text()))
			continue
		}
		gcode, err := sanitizeCmd(in.Text())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid gcode: %v\n", err)
			// Invalid / unrecognized gcode is a halting condition,
			// because we don't know what's the supposed CNC state after this line,
			// and it may hurt the part and the mill.
			os.Exit(1)
		}
		if gcode == "" {
			continue
		}
		m.Send(fmt.Sprintf(`{"gc":"%s"}`, gcode))
	}
	if err := in.Err(); err != nil {
		log.Fatal("Failed to read from stdin: ", err)
	}
}
