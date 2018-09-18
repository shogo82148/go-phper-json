package phperjson

import (
	"bytes"
	"encoding/json"
	"io"
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

func (dec *Decoder) Decode(v interface{}) error {
	var iv interface{}
	if err := dec.dec.Decode(&iv); err != nil {
		return err
	}
	switch vv := iv.(type) {
	case bool:
		if v, ok := v.(*bool); ok {
			*v = vv
		}
	}
	return nil
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
