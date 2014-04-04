// gentle is a simple g-code sender compatible with TinyG.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"
	"unsafe"
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

func ioctl(fd uintptr, req uint, arg unsafe.Pointer) (err syscall.Errno) {
	_, _, err = syscall.RawSyscall(syscall.SYS_IOCTL, fd, uintptr(req), uintptr(arg))
	return
}

const TCSETSF = 0x5404
const NCCS = 32

type termios struct {
	c_ifflag uint32     /* input mode flags */
	c_oflag  uint32     /* output mode flags */
	c_cflags uint32     /* control mode flags */
	c_lflag  uint32     /* local mode flags */
	c_line   byte       /* line discipline */
	c_cc     [NCCS]byte /* control characters */
	c_ispeed uint32     /* input speed */
	c_ospeed uint32     /* output speed */
}

const O_NOCTTY = 0400 /* Not fcntl.  */
const O_NONBLOCK = 00004000

func main() {
	flag.Parse()

	if *ttyDev == "" {
		log.Fatal("-dev (serial device) is not specified.")
	}
	//s, err := sers.Open(*ttyDev)
	s, err := os.OpenFile(*ttyDev, os.O_RDWR|O_NOCTTY /*|O_NONBLOCK*/, 0)
	if err != nil {
		log.Fatalf("Could not open serial port at %s: %v", *ttyDev, err)
	}
	defer s.Close()

	// Now, we need to set the parameters.
	// Currently, just call naked ioctl with pre-baked params.
	arg := &termios{
		c_cflags: 0x1cb2,
		c_cc:     [NCCS]byte{0x03, 0x1c, 0x7f, 0x15, 0x01, 0, 0x01, 0, 0x11, 0x13, 0x1a, 0, 0x12, 0x0f, 0x17, 0x16, 0, 0, 0},
	}
	//arg := &[128]byte{0, 0, 0, 0, 0, 0, 0, 0, 0xb2, 0x14, 0, 0, 0, 0, 0, 0, 0,
	//	0x03, 0x1c, 0x7f, 0x15, 0x01, 0, 0x01, 0, 0x11, 0x13, 0x1a, 0, 0x12, 0x0f, 0x17, 0x16, 0, 0, 0 /* c_cc */}
	if errno := ioctl(s.Fd(), TCSETSF, unsafe.Pointer(arg)); errno != 0 {
		log.Fatal("iotcl(TCSETSF) failed, errno=", errno)
	}

	log.Print("Port opened at ", *ttyDev)
	//	if err = s.SetMode(*baudRate, 8, 0, 1, 0); err != nil {
	//		log.Fatal("Failed to set mode: ", err)
	//	}
	log.Printf("Mode has been set")

	respCh := make(chan *response)
	go scan(s, respCh)

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
		fmt.Println(cmd)
		if _, err := fmt.Fprintln(s, cmd); err != nil {
			log.Fatal("Failed to write to serial port: ", err)
		}
		// Waiting for TinyG to confirm it
		for {
			resp := <-respCh
			fmt.Println("resp:", resp)
			if resp.m["r"] != nil {
				break
			}
		}
	}
	if err := in.Err(); err != nil {
		log.Fatal("Failed to read from stdin: ", err)
	}
}
