package phperjson

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"image"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
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

// Test data structures for anonymous fields.

type Point struct {
	Z int
}

type Top struct {
	Level0 int
	Embed0
	*Embed0a
	*Embed0b `json:"e,omitempty"` // treated as named
	Embed0c  `json:"-"`           // ignored
	Loop
	Embed0p // has Point with X, Y, used
	Embed0q // has Point with Z, used
	embed   // contains exported field
}

type Embed0 struct {
	Level1a int // overridden by Embed0a's Level1a with json tag
	Level1b int // used because Embed0a's Level1b is renamed
	Level1c int // used because Embed0a's Level1c is ignored
	Level1d int // annihilated by Embed0a's Level1d
	Level1e int `json:"x"` // annihilated by Embed0a.Level1e
}

type Embed0a struct {
	Level1a int `json:"Level1a,omitempty"`
	Level1b int `json:"LEVEL1B,omitempty"`
	Level1c int `json:"-"`
	Level1d int // annihilated by Embed0's Level1d
	Level1f int `json:"x"` // annihilated by Embed0's Level1e
}

type Embed0b Embed0

type Embed0c Embed0

type Embed0p struct {
	image.Point
}

type Embed0q struct {
	Point
}

type embed struct {
	Q int
}

type Loop struct {
	Loop1 int `json:",omitempty"`
	Loop2 int `json:",omitempty"`
	*Loop
}

// From reflect test:
// The X in S6 and S7 annihilate, but they also block the X in S8.S9.
type S5 struct {
	S6
	S7
	S8
}

type S6 struct {
	X int
}

type S7 S6

type S8 struct {
	S9
}

type S9 struct {
	X int
	Y int
}

// From reflect test:
// The X in S11.S6 and S12.S6 annihilate, but they also block the X in S13.S8.S9.
type S10 struct {
	S11
	S12
	S13
}

type S11 struct {
	S6
}

type S12 struct {
	S6
}

type S13 struct {
	S8
}

type Ambig struct {
	// Given "hello", the first match should win.
	First  int `json:"HELLO"`
	Second int `json:"Hello"`
}

type XYZ struct {
	X interface{}
	Y interface{}
	Z interface{}
}

func sliceAddr(x []int) *[]int                 { return &x }
func mapAddr(x map[string]int) *map[string]int { return &x }

type byteWithMarshalJSON byte

func (b byteWithMarshalJSON) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"Z%.2x"`, byte(b))), nil
}

func (b *byteWithMarshalJSON) UnmarshalJSON(data []byte) error {
	if len(data) != 5 || data[0] != '"' || data[1] != 'Z' || data[4] != '"' {
		return fmt.Errorf("bad quoted string")
	}
	i, err := strconv.ParseInt(string(data[2:4]), 16, 8)
	if err != nil {
		return fmt.Errorf("bad hex")
	}
	*b = byteWithMarshalJSON(i)
	return nil
}

type byteWithPtrMarshalJSON byte

func (b *byteWithPtrMarshalJSON) MarshalJSON() ([]byte, error) {
	return byteWithMarshalJSON(*b).MarshalJSON()
}

func (b *byteWithPtrMarshalJSON) UnmarshalJSON(data []byte) error {
	return (*byteWithMarshalJSON)(b).UnmarshalJSON(data)
}

type byteWithMarshalText byte

func (b byteWithMarshalText) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf(`Z%.2x`, byte(b))), nil
}

func (b *byteWithMarshalText) UnmarshalText(data []byte) error {
	if len(data) != 3 || data[0] != 'Z' {
		return fmt.Errorf("bad quoted string")
	}
	i, err := strconv.ParseInt(string(data[1:3]), 16, 8)
	if err != nil {
		return fmt.Errorf("bad hex")
	}
	*b = byteWithMarshalText(i)
	return nil
}

type byteWithPtrMarshalText byte

func (b *byteWithPtrMarshalText) MarshalText() ([]byte, error) {
	return byteWithMarshalText(*b).MarshalText()
}

func (b *byteWithPtrMarshalText) UnmarshalText(data []byte) error {
	return (*byteWithMarshalText)(b).UnmarshalText(data)
}

type intWithMarshalJSON int

func (b intWithMarshalJSON) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"Z%.2x"`, int(b))), nil
}

func (b *intWithMarshalJSON) UnmarshalJSON(data []byte) error {
	if len(data) != 5 || data[0] != '"' || data[1] != 'Z' || data[4] != '"' {
		return fmt.Errorf("bad quoted string")
	}
	i, err := strconv.ParseInt(string(data[2:4]), 16, 8)
	if err != nil {
		return fmt.Errorf("bad hex")
	}
	*b = intWithMarshalJSON(i)
	return nil
}

type intWithPtrMarshalJSON int

func (b *intWithPtrMarshalJSON) MarshalJSON() ([]byte, error) {
	return intWithMarshalJSON(*b).MarshalJSON()
}

func (b *intWithPtrMarshalJSON) UnmarshalJSON(data []byte) error {
	return (*intWithMarshalJSON)(b).UnmarshalJSON(data)
}

type intWithMarshalText int

func (b intWithMarshalText) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf(`Z%.2x`, int(b))), nil
}

func (b *intWithMarshalText) UnmarshalText(data []byte) error {
	if len(data) != 3 || data[0] != 'Z' {
		return fmt.Errorf("bad quoted string")
	}
	i, err := strconv.ParseInt(string(data[1:3]), 16, 8)
	if err != nil {
		return fmt.Errorf("bad hex")
	}
	*b = intWithMarshalText(i)
	return nil
}

type intWithPtrMarshalText int

func (b *intWithPtrMarshalText) MarshalText() ([]byte, error) {
	return intWithMarshalText(*b).MarshalText()
}

func (b *intWithPtrMarshalText) UnmarshalText(data []byte) error {
	return (*intWithMarshalText)(b).UnmarshalText(data)
}

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
	golden                bool
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

	// Check that MarshalText and UnmarshalText take precedence
	// over default integer handling in map keys.
	{
		in:  `{"u2":4}`,
		ptr: new(map[u8marshal]int),
		out: map[u8marshal]int{2: 4},
	},
	{
		in:  `{"2":4}`,
		ptr: new(map[u8marshal]int),
		err: errMissingU8Prefix,
	},

	// integer-keyed map errors
	{
		in:  `{"abc":"abc"}`,
		ptr: new(map[int]string),
		err: &UnmarshalTypeError{Value: "number abc", Type: reflect.TypeOf(0)},
	},
	{
		in:  `{"256":"abc"}`,
		ptr: new(map[uint8]string),
		err: &UnmarshalTypeError{Value: "number 256", Type: reflect.TypeOf(uint8(0))},
	},
	{
		in:  `{"128":"abc"}`,
		ptr: new(map[int8]string),
		err: &UnmarshalTypeError{Value: "number 128", Type: reflect.TypeOf(int8(0))},
	},
	{
		in:  `{"-1":"abc"}`,
		ptr: new(map[uint8]string),
		err: &UnmarshalTypeError{Value: "number -1", Type: reflect.TypeOf(uint8(0))},
	},

	// Map keys can be encoding.TextUnmarshalers.
	{in: `{"x:y":true}`, ptr: &ummapType, out: ummapXY},
	// If multiple values for the same key exists, only the most recent value is used.
	{in: `{"x:y":false,"x:y":true}`, ptr: &ummapType, out: ummapXY},

	// Overwriting of data.
	// This is different from package xml, but it's what we've always done.
	// Now documented and tested.
	{in: `[2]`, ptr: sliceAddr([]int{1}), out: []int{2}},
	{in: `{"key": 2}`, ptr: mapAddr(map[string]int{"old": 0, "key": 1}), out: map[string]int{"key": 2}},

	// XXX: impompatible behavior
	// the order of object keys are ignored, so the behavior of following test is non-deterministic.
	// {
	// 	in: `{
	// 		"Level0": 1,
	// 		"Level1b": 2,
	// 		"Level1c": 3,
	// 		"x": 4,
	// 		"Level1a": 5,
	// 		"LEVEL1B": 6,
	// 		"e": {
	// 			"Level1a": 8,
	// 			"Level1b": 9,
	// 			"Level1c": 10,
	// 			"Level1d": 11,
	// 			"x": 12
	// 		},
	// 		"Loop1": 13,
	// 		"Loop2": 14,
	// 		"X": 15,
	// 		"Y": 16,
	// 		"Z": 17,
	// 		"Q": 18
	// 	}`,
	// 	ptr: new(Top),
	// 	out: Top{
	// 		Level0: 1,
	// 		Embed0: Embed0{
	// 			Level1b: 2,
	// 			Level1c: 3,
	// 		},
	// 		Embed0a: &Embed0a{
	// 			Level1a: 5,
	// 			Level1b: 6,
	// 		},
	// 		Embed0b: &Embed0b{
	// 			Level1a: 8,
	// 			Level1b: 9,
	// 			Level1c: 10,
	// 			Level1d: 11,
	// 			Level1e: 12,
	// 		},
	// 		Loop: Loop{
	// 			Loop1: 13,
	// 			Loop2: 14,
	// 		},
	// 		Embed0p: Embed0p{
	// 			Point: image.Point{X: 15, Y: 16},
	// 		},
	// 		Embed0q: Embed0q{
	// 			Point: Point{Z: 17},
	// 		},
	// 		embed: embed{
	// 			Q: 18,
	// 		},
	// 	},
	// },

	// rewrite to be deterministic behavior.
	{
		in: `{
			"Level0": 1,
			"Level1b": 6,
			"Level1c": 3,
			"Level1a": 5,
			"LEVEL1B": 6,
			"e": {
				"Level1a": 8,
				"Level1b": 9,
				"Level1c": 10,
				"Level1d": 11,
				"x": 12
			},
			"Loop1": 13,
			"Loop2": 14,
			"X": 15,
			"Y": 16,
			"Z": 17,
			"Q": 18
		}`,
		ptr: new(Top),
		out: Top{
			Level0: 1,
			Embed0: Embed0{
				Level1b: 6,
				Level1c: 3,
			},
			Embed0a: &Embed0a{
				Level1a: 5,
				Level1b: 6,
			},
			Embed0b: &Embed0b{
				Level1a: 8,
				Level1b: 9,
				Level1c: 10,
				Level1d: 11,
				Level1e: 12,
			},
			Loop: Loop{
				Loop1: 13,
				Loop2: 14,
			},
			Embed0p: Embed0p{
				Point: image.Point{X: 15, Y: 16},
			},
			Embed0q: Embed0q{
				Point: Point{Z: 17},
			},
			embed: embed{
				Q: 18,
			},
		},
	},
	{
		in:  `{"hello": 1}`,
		ptr: new(Ambig),
		out: Ambig{First: 1},
	},

	{
		in:  `{"X": 1,"Y":2}`,
		ptr: new(S5),
		out: S5{S8: S8{S9: S9{Y: 2}}},
	},
	{
		in:                    `{"X": 1,"Y":2}`,
		ptr:                   new(S5),
		err:                   fmt.Errorf("json: unknown field \"X\""),
		disallowUnknownFields: true,
	},
	{
		in:  `{"X": 1,"Y":2}`,
		ptr: new(S10),
		out: S10{S13: S13{S8: S8{S9: S9{Y: 2}}}},
	},
	{
		in:                    `{"X": 1,"Y":2}`,
		ptr:                   new(S10),
		err:                   fmt.Errorf("json: unknown field \"X\""),
		disallowUnknownFields: true,
	},

	// invalid UTF-8 is coerced to valid UTF-8.
	{
		in:  "\"hello\xffworld\"",
		ptr: new(string),
		out: "hello\ufffdworld",
	},
	{
		in:  "\"hello\xc2\xc2world\"",
		ptr: new(string),
		out: "hello\ufffd\ufffdworld",
	},
	{
		in:  "\"hello\xc2\xffworld\"",
		ptr: new(string),
		out: "hello\ufffd\ufffdworld",
	},
	{
		in:  "\"hello\\ud800world\"",
		ptr: new(string),
		out: "hello\ufffdworld",
	},
	{
		in:  "\"hello\\ud800\\ud800world\"",
		ptr: new(string),
		out: "hello\ufffd\ufffdworld",
	},
	{
		in:  "\"hello\\ud800\\ud800world\"",
		ptr: new(string),
		out: "hello\ufffd\ufffdworld",
	},
	{
		in:  "\"hello\xed\xa0\x80\xed\xb0\x80world\"",
		ptr: new(string),
		out: "hello\ufffd\ufffd\ufffd\ufffd\ufffd\ufffdworld",
	},

	// Used to be issue https://github.com/golang/go/issues/8305, but time.Time implements encoding.TextUnmarshaler so this works now.
	{
		in:  `{"2009-11-10T23:00:00Z": "hello world"}`,
		ptr: &map[time.Time]string{},
		out: map[time.Time]string{time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC): "hello world"},
	},

	// issue https://github.com/golang/go/issues/8305
	{
		in:  `{"2009-11-10T23:00:00Z": "hello world"}`,
		ptr: &map[Point]string{},
		err: &UnmarshalTypeError{Value: "object", Type: reflect.TypeOf(map[Point]string{})},
	},
	{
		in:  `{"asdf": "hello world"}`,
		ptr: &map[unmarshaler]string{},
		err: &UnmarshalTypeError{Value: "object", Type: reflect.TypeOf(map[unmarshaler]string{})},
	},

	// related to issue https://github.com/golang/go/issues/13783.
	// Go 1.7 changed marshaling a slice of typed byte to use the methods on the byte type,
	// similar to marshaling a slice of typed int.
	// These tests check that, assuming the byte type also has valid decoding methods,
	// either the old base64 string encoding or the new per-element encoding can be
	// successfully unmarshaled. The custom unmarshalers were accessible in earlier
	// versions of Go, even though the custom marshaler was not.
	{
		in:  `"AQID"`,
		ptr: new([]byteWithMarshalJSON),
		out: []byteWithMarshalJSON{1, 2, 3},
	},
	{
		in:     `["Z01","Z02","Z03"]`,
		ptr:    new([]byteWithMarshalJSON),
		out:    []byteWithMarshalJSON{1, 2, 3},
		golden: true,
	},
	{
		in:  `"AQID"`,
		ptr: new([]byteWithMarshalText),
		out: []byteWithMarshalText{1, 2, 3},
	},
	{
		in:     `["Z01","Z02","Z03"]`,
		ptr:    new([]byteWithMarshalText),
		out:    []byteWithMarshalText{1, 2, 3},
		golden: true,
	},
	{
		in:  `"AQID"`,
		ptr: new([]byteWithPtrMarshalJSON),
		out: []byteWithPtrMarshalJSON{1, 2, 3},
	},
	{
		in:     `["Z01","Z02","Z03"]`,
		ptr:    new([]byteWithPtrMarshalJSON),
		out:    []byteWithPtrMarshalJSON{1, 2, 3},
		golden: true,
	},
	{
		in:  `"AQID"`,
		ptr: new([]byteWithPtrMarshalText),
		out: []byteWithPtrMarshalText{1, 2, 3},
	},
	{
		in:     `["Z01","Z02","Z03"]`,
		ptr:    new([]byteWithPtrMarshalText),
		out:    []byteWithPtrMarshalText{1, 2, 3},
		golden: true,
	},

	{in: `0.000001`, ptr: new(float64), out: 0.000001, golden: true},
	{in: `1e-7`, ptr: new(float64), out: 1e-7, golden: true},
	{in: `100000000000000000000`, ptr: new(float64), out: 100000000000000000000.0, golden: true},
	{in: `1e+21`, ptr: new(float64), out: 1e21, golden: true},
	{in: `-0.000001`, ptr: new(float64), out: -0.000001, golden: true},
	{in: `-1e-7`, ptr: new(float64), out: -1e-7, golden: true},
	{in: `-100000000000000000000`, ptr: new(float64), out: -100000000000000000000.0, golden: true},
	{in: `-1e+21`, ptr: new(float64), out: -1e21, golden: true},
	{in: `999999999999999900000`, ptr: new(float64), out: 999999999999999900000.0, golden: true},
	{in: `9007199254740992`, ptr: new(float64), out: 9007199254740992.0, golden: true},
	{in: `9007199254740993`, ptr: new(float64), out: 9007199254740992.0, golden: false},

	{
		in:  `{"V": {"F2": "hello"}}`,
		ptr: new(VOuter),
		err: &UnmarshalTypeError{
			Value:  "string",
			Struct: "V",
			Field:  "F2",
			Type:   reflect.TypeOf(int32(0)),
		},
	},
	{
		in:  `{"V": {"F4": {}, "F2": "hello"}}`,
		ptr: new(VOuter),
		err: &UnmarshalTypeError{
			Value:  "string",
			Struct: "V",
			Field:  "F2",
			Type:   reflect.TypeOf(int32(0)),
		},
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
		in := []byte(tt.in)
		dec := NewDecoder(bytes.NewReader(in))
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
			data, _ := Marshal(v.Elem().Interface())
			fmt.Println(string(data))
			data, _ = Marshal(tt.out)
			fmt.Println(string(data))
			continue
		}

		// Check round trip also decodes correctly.
		if tt.err == nil {
			enc, err := Marshal(v.Interface())
			if err != nil {
				t.Errorf("#%d: errir re-marshaling: %v", i, err)
			}
			if tt.golden && !bytes.Equal(enc, in) {
				t.Errorf("#%d: remarshal mismatch:\nhave: %s\nwant: %s", i, enc, in)
			}
			vv := reflect.New(reflect.TypeOf(tt.ptr).Elem())
			dec = NewDecoder(bytes.NewReader(enc))
			if tt.useNumber {
				dec.UseNumber()
			}
			if err := dec.Decode(vv.Interface()); err != nil {
				t.Errorf("#%d: error re-unmarshaling %#q: %v", i, enc, err)
				continue
			}
			if !reflect.DeepEqual(v.Elem().Interface(), vv.Elem().Interface()) {
				t.Errorf("#%d: mismatch\nhave: %#+v\nwant: %#+v", i, v.Elem().Interface(), vv.Elem().Interface())
				t.Errorf("     In: %q", strings.Map(noSpace, string(in)))
				t.Errorf("Marshal: %q", strings.Map(noSpace, string(enc)))
				continue
			}
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

type NullTest struct {
	Bool      bool
	Int       int
	Int8      int8
	Int16     int16
	Int32     int32
	Int64     int64
	Uint      uint
	Uint8     uint8
	Uint16    uint16
	Uint32    uint32
	Uint64    uint64
	Float32   float32
	Float64   float64
	String    string
	PBool     *bool
	Map       map[string]string
	Slice     []string
	Interface interface{}

	PRaw    *RawMessage
	PTime   *time.Time
	PBigInt *big.Int
	PText   *MustNotUnmarshalText
	PBuffer *bytes.Buffer // has methods, just not relevant ones
	PStruct *struct{}

	Raw    RawMessage
	Time   time.Time
	BigInt big.Int
	Text   MustNotUnmarshalText
	Buffer bytes.Buffer
	Struct struct{}
}

type NullTestStrings struct {
	Bool      bool              `json:",string"`
	Int       int               `json:",string"`
	Int8      int8              `json:",string"`
	Int16     int16             `json:",string"`
	Int32     int32             `json:",string"`
	Int64     int64             `json:",string"`
	Uint      uint              `json:",string"`
	Uint8     uint8             `json:",string"`
	Uint16    uint16            `json:",string"`
	Uint32    uint32            `json:",string"`
	Uint64    uint64            `json:",string"`
	Float32   float32           `json:",string"`
	Float64   float64           `json:",string"`
	String    string            `json:",string"`
	PBool     *bool             `json:",string"`
	Map       map[string]string `json:",string"`
	Slice     []string          `json:",string"`
	Interface interface{}       `json:",string"`

	PRaw    *RawMessage           `json:",string"`
	PTime   *time.Time            `json:",string"`
	PBigInt *big.Int              `json:",string"`
	PText   *MustNotUnmarshalText `json:",string"`
	PBuffer *bytes.Buffer         `json:",string"`
	PStruct *struct{}             `json:",string"`

	Raw    RawMessage           `json:",string"`
	Time   time.Time            `json:",string"`
	BigInt big.Int              `json:",string"`
	Text   MustNotUnmarshalText `json:",string"`
	Buffer bytes.Buffer         `json:",string"`
	Struct struct{}             `json:",string"`
}

// JSON null values should be ignored for primitives and string values instead of resulting in an error.
// Issue 2540
func TestUnmarshalNulls(t *testing.T) {
	// Unmarshal docs:
	// The JSON null value unmarshals into an interface, map, pointer, or slice
	// by setting that Go value to nil. Because null is often used in JSON to mean
	// ``not present,'' unmarshaling a JSON null into any other Go type has no effect
	// on the value and produces no error.

	jsonData := []byte(`{
				"Bool"    : null,
				"Int"     : null,
				"Int8"    : null,
				"Int16"   : null,
				"Int32"   : null,
				"Int64"   : null,
				"Uint"    : null,
				"Uint8"   : null,
				"Uint16"  : null,
				"Uint32"  : null,
				"Uint64"  : null,
				"Float32" : null,
				"Float64" : null,
				"String"  : null,
				"PBool": null,
				"Map": null,
				"Slice": null,
				"Interface": null,
				"PRaw": null,
				"PTime": null,
				"PBigInt": null,
				"PText": null,
				"PBuffer": null,
				"PStruct": null,
				"Raw": null,
				"Time": null,
				"BigInt": null,
				"Text": null,
				"Buffer": null,
				"Struct": null
			}`)
	nulls := NullTest{
		Bool:      true,
		Int:       2,
		Int8:      3,
		Int16:     4,
		Int32:     5,
		Int64:     6,
		Uint:      7,
		Uint8:     8,
		Uint16:    9,
		Uint32:    10,
		Uint64:    11,
		Float32:   12.1,
		Float64:   13.1,
		String:    "14",
		PBool:     new(bool),
		Map:       map[string]string{},
		Slice:     []string{},
		Interface: new(MustNotUnmarshalJSON),
		PRaw:      new(RawMessage),
		PTime:     new(time.Time),
		PBigInt:   new(big.Int),
		PText:     new(MustNotUnmarshalText),
		PStruct:   new(struct{}),
		PBuffer:   new(bytes.Buffer),
		Raw:       RawMessage("123"),
		Time:      time.Unix(123456789, 0),
		BigInt:    *big.NewInt(123),
	}

	before := nulls.Time.String()

	err := Unmarshal(jsonData, &nulls)
	if err != nil {
		t.Errorf("Unmarshal of null values failed: %v", err)
	}
	if !nulls.Bool || nulls.Int != 2 || nulls.Int8 != 3 || nulls.Int16 != 4 || nulls.Int32 != 5 || nulls.Int64 != 6 ||
		nulls.Uint != 7 || nulls.Uint8 != 8 || nulls.Uint16 != 9 || nulls.Uint32 != 10 || nulls.Uint64 != 11 ||
		nulls.Float32 != 12.1 || nulls.Float64 != 13.1 || nulls.String != "14" {
		t.Errorf("Unmarshal of null values affected primitives")
	}

	if nulls.PBool != nil {
		t.Errorf("Unmarshal of null did not clear nulls.PBool")
	}
	if nulls.Map != nil {
		t.Errorf("Unmarshal of null did not clear nulls.Map")
	}
	if nulls.Slice != nil {
		t.Errorf("Unmarshal of null did not clear nulls.Slice")
	}
	if nulls.Interface != nil {
		t.Errorf("Unmarshal of null did not clear nulls.Interface")
	}
	if nulls.PRaw != nil {
		t.Errorf("Unmarshal of null did not clear nulls.PRaw")
	}
	if nulls.PTime != nil {
		t.Errorf("Unmarshal of null did not clear nulls.PTime")
	}
	if nulls.PBigInt != nil {
		t.Errorf("Unmarshal of null did not clear nulls.PBigInt")
	}
	if nulls.PText != nil {
		t.Errorf("Unmarshal of null did not clear nulls.PText")
	}
	if nulls.PBuffer != nil {
		t.Errorf("Unmarshal of null did not clear nulls.PBuffer")
	}
	if nulls.PStruct != nil {
		t.Errorf("Unmarshal of null did not clear nulls.PStruct")
	}

	if string(nulls.Raw) != "null" {
		t.Errorf("Unmarshal of RawMessage null did not record null: %v", string(nulls.Raw))
	}
	if nulls.Time.String() != before {
		t.Errorf("Unmarshal of time.Time null set time to %v", nulls.Time.String())
	}
	if nulls.BigInt.String() != "123" {
		t.Errorf("Unmarshal of big.Int null set int to %v", nulls.BigInt.String())
	}
}

type MustNotUnmarshalJSON struct{}

func (x MustNotUnmarshalJSON) UnmarshalJSON(data []byte) error {
	return errors.New("MustNotUnmarshalJSON was used")
}

type MustNotUnmarshalText struct{}

func (x MustNotUnmarshalText) UnmarshalText(text []byte) error {
	return errors.New("MustNotUnmarshalText was used")
}

func TestStringKind(t *testing.T) {
	type stringKind string

	var m1, m2 map[stringKind]int
	m1 = map[stringKind]int{
		"foo": 42,
	}

	data, err := Marshal(m1)
	if err != nil {
		t.Errorf("Unexpected error marshaling: %v", err)
	}

	err = Unmarshal(data, &m2)
	if err != nil {
		t.Errorf("Unexpected error unmarshaling: %v", err)
	}

	if !reflect.DeepEqual(m1, m2) {
		t.Error("Items should be equal after encoding and then decoding")
	}
}

// The fix for issue https://github.com/golang/go/issues/8962 introduced a regression.
// Issue https://github.com/golang/go/issues/12921.
func TestSliceOfCustomByte(t *testing.T) {
	type Uint8 uint8

	a := []Uint8("hello")

	data, err := Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var b []Uint8
	err = Unmarshal(data, &b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("expected %v == %v", a, b)
	}
}

// Test handling of unexported fields that should be ignored.
// Issue https://github.com/golang/go/issues/4660
type unexportedFields struct {
	Name string
	m    map[string]interface{} `json:"-"`
	m2   map[string]interface{} `json:"abcd"`
}

func TestUnmarshalUnexported(t *testing.T) {
	input := `{"Name": "Bob", "m": {"x": 123}, "m2": {"y": 456}, "abcd": {"z": 789}}`
	want := &unexportedFields{Name: "Bob"}

	out := &unexportedFields{}
	err := Unmarshal([]byte(input), out)
	if err != nil {
		t.Errorf("got error %v, expected nil", err)
	}
	if !reflect.DeepEqual(out, want) {
		t.Errorf("got %q, want %q", out, want)
	}
}

// Test that extra object elements in an array do not result in a
// "data changing underfoot" error.
// Issue https://github.com/golang/go/issues/3717
func TestSkipArrayObjects(t *testing.T) {
	json := `[{}]`
	var dest [0]interface{}

	err := Unmarshal([]byte(json), &dest)
	if err != nil {
		t.Errorf("got error %q, want nil", err)
	}
}

// Test semantics of pre-filled struct fields and pre-filled map fields.
// Issue https://github.com/golang/go/issues/4900.
func TestPrefilled(t *testing.T) {
	ptrToMap := func(m map[string]interface{}) *map[string]interface{} { return &m }

	// Values here change, cannot reuse table across runs.
	var prefillTests = []struct {
		in  string
		ptr interface{}
		out interface{}
	}{
		{
			in:  `{"X": 1, "Y": 2}`,
			ptr: &XYZ{X: float32(3), Y: int16(4), Z: 1.5},
			out: &XYZ{X: float64(1), Y: float64(2), Z: 1.5},
		},
		{
			in:  `{"X": 1, "Y": 2}`,
			ptr: ptrToMap(map[string]interface{}{"X": float32(3), "Y": int16(4), "Z": 1.5}),
			out: ptrToMap(map[string]interface{}{"X": float64(1), "Y": float64(2), "Z": 1.5}),
		},
	}

	for _, tt := range prefillTests {
		ptrstr := fmt.Sprintf("%v", tt.ptr)
		err := Unmarshal([]byte(tt.in), tt.ptr) // tt.ptr edited here
		if err != nil {
			t.Errorf("Unmarshal: %v", err)
		}
		if !reflect.DeepEqual(tt.ptr, tt.out) {
			t.Errorf("Unmarshal(%#q, %s): have %v, want %v", tt.in, ptrstr, tt.ptr, tt.out)
		}
	}
}

var invalidUnmarshalTests = []struct {
	v    interface{}
	want string
}{
	{nil, "json: Unmarshal(nil)"},
	{struct{}{}, "json: Unmarshal(non-pointer struct {})"},
	{(*int)(nil), "json: Unmarshal(nil *int)"},
}

func TestInvalidUnmarshal(t *testing.T) {
	buf := []byte(`{"a":"1"}`)
	for _, tt := range invalidUnmarshalTests {
		err := Unmarshal(buf, tt.v)
		if err == nil {
			t.Errorf("Unmarshal expecting error, got nil")
			continue
		}
		if got := err.Error(); got != tt.want {
			t.Errorf("Unmarshal = %q; want %q", got, tt.want)
		}
	}
}
