package phperjson

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

type T struct {
	X string
	Y int
	Z int `json:"-"`
}

type tx struct {
	x int
}

type u8 uint8

type phpArray struct {
	First string `json:"0"`
}

type unmarshalTest struct {
	in                    string
	ptr                   interface{}
	out                   interface{}
	err                   error
	useNumber             bool
	disallowUnknownFields bool
}

var unmarshalTests = []unmarshalTest{
	// test compatiblity with encoding/json
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
	{in: `{"X": [1,2,3], "Y": 4}`, ptr: new(T), out: T{Y: 4}, err: &UnmarshalTypeError{Value: "array", Type: reflect.TypeOf(""), Struct: "T", Field: "X"}},
	{in: `{"x": 1}`, ptr: new(tx), out: tx{}},
	{in: `{"x": 1}`, ptr: new(tx), err: fmt.Errorf("json: unknown field \"x\""), disallowUnknownFields: true},
	// {in: `{"F1":1,"F2":2,"F3":3}`, ptr: new(V), out: V{F1: float64(1), F2: int32(2), F3: Number("3")}},
	// {in: `{"F1":1,"F2":2,"F3":3}`, ptr: new(V), out: V{F1: Number("1"), F2: int32(2), F3: Number("3")}, useNumber: true},
	// {in: `{"k1":1,"k2":"s","k3":[1,2.0,3e-3],"k4":{"kk1":"s","kk2":2}}`, ptr: new(interface{}), out: ifaceNumAsFloat64},
	// {in: `{"k1":1,"k2":"s","k3":[1,2.0,3e-3],"k4":{"kk1":"s","kk2":2}}`, ptr: new(interface{}), out: ifaceNumAsNumber, useNumber: true},

	// raw values with whitespace
	{in: "\n true ", ptr: new(bool), out: true},
	{in: "\t 1 ", ptr: new(int), out: 1},
	{in: "\r 1.2 ", ptr: new(float64), out: 1.2},
	{in: "\t -5 \n", ptr: new(int16), out: int16(-5)},
	{in: "\t \"a\\u1234\" \n", ptr: new(string), out: "a\u1234"},

	// Z has a "-" tag.
	{in: `{"Y": 1, "Z": 2}`, ptr: new(T), out: T{Y: 1}},
	{in: `{"Y": 1, "Z": 2}`, ptr: new(T), err: fmt.Errorf("json: unknown field \"Z\""), disallowUnknownFields: true},

	// array tests
	{in: `[1, 2, 3]`, ptr: new([3]int), out: [3]int{1, 2, 3}},
	{in: `[1, 2, 3]`, ptr: new([1]int), out: [1]int{1}},
	{in: `[1, 2, 3]`, ptr: new([5]int), out: [5]int{1, 2, 3, 0, 0}},

	// empty array to interface test
	{in: `[]`, ptr: new([]interface{}), out: []interface{}{}},
	{in: `null`, ptr: new([]interface{}), out: []interface{}(nil)},
	{in: `{"T":[]}`, ptr: new(map[string]interface{}), out: map[string]interface{}{"T": []interface{}{}}},
	{in: `{"T":null}`, ptr: new(map[string]interface{}), out: map[string]interface{}{"T": interface{}(nil)}},

	// integer-keyed map test
	{
		in:  `{"-1":"a","0":"b","1":"c"}`,
		ptr: new(map[int]string),
		out: map[int]string{-1: "a", 0: "b", 1: "c"},
	},
	{
		in:  `{"0":"a","10":"c","9":"b"}`,
		ptr: new(map[u8]string),
		out: map[u8]string{0: "a", 9: "b", 10: "c"},
	},
	{
		in:  `{"-9223372036854775808":"min","9223372036854775807":"max"}`,
		ptr: new(map[int64]string),
		out: map[int64]string{math.MinInt64: "min", math.MaxInt64: "max"},
	},
	{
		in:  `{"18446744073709551615":"max"}`,
		ptr: new(map[uint64]string),
		out: map[uint64]string{math.MaxUint64: "max"},
	},
	{
		in:  `{"0":false,"10":true}`,
		ptr: new(map[uintptr]bool),
		out: map[uintptr]bool{0: false, 10: true},
	},

	// PHP flavored
	// convert to boolean
	{in: `false`, ptr: new(bool), out: false},
	{in: `0`, ptr: new(bool), out: false},
	{in: `1`, ptr: new(bool), out: true},
	{in: `-2`, ptr: new(bool), out: true},
	{in: `0.0`, ptr: new(bool), out: false},
	{in: `2.3e5`, ptr: new(bool), out: true},
	{in: `""`, ptr: new(bool), out: false},
	{in: `"0"`, ptr: new(bool), out: false},
	{in: `"foo"`, ptr: new(bool), out: true},
	{in: `[]`, ptr: new(bool), out: false},
	{in: `{}`, ptr: new(bool), out: false},
	{in: `[12]`, ptr: new(bool), out: true},
	{in: `{"foo":12}`, ptr: new(bool), out: true},
	{in: `null`, ptr: new(bool), out: false},
	{in: `"false"`, ptr: new(bool), out: true},

	// convert to string
	{in: `1`, ptr: new(string), out: "1"},
	{in: `1.2`, ptr: new(string), out: "1.2"},
	{in: `-5`, ptr: new(string), out: "-5"},
	{in: `true`, ptr: new(string), out: "1"},
	{in: `false`, ptr: new(string), out: ""},
	{in: `null`, ptr: new(string), out: ""},

	// convert to integer
	{in: `"1"`, ptr: new(int), out: 1},
	{in: `"1.1"`, ptr: new(int), out: 1},
	{in: `1.1`, ptr: new(int), out: 1},
	{in: `true`, ptr: new(int), out: 1},
	{in: `false`, ptr: new(int), out: 0},
	{in: `""`, ptr: new(int), out: int(0)},

	// convert to floats
	{in: `"1"`, ptr: new(float64), out: float64(1)},
	{in: `"1.1"`, ptr: new(float64), out: 1.1},
	{in: `1.1`, ptr: new(float64), out: 1.1},
	{in: `true`, ptr: new(float64), out: float64(1)},
	{in: `false`, ptr: new(float64), out: float64(0)},
	{in: `""`, ptr: new(float64), out: float64(0)},

	// convert to array
	{in: `true`, ptr: new([]bool), out: []bool{true}},
	{in: `1`, ptr: new([]int), out: []int{1}},
	{in: `1.1`, ptr: new([]float64), out: []float64{1.1}},
	{in: `"foo"`, ptr: new([]string), out: []string{"foo"}},
	{in: `{}`, ptr: new([]interface{}), out: []interface{}{}},
	{in: `{"1":1}`, ptr: new([]int), out: []int{0, 1}},
	{in: `{"1":1,"3":3}`, ptr: new([3]int), out: [3]int{0, 1, 0}},
	{in: `true`, ptr: new(map[string]bool), out: map[string]bool{"0": true}},
	{in: `1`, ptr: new(map[string]int), out: map[string]int{"0": 1}},
	{in: `1.1`, ptr: new(map[string]float64), out: map[string]float64{"0": 1.1}},
	{in: `"foo"`, ptr: new(map[string]string), out: map[string]string{"0": "foo"}},
	{in: `true`, ptr: new(phpArray), out: phpArray{First: "1"}},
	{in: `1`, ptr: new(phpArray), out: phpArray{First: "1"}},
	{in: `1.1`, ptr: new(phpArray), out: phpArray{First: "1.1"}},
	{in: `"foo"`, ptr: new(phpArray), out: phpArray{First: "foo"}},
	{in: `[]`, ptr: new(map[string]interface{}), out: map[string]interface{}{}},
	{in: `[true]`, ptr: new(map[string]bool), out: map[string]bool{"0": true}},
	{in: `[1]`, ptr: new(map[string]int), out: map[string]int{"0": 1}},
	{in: `[1.1]`, ptr: new(map[string]float64), out: map[string]float64{"0": 1.1}},
	{in: `["foo"]`, ptr: new(map[string]string), out: map[string]string{"0": "foo"}},
	{in: `[true]`, ptr: new(phpArray), out: phpArray{First: "1"}},
	{in: `[1]`, ptr: new(phpArray), out: phpArray{First: "1"}},
	{in: `[1.1]`, ptr: new(phpArray), out: phpArray{First: "1.1"}},
	{in: `["foo"]`, ptr: new(phpArray), out: phpArray{First: "foo"}},
}

func TestUnmarshal(t *testing.T) {
	for i, tt := range unmarshalTests {
		dec := NewDecoder(strings.NewReader(tt.in))
		if tt.useNumber {
			dec.UseNumber()
		}
		if tt.disallowUnknownFields {
			dec.DisallowUnknownFields()
		}
		v := reflect.New(reflect.TypeOf(tt.ptr).Elem())
		if err := dec.Decode(v.Interface()); !reflect.DeepEqual(err, tt.err) {
			t.Errorf("#%d: %v, want %v", i, err, tt.err)
			continue
		} else if err != nil {
			continue
		}
		if !reflect.DeepEqual(v.Elem().Interface(), tt.out) {
			t.Errorf("#%d: have %#+v, want %#+v", i, v.Elem().Interface(), tt.out)
		}
	}
}
