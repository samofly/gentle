package tinyg

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Response contains all possible values which may be reported by TinyG.
// A field is only non-null, if it's set in the json report. For example,
// when TinyG is moving by X axis, it will only report X values, but not
// Y or Z, since they remain the same.
type Response struct {
	// Original json response
	Json string `json:"-"`

	// Mpox is the absolute X coordinate
	Mpox *float64

	// Ofsz is the X axis offset.
	Ofsx *float64

	// Mpoy is the absolute Y coordinate
	Mpoy *float64

	// Ofsy is the Y axis offset
	Ofsy *float64

	// Mpoz is the absolute Z coordinate
	Mpoz *float64

	// Ofsz is the Z axis offset
	Ofsz *float64

	// Footer is a part of response to a command.
	// See https://github.com/synthetos/TinyG/wiki/JSON-Operation for more details.
	Footer []int `json:"-"`
}

func (r *Response) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Json: %s", r.Json)

	was := false
	mb := func(name string, val *float64) {
		if val == nil {
			return
		}
		if !was {
			was = true
			fmt.Fprintln(&buf)
		}
		fmt.Fprintf(&buf, "%s: %.3f  ", name, *val)
	}
	mb("Mpox", r.Mpox)
	mb("Ofsx", r.Ofsx)
	mb("Mpoy", r.Mpoy)
	mb("Ofsy", r.Ofsy)
	mb("Mpoz", r.Mpoz)
	mb("Ofsz", r.Ofsz)

	return buf.String()
}

// ParseResponse parses json response from TinyG.
func ParseResponse(resp string) (*Response, error) {
	var b body
	if err := json.Unmarshal([]byte(resp), &b); err != nil {
		return nil, err
	}
	var res *Response
	switch {
	case b.SR != nil:
		res = b.SR
	case b.R != nil && b.R.SR != nil:
		res = b.R.SR
	default:
		res = new(Response)
	}
	res.Footer = b.F
	res.Json = resp
	return res, nil
}

type body struct {
	SR *Response
	R  *resp
	F  []int
}

type resp struct {
	SR *Response
}
