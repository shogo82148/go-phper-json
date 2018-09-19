package phperjson

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

// Decoder is a wrapper of json.Decoder.
type Decoder struct {
	dec                   *json.Decoder
	disallowUnknownFields bool
	useNumber             bool
}

func NewDecoder(r io.Reader) *Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return &Decoder{
		dec: dec,
	}
}

func (dec *Decoder) Buffered() io.Reader {
	return dec.dec.Buffered()
}

// from the encoding/json package.
// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
// if it encounters an Unmarshaler, indirect stops and returns that.
// if decodingNull is true, indirect stops at the last pointer so it can be set to nil.
func indirect(v reflect.Value, decodingNull bool) (Unmarshaler, encoding.TextUnmarshaler, reflect.Value) {
	// Issue #24153 indicates that it is generally not a guaranteed property
	// that you may round-trip a reflect.Value by calling Value.Addr().Elem()
	// and expect the value to still be settable for values derived from
	// unexported embedded struct fields.
	//
	// The logic below effectively does this when it first addresses the value
	// (to satisfy possible pointer methods) and continues to dereference
	// subsequent pointers as necessary.
	//
	// After the first round-trip, we set v back to the original value to
	// preserve the original RW flags contained in reflect.Value.
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Ptr) {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && decodingNull && v.CanSet() {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(Unmarshaler); ok {
				return u, nil, reflect.Value{}
			}
			if !decodingNull {
				if u, ok := v.Interface().(encoding.TextUnmarshaler); ok {
					return nil, u, reflect.Value{}
				}
			}
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}
	return nil, nil, v
}

func (dec *Decoder) Decode(v interface{}) error {
	var iv interface{}
	if err := dec.dec.Decode(&iv); err != nil {
		return err
	}
	return dec.decode(iv, reflect.ValueOf(v))
}

func (dec *Decoder) decode(in interface{}, out reflect.Value) error {
	u, ut, pv := indirect(out, in == nil)
	if u != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		return u.UnmarshalJSON(data)
	}
	if ut != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		return ut.UnmarshalText(data)
	}

	out = pv
	switch v := in.(type) {
	case nil:
		switch out.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			out.Set(reflect.Zero(out.Type()))
			// otherwise, ignore null for primitives/string
		}
	case bool:
		switch out.Kind() {
		default:
			panic("TODO: handle UnmarshalTypeError")
		case reflect.Bool:
			out.SetBool(v)
		case reflect.Interface:
			if out.NumMethod() == 0 {
			} else {
				panic("TODO: handle UnmarshalTypeError")
			}
		}
	case Number:
		switch out.Kind() {
		default:
			panic("TODO: handle UnmarshalTypeError")
		case reflect.String:
			out.SetString(string(v))
		case reflect.Interface:
			n, err := dec.convertNumber(string(v))
			if err != nil {
				panic("TODO: handle UnmarshalTypeError")
			}
			if out.NumMethod() != 0 {
				panic("TODO: handle UnmarshalTypeError")
			}
			out.Set(reflect.ValueOf(n))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := v.Int64()
			if err != nil || out.OverflowInt(n) {
				panic("TODO: handle UnmarshalTypeError")
			}
			out.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			n, err := strconv.ParseUint(string(v), 10, 64)
			if err != nil || out.OverflowUint(n) {
				panic("TODO: handle UnmarshalTypeError")
			}
			out.SetUint(n)
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(string(v), out.Type().Bits())
			if err != nil || out.OverflowFloat(n) {
				panic("TODO: handle UnmarshalTypeError")
			}
			out.SetFloat(n)
		}
	case string:
		switch out.Kind() {
		default:
			panic("TODO: handle UnmarshalTypeError")
		case reflect.String:
			out.SetString(v)
		case reflect.Interface:
			out.Set(reflect.ValueOf(v))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(string(v), 10, 64)
			if err != nil || out.OverflowInt(n) {
				panic("TODO: handle UnmarshalTypeError")
			}
			out.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			n, err := strconv.ParseUint(string(v), 10, 64)
			if err != nil || out.OverflowUint(n) {
				panic("TODO: handle UnmarshalTypeError")
			}
			out.SetUint(n)
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(string(v), out.Type().Bits())
			if err != nil || out.OverflowFloat(n) {
				panic("TODO: handle UnmarshalTypeError")
			}
			out.SetFloat(n)
		}
	default:
		panic(fmt.Sprintf("unkown type: %v", reflect.TypeOf(v)))
	}
	return nil
}

func (dec *Decoder) convertNumber(s string) (interface{}, error) {
	if dec.useNumber {
		return Number(s), nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, &UnmarshalTypeError{Value: "number " + s, Type: reflect.TypeOf(0.0)}
	}
	return f, nil
}

func (dec *Decoder) DisallowUnknownFields() {
	dec.disallowUnknownFields = true
}

func (dec *Decoder) More() bool {
	return dec.dec.More()
}

func (dec *Decoder) Token() (json.Token, error) {
	return dec.dec.Token()
}

func (dec *Decoder) UseNumber() {
	dec.useNumber = true
}

func Unmarshal(data []byte, v interface{}) error {
	// Check for well-formedness.
	// Avoids filling out half a data structure
	// before discovering a JSON syntax error.
	err := json.Unmarshal(data, nil)
	if _, ok := err.(*InvalidUnmarshalError); !ok {
		return err
	}

	d := NewDecoder(bytes.NewReader(data))
	return d.Decode(v)
}

// Valid is an alias for json.Valid.
func Valid(data []byte) bool {
	return json.Valid(data)
}

// UnmarshalFieldError is an alias for json.UnmarshalFieldError.
type UnmarshalFieldError = json.UnmarshalFieldError

// UnmarshalTypeError is an alias for json.UnmarshalTypeError.
type UnmarshalTypeError = json.UnmarshalTypeError

// Unmarshaler is an alias for json.Unmarshaler.
type Unmarshaler = json.Unmarshaler

// UnsupportedTypeError is an alias for json.UnsupportedTypeError.
type UnsupportedTypeError = json.UnsupportedTypeError

// UnsupportedValueError is an alias for json.UnsupportedValueError.
type UnsupportedValueError = json.UnsupportedValueError
