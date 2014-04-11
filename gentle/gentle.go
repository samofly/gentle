// gentle is a simple g-code sender compatible with TinyG.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"

	"github.com/samofly/serial"
)

var (
	ttyDev   = flag.String("dev", "/dev/ttyUSB0", "Serial device to open")
	baudRate = flag.Int("rate", 115200, "Baud rate")
	jsonMode = flag.Bool("json", true, "Whether to use TinyG json protocol. If false, just send raw gcode")
)

type response struct {
	line string
	m    map[string]interface{}
}

func scan(s io.Reader, ch chan<- *response) {
	scanner := bufio.NewScanner(s)
	for scanner.Scan() {
		line := scanner.Text()
		var m map[string]interface{}
		if *jsonMode {
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				log.Fatalf("Failed to parse TinyG response: %q, err: %v", line, err)
			}
		}
		ch <- &response{line: line, m: m}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("Failed to read from serial port:", err)
	}
	log.Fatal("Serial port closed")
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

func send(s io.Writer, toCh <-chan string, respCh <-chan *response) {
	st := &state{x: math.NaN(), y: math.NaN(), z: math.NaN()}

	proc := func(resp *response) {
		fmt.Println("resp: ", resp)
		if resp.m["sr"] != nil {
			sr := resp.m["sr"].(map[string]interface{})
			if sr["mpox"] != nil {
				st.x = sr["mpox"].(float64)
			}
			if sr["mpoy"] != nil {
				st.y = sr["mpoy"].(float64)
			}
			if sr["mpoz"] != nil {
				st.z = sr["mpoz"].(float64)
			}

			fmt.Println("st: ", st)
		}
	}

	for {
		select {
		case cmd := <-toCh:
			if cmd == "" {
				continue
			}
			fmt.Println(cmd)
			if _, err := fmt.Fprintln(s, cmd); err != nil {
				log.Fatal("Failed to write to serial port: ", err)
			}
			// Waiting for TinyG to confirm it
			for {
				resp := <-respCh
				if resp == nil {
					// channel is closed
					return
				}
				proc(resp)
				if resp.m["r"] != nil {
					break
				}
			}
		case resp := <-respCh:
			if resp == nil {
				// channel is closed
				return
			}
			proc(resp)
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

	respCh := make(chan *response)
	toCh := make(chan string)
	go scan(s, respCh)
	go send(s, toCh, respCh)

	fmt.Fprintln(os.Stderr, "Please, enter g-code lines below:")
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
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
		cmd := gcode
		if *jsonMode {
			cmd = fmt.Sprintf(`{"gc":"%s"}`, cmd)
		}
		toCh <- cmd
	}
	if err := in.Err(); err != nil {
		log.Fatal("Failed to read from stdin: ", err)
	}
}
