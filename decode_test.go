package phperjson

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type T struct {
	X string
	Y int
	Z int `json:"-"`
}

type U struct {
	Alphabet string `json:"alpha"`
}

type V struct {
	F1 interface{}
	F2 int32
	F3 Number
	F4 *VOuter
}

type VOuter struct {
	V V
}

// ifaceNumAsFloat64/ifaceNumAsNumber are used to test unmarshaling with and
// without UseNumber
var ifaceNumAsFloat64 = map[string]interface{}{
	"k1": float64(1),
	"k2": "s",
	"k3": []interface{}{float64(1), float64(2.0), float64(3e-3)},
	"k4": map[string]interface{}{"kk1": "s", "kk2": float64(2)},
}

var ifaceNumAsNumber = map[string]interface{}{
	"k1": Number("1"),
	"k2": "s",
	"k3": []interface{}{Number("1"), Number("2.0"), Number("3e-3")},
	"k4": map[string]interface{}{"kk1": "s", "kk2": Number("2")},
}

type tx struct {
	x int
}

type u8 uint8

// A type that can unmarshal itself.

type unmarshaler struct {
	T bool
}

func (u *unmarshaler) UnmarshalJSON(b []byte) error {
	*u = unmarshaler{true} // All we need to see that UnmarshalJSON is called.
	return nil
}

type ustruct struct {
	M unmarshaler
}

type unmarshalerText struct {
	A, B string
}

// needed for re-marshaling tests
func (u unmarshalerText) MarshalText() ([]byte, error) {
	return []byte(u.A + ":" + u.B), nil
}

func (u *unmarshalerText) UnmarshalText(b []byte) error {
	pos := bytes.IndexByte(b, ':')
	if pos == -1 {
		return errors.New("missing separator")
	}
	u.A, u.B = string(b[:pos]), string(b[pos+1:])
	return nil
}

var _ encoding.TextUnmarshaler = (*unmarshalerText)(nil)

type ustructText struct {
	M unmarshalerText
}

// u8marshal is an integer type that can marshal/unmarshal itself.
type u8marshal uint8

func (u8 u8marshal) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("u%d", u8)), nil
}

var errMissingU8Prefix = errors.New("missing 'u' prefix")

func (u8 *u8marshal) UnmarshalText(b []byte) error {
	if !bytes.HasPrefix(b, []byte{'u'}) {
		return errMissingU8Prefix
	}
	n, err := strconv.Atoi(string(b[1:]))
	if err != nil {
		return err
	}
	*u8 = u8marshal(n)
	return nil
}

var _ encoding.TextUnmarshaler = (*u8marshal)(nil)

var (
	um0, um1 unmarshaler // target2 of unmarshaling
	ump      = &um1
	umtrue   = unmarshaler{true}
	umslice  = []unmarshaler{{true}}
	umslicep = new([]unmarshaler)
	umstruct = ustruct{unmarshaler{true}}

	um0T, um1T   unmarshalerText // target2 of unmarshaling
	umpType      = &um1T
	umtrueXY     = unmarshalerText{"x", "y"}
	umsliceXY    = []unmarshalerText{{"x", "y"}}
	umslicepType = new([]unmarshalerText)
	umstructType = new(ustructText)
	umstructXY   = ustructText{unmarshalerText{"x", "y"}}

	ummapType = map[unmarshalerText]bool{}
	ummapXY   = map[unmarshalerText]bool{unmarshalerText{"x", "y"}: true}
)

// A type for test php array.
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
	{in: `{"F1":1,"F2":2,"F3":3}`, ptr: new(V), out: V{F1: float64(1), F2: int32(2), F3: Number("3")}},
	{in: `{"F1":1,"F2":2,"F3":3}`, ptr: new(V), out: V{F1: Number("1"), F2: int32(2), F3: Number("3")}, useNumber: true},
	{in: `{"k1":1,"k2":"s","k3":[1,2.0,3e-3],"k4":{"kk1":"s","kk2":2}}`, ptr: new(interface{}), out: ifaceNumAsFloat64},
	{in: `{"k1":1,"k2":"s","k3":[1,2.0,3e-3],"k4":{"kk1":"s","kk2":2}}`, ptr: new(interface{}), out: ifaceNumAsNumber, useNumber: true},

	// raw values with whitespace
	{in: "\n true ", ptr: new(bool), out: true},
	{in: "\t 1 ", ptr: new(int), out: 1},
	{in: "\r 1.2 ", ptr: new(float64), out: 1.2},
	{in: "\t -5 \n", ptr: new(int16), out: int16(-5)},
	{in: "\t \"a\\u1234\" \n", ptr: new(string), out: "a\u1234"},

	// Z has a "-" tag.
	{in: `{"Y": 1, "Z": 2}`, ptr: new(T), out: T{Y: 1}},
	{in: `{"Y": 1, "Z": 2}`, ptr: new(T), err: fmt.Errorf("json: unknown field \"Z\""), disallowUnknownFields: true},

	{in: `{"alpha": "abc", "alphabet": "xyz"}`, ptr: new(U), out: U{Alphabet: "abc"}},
	{in: `{"alpha": "abc", "alphabet": "xyz"}`, ptr: new(U), err: fmt.Errorf("json: unknown field \"alphabet\""), disallowUnknownFields: true},
	{in: `{"alpha": "abc"}`, ptr: new(U), out: U{Alphabet: "abc"}},
	{in: `{"alphabet": "xyz"}`, ptr: new(U), out: U{}},
	{in: `{"alphabet": "xyz"}`, ptr: new(U), err: fmt.Errorf("json: unknown field \"alphabet\""), disallowUnknownFields: true},

	// syntax errors
	// SyntaxError.msg is private field, so I can't test it.
	// {in: `{"X": "foo", "Y"}`, err: &SyntaxError{"invalid character '}' after object key", 17}},
	// {in: `[1, 2, 3+]`, err: &SyntaxError{"invalid character '+' after array element", 9}},
	// {in: `{"X":12x}`, err: &SyntaxError{"invalid character 'x' after object key:value pair", 8}, useNumber: true},

	// raw value errors
	// SyntaxError.msg is private field, so I can't test it.
	// {in: "\x01 42", err: &SyntaxError{"invalid character '\\x01' looking for beginning of value", 1}},
	// {in: " 42 \x01", err: &SyntaxError{"invalid character '\\x01' after top-level value", 5}},
	// {in: "\x01 true", err: &SyntaxError{"invalid character '\\x01' looking for beginning of value", 1}},
	// {in: " false \x01", err: &SyntaxError{"invalid character '\\x01' after top-level value", 8}},
	// {in: "\x01 1.2", err: &SyntaxError{"invalid character '\\x01' looking for beginning of value", 1}},
	// {in: " 3.4 \x01", err: &SyntaxError{"invalid character '\\x01' after top-level value", 6}},
	// {in: "\x01 \"string\"", err: &SyntaxError{"invalid character '\\x01' looking for beginning of value", 1}},
	// {in: " \"string\" \x01", err: &SyntaxError{"invalid character '\\x01' after top-level value", 11}},

	// array tests
	{in: `[1, 2, 3]`, ptr: new([3]int), out: [3]int{1, 2, 3}},
	{in: `[1, 2, 3]`, ptr: new([1]int), out: [1]int{1}},
	{in: `[1, 2, 3]`, ptr: new([5]int), out: [5]int{1, 2, 3, 0, 0}},

	// empty array to interface test
	{in: `[]`, ptr: new([]interface{}), out: []interface{}{}},
	{in: `null`, ptr: new([]interface{}), out: []interface{}(nil)},
	{in: `{"T":[]}`, ptr: new(map[string]interface{}), out: map[string]interface{}{"T": []interface{}{}}},
	{in: `{"T":null}`, ptr: new(map[string]interface{}), out: map[string]interface{}{"T": interface{}(nil)}},

	// composite tests
	{in: allValueIndent, ptr: new(All), out: allValue},
	{in: allValueCompact, ptr: new(All), out: allValue},
	{in: allValueIndent, ptr: new(*All), out: &allValue},
	{in: allValueCompact, ptr: new(*All), out: &allValue},
	{in: pallValueIndent, ptr: new(All), out: pallValue},
	{in: pallValueCompact, ptr: new(All), out: pallValue},
	{in: pallValueIndent, ptr: new(*All), out: &pallValue},
	{in: pallValueCompact, ptr: new(*All), out: &pallValue},

	// unmarshal interface test
	{in: `{"T":false}`, ptr: &um0, out: umtrue}, // use "false" so test will fail if custom unmarshaler is not called
	{in: `{"T":false}`, ptr: &ump, out: &umtrue},
	{in: `[{"T":false}]`, ptr: &umslice, out: umslice},
	{in: `[{"T":false}]`, ptr: &umslicep, out: &umslice},
	{in: `{"M":{"T":"x:y"}}`, ptr: &umstruct, out: umstruct},

	// UnmarshalText interface test
	{in: `"x:y"`, ptr: &um0T, out: umtrueXY},
	{in: `"x:y"`, ptr: &umpType, out: &umtrueXY},
	{in: `["x:y"]`, ptr: &umsliceXY, out: umsliceXY},
	{in: `["x:y"]`, ptr: &umslicepType, out: &umsliceXY},
	{in: `{"M":"x:y"}`, ptr: umstructType, out: umstructXY},

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

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func noSpace(c rune) rune {
	if isSpace(byte(c)) { //only used for ascii
		return -1
	}
	return c
}

type All struct {
	Bool    bool
	Int     int
	Int8    int8
	Int16   int16
	Int32   int32
	Int64   int64
	Uint    uint
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Uintptr uintptr
	Float32 float32
	Float64 float64

	Foo  string `json:"bar"`
	Foo2 string `json:"bar2,dummyopt"`

	IntStr     int64   `json:",string"`
	UintptrStr uintptr `json:",string"`

	PBool    *bool
	PInt     *int
	PInt8    *int8
	PInt16   *int16
	PInt32   *int32
	PInt64   *int64
	PUint    *uint
	PUint8   *uint8
	PUint16  *uint16
	PUint32  *uint32
	PUint64  *uint64
	PUintptr *uintptr
	PFloat32 *float32
	PFloat64 *float64

	String  string
	PString *string

	Map   map[string]Small
	MapP  map[string]*Small
	PMap  *map[string]Small
	PMapP *map[string]*Small

	EmptyMap map[string]Small
	NilMap   map[string]Small

	Slice   []Small
	SliceP  []*Small
	PSlice  *[]Small
	PSliceP *[]*Small

	EmptySlice []Small
	NilSlice   []Small

	StringSlice []string
	ByteSlice   []byte

	Small   Small
	PSmall  *Small
	PPSmall **Small

	Interface  interface{}
	PInterface *interface{}

	unexported int
}

type Small struct {
	Tag string
}

var allValue = All{
	Bool:       true,
	Int:        2,
	Int8:       3,
	Int16:      4,
	Int32:      5,
	Int64:      6,
	Uint:       7,
	Uint8:      8,
	Uint16:     9,
	Uint32:     10,
	Uint64:     11,
	Uintptr:    12,
	Float32:    14.1,
	Float64:    15.1,
	Foo:        "foo",
	Foo2:       "foo2",
	IntStr:     42,
	UintptrStr: 44,
	String:     "16",
	Map: map[string]Small{
		"17": {Tag: "tag17"},
		"18": {Tag: "tag18"},
	},
	MapP: map[string]*Small{
		"19": {Tag: "tag19"},
		"20": nil,
	},
	EmptyMap:    map[string]Small{},
	Slice:       []Small{{Tag: "tag20"}, {Tag: "tag21"}},
	SliceP:      []*Small{{Tag: "tag22"}, nil, {Tag: "tag23"}},
	EmptySlice:  []Small{},
	StringSlice: []string{"str24", "str25", "str26"},
	ByteSlice:   []byte{27, 28, 29},
	Small:       Small{Tag: "tag30"},
	PSmall:      &Small{Tag: "tag31"},
	Interface:   5.2,
}

var pallValue = All{
	PBool:      &allValue.Bool,
	PInt:       &allValue.Int,
	PInt8:      &allValue.Int8,
	PInt16:     &allValue.Int16,
	PInt32:     &allValue.Int32,
	PInt64:     &allValue.Int64,
	PUint:      &allValue.Uint,
	PUint8:     &allValue.Uint8,
	PUint16:    &allValue.Uint16,
	PUint32:    &allValue.Uint32,
	PUint64:    &allValue.Uint64,
	PUintptr:   &allValue.Uintptr,
	PFloat32:   &allValue.Float32,
	PFloat64:   &allValue.Float64,
	PString:    &allValue.String,
	PMap:       &allValue.Map,
	PMapP:      &allValue.MapP,
	PSlice:     &allValue.Slice,
	PSliceP:    &allValue.SliceP,
	PPSmall:    &allValue.PSmall,
	PInterface: &allValue.Interface,
}

var allValueIndent = `{
	"Bool": true,
	"Int": 2,
	"Int8": 3,
	"Int16": 4,
	"Int32": 5,
	"Int64": 6,
	"Uint": 7,
	"Uint8": 8,
	"Uint16": 9,
	"Uint32": 10,
	"Uint64": 11,
	"Uintptr": 12,
	"Float32": 14.1,
	"Float64": 15.1,
	"bar": "foo",
	"bar2": "foo2",
	"IntStr": "42",
	"UintptrStr": "44",
	"PBool": null,
	"PInt": null,
	"PInt8": null,
	"PInt16": null,
	"PInt32": null,
	"PInt64": null,
	"PUint": null,
	"PUint8": null,
	"PUint16": null,
	"PUint32": null,
	"PUint64": null,
	"PUintptr": null,
	"PFloat32": null,
	"PFloat64": null,
	"String": "16",
	"PString": null,
	"Map": {
		"17": {
			"Tag": "tag17"
		},
		"18": {
			"Tag": "tag18"
		}
	},
	"MapP": {
		"19": {
			"Tag": "tag19"
		},
		"20": null
	},
	"PMap": null,
	"PMapP": null,
	"EmptyMap": {},
	"NilMap": null,
	"Slice": [
		{
			"Tag": "tag20"
		},
		{
			"Tag": "tag21"
		}
	],
	"SliceP": [
		{
			"Tag": "tag22"
		},
		null,
		{
			"Tag": "tag23"
		}
	],
	"PSlice": null,
	"PSliceP": null,
	"EmptySlice": [],
	"NilSlice": null,
	"StringSlice": [
		"str24",
		"str25",
		"str26"
	],
	"ByteSlice": "Gxwd",
	"Small": {
		"Tag": "tag30"
	},
	"PSmall": {
		"Tag": "tag31"
	},
	"PPSmall": null,
	"Interface": 5.2,
	"PInterface": null
}`

var allValueCompact = strings.Map(noSpace, allValueIndent)

var pallValueIndent = `{
	"Bool": false,
	"Int": 0,
	"Int8": 0,
	"Int16": 0,
	"Int32": 0,
	"Int64": 0,
	"Uint": 0,
	"Uint8": 0,
	"Uint16": 0,
	"Uint32": 0,
	"Uint64": 0,
	"Uintptr": 0,
	"Float32": 0,
	"Float64": 0,
	"bar": "",
	"bar2": "",
        "IntStr": "0",
	"UintptrStr": "0",
	"PBool": true,
	"PInt": 2,
	"PInt8": 3,
	"PInt16": 4,
	"PInt32": 5,
	"PInt64": 6,
	"PUint": 7,
	"PUint8": 8,
	"PUint16": 9,
	"PUint32": 10,
	"PUint64": 11,
	"PUintptr": 12,
	"PFloat32": 14.1,
	"PFloat64": 15.1,
	"String": "",
	"PString": "16",
	"Map": null,
	"MapP": null,
	"PMap": {
		"17": {
			"Tag": "tag17"
		},
		"18": {
			"Tag": "tag18"
		}
	},
	"PMapP": {
		"19": {
			"Tag": "tag19"
		},
		"20": null
	},
	"EmptyMap": null,
	"NilMap": null,
	"Slice": null,
	"SliceP": null,
	"PSlice": [
		{
			"Tag": "tag20"
		},
		{
			"Tag": "tag21"
		}
	],
	"PSliceP": [
		{
			"Tag": "tag22"
		},
		null,
		{
			"Tag": "tag23"
		}
	],
	"EmptySlice": null,
	"NilSlice": null,
	"StringSlice": null,
	"ByteSlice": null,
	"Small": {
		"Tag": ""
	},
	"PSmall": null,
	"PPSmall": {
		"Tag": "tag31"
	},
	"Interface": null,
	"PInterface": 5.2
}`

var pallValueCompact = strings.Map(noSpace, pallValueIndent)
