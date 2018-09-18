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
