package phperjson

import (
	"encoding/json"
	"io"
)

type Decoder struct{}

func NewDecoder(r io.Reader) *Decoder {
	return nil
}

func (dec *Decoder) Buffered() io.Reader {
	return nil
}

func (dec *Decoder) Decode(v interface{}) error {
	return nil
}

func (dec *Decoder) DisallowUnknownFields() {
}

func (dec *Decoder) More() bool {
	return false
}

func (dec *Decoder) Token() (json.Token, error) {
	return nil, nil
}

func (dec *Decoder) UseNumber() {}

func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
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
