package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestJsonSplit(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		atEOF bool
		adv   int
		tok   string
		err   error
	}{
		{
			name:  "small well-formed json",
			in:    `{"a":3}`,
			atEOF: true,
			adv:   7,
			tok:   `{"a":3}`,
		},
		{
			name:  "two json objects",
			in:    `{"a":"b"}{"z":3"}`,
			atEOF: true,
			adv:   9,
			tok:   `{"a":"b"}`,
		},
		{
			name:  "unexpected EOF",
			in:    `{"a",`,
			atEOF: true,
			err:   io.ErrUnexpectedEOF,
		},
		{
			name: "want more",
			in:   `{"a",`,
		},
		{
			name: "closing brace within string",
			in:   `{"a":"}"}`,
			adv:  9,
			tok:  `{"a":"}"}`,
		},
		{
			name: "escaped quotes",
			in:   `{"a":"\""}`,
			adv:  10,
			tok:  `{"a":"\""}`,
		},
		{
			name: "escaped backslash",
			in:   `{"a":"\\"}`,
			adv:  10,
			tok:  `{"a":"\\"}`,
		},
		{
			name: "too many closing braces",
			in:   `}`,
			err:  errUnexpectedClosingBrace,
		},
	}
	for _, tt := range tests {
		var js jsonSplitter
		adv, tok, err := js.Split([]byte(tt.in), tt.atEOF)
		if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.err) {
			t.Errorf("%q: js.Split(%s) = %d, _, %v. Want err: %v", tt.name, tt.in, adv, err, tt.err)
			continue
		}
		if err != nil {
			continue
		}
		if adv != tt.adv {
			t.Errorf("%q: js.Split(%s) = %d, _, _. Want adv: %d", tt.name, tt.in, adv, tt.adv)
			continue
		}
		if !bytes.Equal(tok, []byte(tt.tok)) {
			t.Errorf("%q: js.Split(%s) = _, %s, _. Want tok: %s", tt.name, tt.in, string(tok), tt.tok)
			continue
		}
	}
}
