package phperjson

import (
	"reflect"
	"strings"
	"testing"
)

type T struct {
	X string
	Y int
	Z int `json:"-"`
}

type unmarshalTest struct {
	in        string
	ptr       interface{}
	out       interface{}
	err       error
	useNumber bool
}

var unmarshalTests = []unmarshalTest{
	// basic types
	{in: `true`, ptr: new(bool), out: true},
	{in: `1`, ptr: new(int), out: 1},
	{in: `1.2`, ptr: new(float64), out: 1.2},
	{in: `-5`, ptr: new(int16), out: int16(-5)},
	{in: `2`, ptr: new(Number), out: Number("2"), useNumber: true},
	{in: `2`, ptr: new(Number), out: Number("2")},
	{in: `2`, ptr: new(interface{}), out: float64(2.0)},
	{in: `2`, ptr: new(interface{}), out: Number("2"), useNumber: true},
	{in: `"a\u1234"`, ptr: new(string), out: "a\u1234"},
	{in: `"http:\/\/"`, ptr: new(string), out: "http://"},
	{in: `"g-clef: \uD834\uDD1E"`, ptr: new(string), out: "g-clef: \U0001D11E"},
	{in: `"invalid: \uD834x\uDD1E"`, ptr: new(string), out: "invalid: \uFFFDx\uFFFD"},
	{in: "null", ptr: new(interface{}), out: nil},

	{in: `1`, ptr: new(string), out: "1"},
}

func TestUnmarshal(t *testing.T) {
	for i, tt := range unmarshalTests {
		dec := NewDecoder(strings.NewReader(tt.in))
		if tt.useNumber {
			dec.UseNumber()
		}
		v := reflect.New(reflect.TypeOf(tt.ptr).Elem())
		err := dec.Decode(v.Interface())
		if err != nil {
			t.Errorf("#%d: error %s", i, err)
			continue
		}

		if !reflect.DeepEqual(v.Elem().Interface(), tt.out) {
			t.Errorf("#%d: have %#+v, want %#+v", i, v.Elem().Interface(), tt.out)
		}
	}
}
