package phperjson

import (
	"bytes"
	"encoding/json"
	"io"
)

// Compact is an alias for json.Compact.
func Compact(dst *bytes.Buffer, src []byte) error {
	return json.Compact(dst, src)
}

// HTMLEscape is an alias for json.HTMLEscape.
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	json.HTMLEscape(dst, src)
}

// Indent is an alias for json.Indent.
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	return json.Indent(dst, src, prefix, indent)
}

// Marshal is an alias for json.Marshal.
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// MarshalIndent is an alias for json.MarshalIndent.
func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// Delim is an alias for json.Delim.
type Delim = json.Delim

// Encoder is an alias for json.Encoder.
type Encoder = json.Encoder

// NewEncoder is an alias for json.NewEncoder.
func NewEncoder(w io.Writer) *Encoder {
	return json.NewEncoder(w)
}

// InvalidUTF8Error is an alias for json.InvalidUTF8Error.
type InvalidUTF8Error = json.InvalidUTF8Error

// InvalidUnmarshalError is an alias for json.InvalidUnmarshalError.
type InvalidUnmarshalError = json.InvalidUnmarshalError

// Marshaler is an alias for json.Marshaler.
type Marshaler = json.Marshaler

// MarshalerError is an alias for json.MarshalerError.
type MarshalerError = json.MarshalerError

// Number is an alias for json.Number.
type Number = json.Number

// RawMessage is an alias for json.RawMessage.
type RawMessage = json.RawMessage

// SyntaxError = json.SyntaxError.
type SyntaxError = json.SyntaxError

// Token is an alias for json.Token.
type Token = json.Token
