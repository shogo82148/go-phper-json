package phperjson

import (
	"reflect"
	"testing"
)

type T struct {
	X string
	Y int
	Z int `json:"-"`
}

type unmarshalTest struct {
	in  string
	ptr interface{}
	out interface{}
	err error
}

var unmarshalTests = []unmarshalTest{
	// basic types
	{in: `true`, ptr: new(bool), out: true},
	{in: `1`, ptr: new(int), out: 1},
	{in: `1`, ptr: new(int8), out: int8(1)},
	{in: `1`, ptr: new(int16), out: int16(1)},
	{in: `1`, ptr: new(int32), out: int32(1)},
	{in: `1`, ptr: new(int64), out: int64(1)},
	{in: `1`, ptr: new(float32), out: float32(1)},
	{in: `1`, ptr: new(float64), out: float64(1)},

	{in: `1`, ptr: new(string), out: "1"},
}

func TestUnmarshal(t *testing.T) {
	for i, tt := range unmarshalTests {
		v := reflect.New(reflect.TypeOf(tt.ptr).Elem())
		err := Unmarshal([]byte(tt.in), v.Interface())
		if err != nil {
			t.Errorf("#%d: error %s", i, err)
			continue
		}

		if !reflect.DeepEqual(v.Elem().Interface(), tt.out) {
			t.Errorf("#%d: have %#+v, want %#+v", i, v.Elem().Interface(), tt.out)
		}
	}
}
