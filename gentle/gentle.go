// gentle is a simple g-code sender compatible with TinyG.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/samofly/sers"
)

var ttyDev = flag.String("dev", "/dev/ttyUSB0", "Serial device to open")

func main() {
	if *ttyDev == "" {
		log.Fatal("-dev (serial device) is not specified.")
	}
	s, err := sers.Open("/dev/ttyUSB0")
	if err != nil {
		log.Fatalf("Could not open serial port at %s: %v", *ttyDev, err)
	}
	defer s.Close()
	log.Print("Port opened at ", *ttyDev)
	if err = s.SetMode(115200, 8, 0, 1, 0); err != nil {
		log.Fatal("Failed to set mode:", err)
	}
	log.Printf("Mode has been set")

	if _, err = s.Write([]byte("{\"sr\":\"\"}\n")); err != nil {
		log.Fatal("Failed to write to serial port:", err)
	}
	log.Print("Command sent, waiting for reply...")

	for {
		buf := make([]byte, 100)
		n, err := s.Read(buf)
		if err != nil {
			log.Fatal("Failed to read from serial port:", err)
		}
		fmt.Printf(string(buf[:n]))
		time.Sleep(20 * time.Millisecond)
	}
}
