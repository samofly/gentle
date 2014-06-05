// Package engine provides a Command/Message interface to talk to the CNC machine.
// One running instance of a Machine can accept commands and send messages to multiple clients.
// Currently, there's two types of clients: terminal and web.
package engine

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"sync"

	"github.com/samofly/gentle/tinyg"
)

// Machine represents a connected CNC machine. It can send commands to machines and distribute messages to the listeners.
type Machine interface {
	// Send sends a command to the machine. Send returns early.
	// It does not wait until the command is executed or even sent to the real machine.
	Send(cmd string)

	// Sub returns a channel to follow message from the machine.
	// Messages will be sent to the channel and discarded, if sending to the channel would block.
	// Thus, it's safe to not read from this channel.
	Sub() <-chan *Message
}

// Message is a message from the connected machine to the listeners.
type Message struct {
	// Raw is a raw output from the CNC machine. It's up to the listener to interpret this.
	// The primary goal is to enable manual control of the CNC machine by a human operator.
	Raw string `json:"raw"`
}

// New starts a new machine available over the provided connection.
// Usually, it would be an opened serial connection.
func New(conn io.ReadWriter, jsonMode bool) Machine {
	toCh := make(chan string)
	m := &machine{conn: conn, jsonMode: jsonMode, ps: newPubSub(), toCh: toCh}
	respCh := make(chan *tinyg.Response)
	go m.scan(respCh)

	go m.send(toCh, respCh)

	return m
}

// machine represents a connected CNC machine. It can receive commands and send messages.
type machine struct {
	conn     io.ReadWriter
	jsonMode bool
	ps       *pubsub
	toCh     chan<- string
}

func (m *machine) Send(cmd string) {
	m.toCh <- cmd
}

func (m *machine) Sub() <-chan *Message {
	return m.ps.Sub()
}

func (m *machine) scan(ch chan<- *tinyg.Response) {
	scanner := bufio.NewScanner(m.conn)
	for scanner.Scan() {
		line := scanner.Text()
		if !m.jsonMode {
			ch <- &tinyg.Response{Json: line}
			continue
		}
		r, err := tinyg.ParseResponse(line)
		if err != nil {
			// TODO(krasin): handle invalid response without a crash.
			// https://github.com/samofly/gentle/issues/1
			log.Fatalf("Failed to parse TinyG response:\n%s\nerr: %v", line, err)
		}
		ch <- r
	}
	if err := scanner.Err(); err != nil {
		// TODO(krasin): handle connection lost case better.
		// https://github.com/samofly/gentle/issues/1
		log.Fatal("Failed to read from machine connection:", err)
	}
	close(ch)
	log.Println("Machine connection closed")
}

func (m *machine) send(toCh <-chan string, respCh <-chan *tinyg.Response) {
	st := &state{x: math.NaN(), y: math.NaN(), z: math.NaN()}

	must := func(cmd string) {
		fmt.Println(cmd)
		if _, err := fmt.Fprintln(m.conn, cmd); err != nil {
			// TODO(krasin): don't crash if it's impossible to send a command to the machine
			// https://github.com/samofly/gentle/issues/1
			log.Fatal("Failed to write to the machine connection: ", err)
		}
	}

	proc := func(r *tinyg.Response) {
		m.ps.Pub(&Message{Raw: fmt.Sprintf("%v", r)})
		if r.Mpox != nil {
			st.x = *r.Mpox
		}
		if r.Mpoy != nil {
			st.y = *r.Mpoy
		}
		if r.Mpoz != nil {
			st.z = *r.Mpoz
		}
		m.ps.Pub(&Message{Raw: fmt.Sprintf("State: %v", st)})
	}

	for {
		select {
		case cmd := <-toCh:
			if cmd == "" {
				continue
			}
			must(cmd)
			if !m.jsonMode {
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
			if !m.jsonMode {
				m.ps.Pub(&Message{Raw: fmt.Sprintf("%s", resp.Json)})
				continue
			}
			proc(resp)
		}
	}
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

type pubsub struct {
	mu    sync.Mutex
	pubCh chan *Message
	ch    []chan<- *Message
}

func newPubSub() *pubsub {
	ps := &pubsub{pubCh: make(chan *Message)}
	go ps.run()
	return ps
}

func (ps *pubsub) Sub() <-chan *Message {
	ch := make(chan *Message)
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.ch = append(ps.ch, ch)
	return ch
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

func (ps *pubsub) Pub(msg *Message) {
	ps.pubCh <- msg
}
