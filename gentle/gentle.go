// gentle is a simple g-code sender compatible with TinyG.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/samofly/sers"
)

var ttyDev = flag.String("dev", "/dev/ttyUSB0", "Serial device to open")

func scan(s sers.SerialPort) {
	scanner := bufio.NewScanner(s)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("Failed to read from serial port:", err)
	}
	log.Fatal("Serial port closed")
}

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

	go scan(s)

	fmt.Fprintln(os.Stderr, "Please, enter g-code lines below:")
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		if _, err := fmt.Fprintln(s, in.Text()); err != nil {
			log.Fatal("Failed to write to serial port:", err)
		}
	}
	if err := in.Err(); err != nil {
		log.Fatal("Failed to read from stdin:", err)
	}
}
