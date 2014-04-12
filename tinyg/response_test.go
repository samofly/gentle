package tinyg

import (
	"fmt"
	"reflect"
	"testing"
)

func f64(v float64) *float64 { return &v }

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name string
		json string
		resp *Response
		err  error
	}{
		{
			name: "full status report",
			json: `{"r":{"sr":{"mpox":0.000,"mpoy":0.000,"mpoz":0.000,"mpoa":0.000,"ofsx":0.000,"ofsy":0.000,"ofsz":-60.310,"ofsa":0.000,"unit":1,"stat":3,"coor":2,"momo":0,"dist":0,"home":1,"hold":0,"macs":3,"cycs":0,"mots":0,"plan":0}},"f":[1,0,10,9925]}`,
			resp: &Response{Mpox: f64(0), Mpoy: f64(0), Mpoz: f64(0), Footer: []int{1, 0, 10, 9925}},
		},
		{
			name: "moving X report",
			json: `{"sr":{"mpox":0.000,"stat":5,"macs":5,"cycs":1,"mots":1}}`,
			resp: &Response{Mpox: f64(0)},
		},
		{
			name: "just qr",
			json: `{"qr":27}`,
			resp: &Response{},
		},
	}
	for _, tt := range tests {
		resp, err := ParseResponse(tt.json)
		if fmt.Sprintf("%v", err) != fmt.Sprintf("%v", tt.err) {
			t.Errorf("%q: ParseResponse(%s) failed: %v, expected: %v", tt.name, tt.json, err, tt.err)
			continue
		}
		if err != nil {
			continue
		}
		tt.resp.Json = tt.json
		if !reflect.DeepEqual(resp, tt.resp) {
			t.Errorf("%q: ParseResponse(%s),\ngot:  %+v\n want: %+v", tt.name, tt.json, resp, tt.resp)
			continue
		}
	}
}
