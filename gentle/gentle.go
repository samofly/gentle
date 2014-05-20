// gentle is a simple g-code sender compatible with TinyG.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"code.google.com/p/go.net/websocket"

	"github.com/samofly/gentle/tinyg"
	"github.com/samofly/serial"
)

var (
	ttyDev   = flag.String("dev", "/dev/ttyUSB0", "Serial device to open")
	baudRate = flag.Int("rate", 115200, "Baud rate")
	jsonMode = flag.Bool("json", true, "Whether to use TinyG json protocol. If false, just send raw gcode")
	web      = flag.Bool("web", false, "Whether to start a web interface")
	port     = flag.Int("port", 9000, "HTTP port (only used with if -web is active)")
)

func scan(s io.Reader, ch chan<- *tinyg.Response) {
	scanner := bufio.NewScanner(s)
	for scanner.Scan() {
		line := scanner.Text()
		if !*jsonMode {
			ch <- &tinyg.Response{Json: line}
			continue
		}
		r, err := tinyg.ParseResponse(line)
		if err != nil {
			log.Fatalf("Failed to parse TinyG response:\n%s\nerr: %v", line, err)
		}
		ch <- r
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("Failed to read from serial port:", err)
	}
	close(ch)
	log.Println("Serial port closed")
}

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

// state is the cnc machine state
type state struct {
	x float64
	y float64
	z float64
}

func (st *state) String() string {
	return fmt.Sprintf("[X: %.3f, Y: %.3f, Z: %3f]", st.x, st.y, st.z)
}

func send(s io.Writer, toCh <-chan string, respCh <-chan *tinyg.Response, outCh chan<- string) {
	st := &state{x: math.NaN(), y: math.NaN(), z: math.NaN()}

	must := func(cmd string) {
		fmt.Println(cmd)
		if _, err := fmt.Fprintln(s, cmd); err != nil {
			log.Fatal("Failed to write to serial port: ", err)
		}
	}

	proc := func(r *tinyg.Response) {
		outCh <- fmt.Sprintf("%v\n", r)
		if r.Mpox != nil {
			st.x = *r.Mpox
		}
		if r.Mpoy != nil {
			st.y = *r.Mpoy
		}
		if r.Mpoz != nil {
			st.z = *r.Mpoz
		}
		outCh <- fmt.Sprintf("State: %v\n", st)
	}

	for {
		select {
		case cmd := <-toCh:
			if cmd == "" {
				continue
			}
			must(cmd)
			if !*jsonMode {
				continue
			}
			// Waiting for TinyG to confirm it
			for {
				resp := <-respCh
				if resp == nil {
					// channel is closed
					return
				}
				proc(resp)
				if resp.Footer != nil {
					break
				}
			}
		case resp := <-respCh:
			if resp == nil {
				// channel is closed
				return
			}
			if !*jsonMode {
				outCh <- fmt.Sprintf("%s\n", resp.Json)
				continue
			}
			proc(resp)
		}
	}
}

type server struct {
	toCh chan<- string
	ps   *pubsub
}

func (s *server) Serve(ws *websocket.Conn) {
	respCh := make(chan string, 10)
	s.ps.Sub(respCh)
	go print(ws, respCh)
	defer ws.Close()
	in := bufio.NewScanner(ws)
	for in.Scan() {
		s.toCh <- in.Text()
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

func runWeb(port int, toCh chan<- string, ps *pubsub) {
	s := &server{toCh, ps}
	http.Handle("/ws", websocket.Handler(s.Serve))
	http.HandleFunc("/", handleEmbed)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}

type pubsub struct {
	mu    sync.Mutex
	pubCh chan string
	ch    []chan<- string
}

func newPubSub() *pubsub {
	ps := &pubsub{pubCh: make(chan string)}
	go ps.run()
	return ps
}

func (ps *pubsub) Sub(ch chan<- string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.ch = append(ps.ch, ch)
}

func (ps *pubsub) run() {
	for msg := range ps.pubCh {
		ps.mu.Lock()
		list := ps.ch
		ps.mu.Unlock()
		for _, ch := range list {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

func (ps *pubsub) Pub() chan<- string {
	return ps.pubCh
}

func print(w io.Writer, ch <-chan string) {
	for msg := range ch {
		if _, err := fmt.Fprint(w, msg); err != nil {
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

	ps := newPubSub()
	printCh := make(chan string, 10)
	ps.Sub(printCh)
	go print(os.Stdout, printCh)

	respCh := make(chan *tinyg.Response)
	toCh := make(chan string)

	if *web {
		go runWeb(*port, toCh, ps)
	}

	go scan(s, respCh)
	go send(s, toCh, respCh, ps.Pub())

	// init
	toCh <- `{"sr":""}`

	fmt.Fprintln(os.Stderr, "Please, enter g-code lines below:")
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		if !*jsonMode {
			toCh <- strings.TrimSpace(in.Text())
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
		toCh <- fmt.Sprintf(`{"gc":"%s"}`, gcode)
	}
	if err := in.Err(); err != nil {
		log.Fatal("Failed to read from stdin: ", err)
	}
}
