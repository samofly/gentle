package main

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
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

func TestJsonScanner(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []map[string]interface{}
	}{
		{
			name: "empty",
		},
		{
			name: "single message",
			in:   `{"a":"b"}`,
			want: []map[string]interface{}{
				map[string]interface{}{"a": "b"},
			},
		},
		{
			name: "two messages",
			in:   `{"a":"b"}{"c":"d"}`,
			want: []map[string]interface{}{
				map[string]interface{}{"a": "b"},
				map[string]interface{}{"c": "d"},
			},
		},
	}
	for _, tt := range tests {
		r := newJsonScanner(bytes.NewBufferString(tt.in))
		ok := true
		for _, want := range tt.want {
			var cur map[string]interface{}
			if err := r.Scan(&cur); err != nil {
				t.Errorf("%q: Failed to read json: %v", tt.name, err)
				ok = false
				break
			}
			if !reflect.DeepEqual(cur, want) {
				t.Errorf("%q: unexpected message: %v, want: %v", cur, want)
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		var m map[string]interface{}
		if err := r.Scan(&m); err != io.EOF {
			t.Errorf("%q: wanted io.EOF, got: %v", err)
			continue
		}
	}
}
