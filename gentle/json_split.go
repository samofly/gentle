package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var errUnexpectedClosingBrace = errors.New("unexpected '}'. Input is not a valid stream of json objects")

type jsonScanner struct {
	s *bufio.Scanner
}

func newJsonScanner(r io.Reader) *jsonScanner {
	s := bufio.NewScanner(r)
	var js jsonSplitter
	s.Split(js.Split)
	return &jsonScanner{s: s}
}

// ReadJson reads the next message from the Reader.
func (js *jsonScanner) Scan(v interface{}) error {
	if !js.s.Scan() {
		if js.s.Err() != nil {
			return js.s.Err()
		}
		return io.EOF
	}
	return json.Unmarshal(js.s.Bytes(), v)
}

// jsonSplitter splits incoming data into json messages.
// It's intended to be used with bufio.Scanner.
type jsonSplitter struct {
	nest  int
	last  byte
	isStr bool
	skip  int
}

// Split implements bufio.SplitFunc
func (js *jsonSplitter) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if js.skip > len(data) {
		return 0, nil, fmt.Errorf("jsonSplitter.Split: %d = skip > len(data) = %d.", js.skip, len(data))
	}
	orig := data
	data = data[js.skip:]

	for i, b := range data {
		if js.isStr && js.last == '\\' {
			js.last = 0
			continue
		}
		if js.isStr && b == '"' {
			js.last = 0
			js.isStr = false
			continue
		}
		js.last = b
		if js.isStr {
			continue
		}
		if b == '"' {
			js.isStr = true
			continue
		}
		if b == '{' {
			js.nest++
			continue
		}
		if b == '}' {
			js.nest--
			if js.nest < 0 {
				return 0, nil, errUnexpectedClosingBrace
			}
			if js.nest == 0 {
				advance = js.skip + i + 1
				token = orig[:advance]
				js.skip = 0
				return
			}
			continue
		}
	}

	// Matching curly braces not found yet.
	if atEOF {
		return 0, nil, io.ErrUnexpectedEOF
	}
	js.skip += len(data)
	return 0, nil, nil
}
