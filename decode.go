// Copyright 2011 The Go Authors. All rights reserved.
// Copyright 2018 Shogo Ichinose. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package phperjson

import (
	"bytes"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"strconv"
)

var textUnmarshalerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

// A Decoder reads and decodes JSON values from an input stream.
type Decoder struct {
	dec                   *json.Decoder
	disallowUnknownFields bool
	useNumber             bool
	errorContext          struct { // provides context for type errors
		Struct string
		Field  string
	}
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return &Decoder{
		dec: dec,
	}
}

// Buffered returns a reader of the data remaining in the Decoder's buffer.
// The reader is valid until the next call to Decode.
func (dec *Decoder) Buffered() io.Reader {
	return dec.dec.Buffered()
}

func (dec *Decoder) withErrorContext(err error) error {
	if dec.errorContext.Struct != "" || dec.errorContext.Field != "" {
		switch err := err.(type) {
		case *UnmarshalTypeError:
			err.Struct = dec.errorContext.Struct
			err.Field = dec.errorContext.Field
			return err
		}
	}
	return err
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

// Decode reads the next JSON-encoded value from its input and stores it in the value pointed to by v.
func (dec *Decoder) Decode(v interface{}) error {
	var iv interface{}
	if err := dec.dec.Decode(&iv); err != nil {
		return err
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &InvalidUnmarshalError{Type: reflect.TypeOf(v)}
	}
	return dec.decode(iv, rv)
}

func (dec *Decoder) decode(in interface{}, out reflect.Value) error {
	if !out.IsValid() {
		return nil
	}

	u, ut, pv := indirect(out, in == nil)
	if u != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		return u.UnmarshalJSON(data)
	}
	if ut != nil {
		switch v := in.(type) {
		default:
			return dec.withErrorContext(&UnmarshalTypeError{Type: out.Type()})
		case nil:
			return dec.withErrorContext(&UnmarshalTypeError{Value: "null", Type: out.Type()})
		case bool:
			return dec.withErrorContext(&UnmarshalTypeError{Value: "bool", Type: out.Type()})
		case Number:
			return dec.withErrorContext(&UnmarshalTypeError{Value: "number", Type: out.Type()})
		case string:
			return ut.UnmarshalText([]byte(v))
		}
	}

	out = pv
	switch v := in.(type) {
	case nil:
		switch out.Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice:
			out.Set(reflect.Zero(out.Type()))
			// otherwise, ignore null for primitives
		}
	case bool:
		switch out.Kind() {
		default:
			return dec.withErrorContext(&UnmarshalTypeError{Value: "bool", Type: out.Type()})
		case reflect.Bool:
			out.SetBool(v)
		case reflect.Interface:
			if out.NumMethod() == 0 {
				out.Set(reflect.ValueOf(v))
			} else {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "bool", Type: out.Type()})
			}
		case reflect.String:
			// PHP flavored http://php.net/manual/en/language.types.string.php#language.types.string.casting
			// A boolean TRUE value is converted to the string "1".
			// Boolean FALSE is converted to "" (the empty string).
			// This allows conversion back and forth between boolean and string values.
			if v {
				out.SetString("1")
			} else {
				out.SetString("")
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// PHP flavored http://php.net/manual/en/language.types.integer.php#language.types.integer.casting
			// FALSE will yield 0 (zero), and TRUE will yield 1 (one).
			if v {
				out.SetInt(1)
			} else {
				out.SetInt(0)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			// PHP flavored http://php.net/manual/en/language.types.string.php#language.types.string.casting
			// FALSE will yield 0 (zero), and TRUE will yield 1 (one).
			if v {
				out.SetUint(1)
			} else {
				out.SetUint(0)
			}
		case reflect.Float32, reflect.Float64:
			// PHP flavored http://php.net/manual/en/language.types.float.php#language.types.float.casting
			// FALSE will yield 0 (zero), and TRUE will yield 1 (one).
			if v {
				out.SetFloat(1)
			} else {
				out.SetFloat(0)
			}
		case reflect.Slice:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if out.Cap() == 0 {
				newout := reflect.MakeSlice(out.Type(), 1, 1)
				out.Set(newout)
			}
			out.SetLen(1)
			if err := dec.decode(v, out.Index(0)); err != nil {
				return err
			}
		case reflect.Map:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if err := dec.decode(map[string]interface{}{"0": v}, out); err != nil {
				return err
			}
		case reflect.Struct:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if err := dec.decode(map[string]interface{}{"0": v}, out); err != nil {
				return err
			}
		}
	case Number:
		switch out.Kind() {
		default:
			return dec.withErrorContext(&UnmarshalTypeError{Value: "number", Type: out.Type()})
		case reflect.String:
			out.SetString(string(v))
		case reflect.Interface:
			n, err := dec.convertNumber(string(v))
			if err != nil {
				return err
			}
			if out.NumMethod() != 0 {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "number", Type: out.Type()})
			}
			out.Set(reflect.ValueOf(n))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := parseInt(string(v), out.Type())
			if err != nil {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "number " + string(v), Type: out.Type()})
			}
			out.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			n, err := parseUint(string(v), out.Type())
			if err != nil {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "number " + string(v), Type: out.Type()})
			}
			out.SetUint(n)
		case reflect.Float32, reflect.Float64:
			n, err := strconv.ParseFloat(string(v), out.Type().Bits())
			if err != nil || out.OverflowFloat(n) {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "number " + string(v), Type: out.Type()})
			}
			out.SetFloat(n)
		case reflect.Bool:
			// PHP flavored http://php.net/manual/en/language.types.boolean.php#language.types.boolean.casting
			// the integer 0 (zero)
			// the float 0.0 (zero)
			n, err := strconv.ParseFloat(string(v), 64)
			if err == nil && n == 0 {
				out.SetBool(false)
			} else {
				out.SetBool(true)
			}
		case reflect.Slice:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if out.Cap() == 0 {
				newout := reflect.MakeSlice(out.Type(), 1, 1)
				out.Set(newout)
			}
			out.SetLen(1)
			if err := dec.decode(v, out.Index(0)); err != nil {
				return err
			}
		case reflect.Map:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if err := dec.decode(map[string]interface{}{"0": v}, out); err != nil {
				return err
			}
		case reflect.Struct:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if err := dec.decode(map[string]interface{}{"0": v}, out); err != nil {
				return err
			}
		}
	case string:
		switch out.Kind() {
		default:
			return dec.withErrorContext(&UnmarshalTypeError{Value: "string", Type: out.Type()})
		case reflect.String:
			out.SetString(v)
		case reflect.Interface:
			out.Set(reflect.ValueOf(v))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v == "" {
				out.SetInt(0)
				break
			}
			n, err := parseInt(v, out.Type())
			if err != nil {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "string", Type: out.Type()})
			}
			out.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if v == "" {
				out.SetUint(0)
				break
			}
			n, err := parseUint(v, out.Type())
			if err != nil {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "string", Type: out.Type()})
			}
			out.SetUint(n)
		case reflect.Float32, reflect.Float64:
			if v == "" {
				out.SetFloat(0)
				break
			}
			n, err := strconv.ParseFloat(string(v), out.Type().Bits())
			if err != nil || out.OverflowFloat(n) {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "string", Type: out.Type()})
			}
			out.SetFloat(n)
		case reflect.Bool:
			// PHP flavored http://php.net/manual/en/language.types.boolean.php#language.types.boolean.casting
			// When converting to boolean, the following values are considered FALSE:
			// the empty string, and the string "0"
			if v == "" || v == "0" {
				out.SetBool(false)
			} else {
				out.SetBool(true)
			}
		case reflect.Slice:
			if out.Type().Elem().Kind() == reflect.Uint8 {
				b, err := base64.StdEncoding.DecodeString(v)
				if err != nil {
					return err
				}
				out.SetBytes(b)
				break
			}
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if out.Cap() == 0 {
				newout := reflect.MakeSlice(out.Type(), 1, 1)
				out.Set(newout)
			}
			out.SetLen(1)
			if err := dec.decode(v, out.Index(0)); err != nil {
				return err
			}
		case reflect.Map:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if err := dec.decode(map[string]interface{}{"0": v}, out); err != nil {
				return err
			}
		case reflect.Struct:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// For any of the types integer, float, string, boolean and resource,
			// converting a value to an array results in an array with a single element with index zero and the value of the scalar which was converted.
			// In other words, (array)$scalarValue is exactly the same as array($scalarValue).
			if err := dec.decode(map[string]interface{}{"0": v}, out); err != nil {
				return err
			}
		}
	case []interface{}:
		switch out.Kind() {
		default:
			return dec.withErrorContext(&UnmarshalTypeError{Value: "array", Type: out.Type()})
		case reflect.Interface:
			if out.NumMethod() == 0 {
				out.Set(reflect.ValueOf(v))
			} else {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "array", Type: out.Type()})
			}
		case reflect.Array:
			l := len(v)
			if out.Len() < l {
				// Ran out of fixed array: skip.
				l = out.Len()
			}
			var i int
			for i = 0; i < l; i++ {
				if err := dec.decode(v[i], out.Index(i)); err != nil {
					return err
				}
			}
			if i < out.Len() {
				// Zero the rest.
				zero := reflect.Zero(out.Type().Elem())
				for ; i < out.Len(); i++ {
					out.Index(i).Set(zero)
				}
			}
		case reflect.Slice:
			if len(v) == 0 {
				out.Set(reflect.MakeSlice(out.Type(), 0, 0))
				break
			}
			// Grow slice if necessary
			if len(v) > out.Cap() {
				newout := reflect.MakeSlice(out.Type(), len(v), len(v))
				out.Set(newout)
			}
			out.SetLen(len(v))
			for i, vv := range v {
				if err := dec.decode(vv, out.Index(i)); err != nil {
					return err
				}
			}
		case reflect.Bool:
			// PHP flavored http://php.net/manual/en/language.types.boolean.php#language.types.boolean.casting
			// When converting to boolean, the following values are considered FALSE:
			// an array with zero elements
			if len(v) == 0 {
				out.SetBool(false)
			} else {
				out.SetBool(true)
			}
		case reflect.Map:
			// PHP flavored
			// PHP doesn's not distinguish JSON arrays from JSON objects.
			t := out.Type()
			kt := t.Key()
			// Map key must either have string kind, have an integer kind,
			// or be an encoding.TextUnmarshaler.
			switch kt.Kind() {
			case reflect.String,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			default:
				if !reflect.PtrTo(kt).Implements(textUnmarshalerType) {
					return dec.withErrorContext(&UnmarshalTypeError{Value: "object", Type: out.Type()})
				}
			}
			if out.IsNil() {
				out.Set(reflect.MakeMap(t))
			}
			var mapElem reflect.Value
			for i, vv := range v {
				// decode value
				elemType := out.Type().Elem()
				if !mapElem.IsValid() {
					mapElem = reflect.New(elemType).Elem()
				} else {
					mapElem.Set(reflect.Zero(elemType))
				}
				subv := mapElem
				if err := dec.decode(vv, subv); err != nil {
					return err
				}
				// decode key
				var kv reflect.Value
				switch {
				case kt.Kind() == reflect.String:
					kv = reflect.ValueOf(strconv.Itoa(i)).Convert(kt)
				case reflect.PtrTo(kt).Implements(textUnmarshalerType):
					kv = reflect.New(kt)
					if err := dec.decode(strconv.Itoa(i), kv); err != nil {
						return err
					}
					kv = kv.Elem()
				default:
					switch kt.Kind() {
					case reflect.String:
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						if reflect.Zero(kt).OverflowInt(int64(i)) {
							return dec.withErrorContext(&UnmarshalTypeError{Value: "number", Type: kt})
						}
						kv = reflect.ValueOf(int64(i)).Convert(kt)
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
						if reflect.Zero(kt).OverflowUint(uint64(i)) {
							return dec.withErrorContext(&UnmarshalTypeError{Value: "number", Type: kt})
						}
						kv = reflect.ValueOf(uint64(i)).Convert(kt)
					default:
						panic("json: Unexpected key type") // should never occur
					}
				}
				out.SetMapIndex(kv, subv)
			}
		case reflect.Struct:
			// PHP flavored
			// PHP doesn's not distinguish JSON arrays from JSON objects.
			for i, value := range v {
				// Figure out field corresponding to key.
				key := strconv.Itoa(i)
				var subv reflect.Value
				var f *field
				fields := cachedTypeFields(out.Type())
				for i := range fields {
					ff := &fields[i]
					if ff.name == key {
						f = ff
						break
					}
				}
				if f != nil {
					subv = out
					for _, i := range f.index {
						if subv.Kind() == reflect.Ptr {
							if subv.IsNil() {
								if !subv.CanSet() {
									return fmt.Errorf("phperjson: cannot set embedded pointer to unexported struct: %v", subv.Type().Elem())
								}
								subv.Set(reflect.New(subv.Type().Elem()))
							}
							subv = subv.Elem()
						}
						subv = subv.Field(i)
					}
					dec.errorContext.Struct = out.Type().Name()
					dec.errorContext.Field = f.name
				} else if dec.disallowUnknownFields {
					return fmt.Errorf("json: unknown field %q", key)
				}
				err := dec.decode(value, subv)
				dec.errorContext.Struct = ""
				dec.errorContext.Field = ""
				if err != nil {
					return err
				}
			}
		}
	case map[string]interface{}:
		switch out.Kind() {
		default:
			return dec.withErrorContext(&UnmarshalTypeError{Value: "object", Type: out.Type()})
		case reflect.Interface:
			if out.NumMethod() == 0 {
				if dec.useNumber {
					out.Set(reflect.ValueOf(v))
				} else if converted, err := dec.convertNumber2Float64(v); err == nil {
					out.Set(reflect.ValueOf(converted))
				} else {
					return err
				}
			} else {
				return dec.withErrorContext(&UnmarshalTypeError{Value: "object", Type: out.Type()})
			}
		case reflect.Map:
			t := out.Type()
			kt := t.Key()
			if kt.Kind() == reflect.String && t.Elem().Kind() == reflect.Interface && out.Len() == 0 {
				out.Set(reflect.ValueOf(v))
				break
			}

			// Map key must either have string kind, have an integer kind,
			// or be an encoding.TextUnmarshaler.
			switch kt.Kind() {
			case reflect.String,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			default:
				if !reflect.PtrTo(kt).Implements(textUnmarshalerType) {
					return dec.withErrorContext(&UnmarshalTypeError{Value: "object", Type: out.Type()})
				}
			}
			if out.IsNil() {
				out.Set(reflect.MakeMap(t))
			}
			var mapElem reflect.Value
			for key, vv := range v {
				elemType := out.Type().Elem()
				if !mapElem.IsValid() {
					mapElem = reflect.New(elemType).Elem()
				} else {
					mapElem.Set(reflect.Zero(elemType))
				}
				subv := mapElem
				if err := dec.decode(vv, subv); err != nil {
					return err
				}
				var kv reflect.Value
				switch {
				case kt.Kind() == reflect.String:
					kv = reflect.ValueOf(key).Convert(kt)
				case reflect.PtrTo(kt).Implements(textUnmarshalerType):
					kv = reflect.New(kt)
					if err := dec.decode(key, kv); err != nil {
						return err
					}
					kv = kv.Elem()
				default:
					switch kt.Kind() {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
						n, err := strconv.ParseInt(key, 10, 64)
						if err != nil || reflect.Zero(kt).OverflowInt(n) {
							return dec.withErrorContext(&UnmarshalTypeError{Value: "number " + key, Type: kt})
						}
						kv = reflect.ValueOf(n).Convert(kt)
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
						n, err := strconv.ParseUint(key, 10, 64)
						if err != nil || reflect.Zero(kt).OverflowUint(n) {
							return dec.withErrorContext(&UnmarshalTypeError{Value: "number " + key, Type: kt})
						}
						kv = reflect.ValueOf(n).Convert(kt)
					default:
						panic("json: Unexpected key type") // should never occur
					}
				}
				out.SetMapIndex(kv, subv)
			}
		case reflect.Struct:
			for key, value := range v {
				// Figure out field corresponding to key.
				var subv reflect.Value
				var f *field
				fields := cachedTypeFields(out.Type())
				for i := range fields {
					ff := &fields[i]
					if ff.name == key {
						f = ff
						break
					}
					if f == nil && ff.equalFold(ff.nameBytes, []byte(key)) {
						f = ff
					}
				}
				if f != nil {
					subv = out
					for _, i := range f.index {
						if subv.Kind() == reflect.Ptr {
							if subv.IsNil() {
								if !subv.CanSet() {
									return fmt.Errorf("phperjson: cannot set embedded pointer to unexported struct: %v", subv.Type().Elem())
								}
								subv.Set(reflect.New(subv.Type().Elem()))
							}
							subv = subv.Elem()
						}
						subv = subv.Field(i)
					}
					dec.errorContext.Struct = out.Type().Name()
					dec.errorContext.Field = f.name
				} else if dec.disallowUnknownFields {
					return fmt.Errorf("json: unknown field %q", key)
				}
				err := dec.decode(value, subv)
				dec.errorContext.Struct = ""
				dec.errorContext.Field = ""
				if err != nil {
					return err
				}
			}
		case reflect.Bool:
			// PHP flavored http://php.net/manual/en/language.types.boolean.php#language.types.boolean.casting
			// When converting to boolean, the following values are considered FALSE:
			// an array with zero elements
			if len(v) == 0 {
				out.SetBool(false)
			} else {
				out.SetBool(true)
			}
		case reflect.Slice:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// check all keys are number, and find the max key.
			max := -1
			for key := range v {
				i, err := strconv.ParseInt(key, 10, 0)
				if err != nil {
					return dec.withErrorContext(&UnmarshalTypeError{Value: "number", Type: reflect.TypeOf("")})
				}
				if int(i) > max {
					max = int(i)
				}
			}
			// Grow slice if necessary
			if max < 0 || max+1 > out.Cap() {
				newout := reflect.MakeSlice(out.Type(), max+1, max+1)
				out.Set(newout)
			} else {
				// fill zero
				zero := reflect.Zero(out.Type().Elem())
				for i := 0; i <= max; i++ {
					out.Index(i).Set(zero)
				}
			}
			out.SetLen(max + 1)
			for key, vv := range v {
				i, _ := strconv.ParseInt(key, 10, 0) // err has been already checked, so no need to check here.
				if err := dec.decode(vv, out.Index(int(i))); err != nil {
					return err
				}
			}
		case reflect.Array:
			// PHP flavored http://php.net/manual/en/language.types.array.php#language.types.array.casting
			// fill zero
			zero := reflect.Zero(out.Type().Elem())
			for i := 0; i < out.Len(); i++ {
				out.Index(i).Set(zero)
			}

			for key, vv := range v {
				i, err := strconv.ParseInt(key, 10, 0)
				if err != nil {
					return dec.withErrorContext(&UnmarshalTypeError{Value: "number", Type: reflect.TypeOf("")})
				}
				if int(i) >= out.Len() {
					continue
				}
				if err := dec.decode(vv, out.Index(int(i))); err != nil {
					return err
				}
			}
		}
	default:
		panic(fmt.Sprintf("unknown type: %v", reflect.TypeOf(v)))
	}
	return nil
}

// convertNumber converts the number literal s to a float64 or a Number
// depending on the setting of dec.useNumber.
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

// convertNumber2Float64 converts number literals in v recursively.
func (dec *Decoder) convertNumber2Float64(v interface{}) (interface{}, error) {
	switch v := v.(type) {
	case Number:
		f, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			return nil, &UnmarshalTypeError{Value: "number " + string(v), Type: reflect.TypeOf(0.0)}
		}
		return f, nil
	case []interface{}:
		for i, vv := range v {
			var err error
			v[i], err = dec.convertNumber2Float64(vv)
			if err != nil {
				return nil, err
			}
		}
	case map[string]interface{}:
		for key, vv := range v {
			var err error
			v[key], err = dec.convertNumber2Float64(vv)
			if err != nil {
				return nil, err
			}
		}
	}
	return v, nil
}

// parse numbers as interger values.
func parseInt(s string, t reflect.Type) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		if reflect.Zero(t).OverflowInt(n) {
			return 0, errors.New("overflow")
		}
		return n, nil
	}

	// PHP flavored http://php.net/manual/en/language.types.integer.php#language.types.integer.casting
	// convert floating point numbers to integer
	if t.Kind() == reflect.Int64 {
		// Go's built-in float64 doesn't have enough precision
		// to present int64 values.
		// so we use math/big.Float here.
		f, _, err := big.ParseFloat(s, 10, 64, big.ToZero)
		if err != nil {
			return 0, err
		}
		n, acc := f.Int64()
		if acc != big.Exact {
			return 0, errors.New("overflow")
		}
		return n, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if f < math.MinInt64 || f > math.MaxInt64 {
		return 0, errors.New("overflow")
	}
	n = int64(f)
	if reflect.Zero(t).OverflowInt(n) {
		return 0, errors.New("overflow")
	}
	return n, nil
}

// parse numbers as unsigned interger values.
func parseUint(s string, t reflect.Type) (uint64, error) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err == nil {
		if reflect.Zero(t).OverflowUint(n) {
			return 0, errors.New("overflow")
		}
		return n, nil
	}

	// PHP flavored http://php.net/manual/en/language.types.integer.php#language.types.integer.casting
	// convert floating point numbers to integer
	if t.Kind() == reflect.Uint64 {
		// Go's built-in float64 doesn't have enough precision
		// to present int64 values.
		// so we use math/big.Float here.
		f, _, err := big.ParseFloat(s, 10, 64, big.ToZero)
		if err != nil {
			return 0, err
		}
		n, acc := f.Uint64()
		if acc != big.Exact {
			return 0, errors.New("overflow")
		}
		return n, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if f < 0 || f > math.MaxUint64 {
		return 0, errors.New("overflow")
	}
	n = uint64(f)
	if reflect.Zero(t).OverflowUint(n) {
		return 0, errors.New("overflow")
	}
	return n, nil
}

// DisallowUnknownFields causes the Decoder to return an error
// when the destination is a struct and the input contains object keys
// which do not match any non-ignored, exported fields in the destination.
func (dec *Decoder) DisallowUnknownFields() {
	dec.disallowUnknownFields = true
}

// More reports whether there is another element in the current array or object being parsed.
func (dec *Decoder) More() bool {
	return dec.dec.More()
}

// Token returns the next JSON token in the input stream.
// At the end of the input stream, Token returns nil, io.EOF.
func (dec *Decoder) Token() (json.Token, error) {
	return dec.dec.Token()
}

// UseNumber causes the Decoder to unmarshal a number into an interface{} as a Number instead of as a float64.
func (dec *Decoder) UseNumber() {
	dec.useNumber = true
}

// Unmarshal parses the JSON-encoded data and stores the result
// in the value pointed to by v. If v is nil or not a pointer,
// Unmarshal returns an InvalidUnmarshalError.
//
// phperjson.Unmashal works in the same way as json.Unmashal,
// but it is useful for dealing with PHP-encoded JSON.
// http://php.net/manual/en/function.json-encode.php
//
// Unlike json.Unmarshal, phperjson.Unmarshal can unmashal a JSON object into a slice.
// The key of the object is interpreted as an index of the slice.
// It is use for decoding PHP-encoded JSON with JSON_FORCE_OBJECT option.
//
// And more, you can use ``Type Juggling'' of PHP.
// For example, phperjson.Unmarshal can unmashal a JSON string into int,
// if the string can be parsed as number.
// See http://php.net/manual/en/language.types.type-juggling.php for more detail.
func Unmarshal(data []byte, v interface{}) error {
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
