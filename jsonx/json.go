// Copyright 2025 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package jsonx provides some extra json functions.
package jsonx

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

var (
	marshaler   func(out io.Writer, in any) error
	unmarshaler func(out any, in io.Reader) error
)

func init() {
	SetMarshalWriterFunc(func(out io.Writer, in any) error {
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		return enc.Encode(in)
	})

	SetUnmarshalReaderFunc(func(out any, in io.Reader) error {
		return json.NewDecoder(in).Decode(out)
	})
}

// SetMarshalWriterFunc sets the marshal writer function
// to marshal a value to a writer.
func SetMarshalWriterFunc(f func(out io.Writer, in any) error) {
	if f == nil {
		panic("jsonx: marshal writer function is nil")
	}
	marshaler = f
}

// SetUnmarshalReaderFunc sets the unmarshal reader function
// to unmarshal a value from a reader.
func SetUnmarshalReaderFunc(f func(out any, in io.Reader) error) {
	if f == nil {
		panic("jsonx: unmarshal reader function is nil")
	}
	unmarshaler = f
}

// Marshal is short for MarshalBytes.
func Marshal(in any) ([]byte, error) {
	return MarshalBytes(in)
}

// Unmarshal is short for UnmarshalBytes.
func Unmarshal(in []byte, out any) error {
	return UnmarshalBytes(in, out)
}

// UnmarshalBytes is similar to UnmarshalReader, but unmarshals a value directly
// from a []byte instead of reading from an io.Reader.
func UnmarshalBytes(in []byte, out any) error {
	return UnmarshalReader(out, bytes.NewReader(in))
}

// UnmarshalString is similar to UnmarshalReader, but unmarshals a value directly
// from a string instead of reading from an io.Reader.
func UnmarshalString(in string, out any) error {
	return UnmarshalReader(out, strings.NewReader(in))
}

// MarshalBytes is similar to MarshalWriter, but marshals a value directly
// to a []byte instead of writing to an io.Writer.
func MarshalBytes(v any) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(256)
	err := MarshalWriter(&buf, v)
	data := bytes.TrimRight(buf.Bytes(), "\n")
	return data, err
}

// MarshalString is eqaul to MarshalStringWithCap(v, 256).
func MarshalString(v any) (string, error) {
	return MarshalStringWithCap(v, 256)
}

// MarshalString is similar to MarshalWriter, but marshals a value directly
// to a string instead of writing to an io.Writer.
func MarshalStringWithCap(v any, cap int) (string, error) {
	var buf strings.Builder
	buf.Grow(cap)
	err := MarshalWriter(&buf, v)
	data := strings.TrimRight(buf.String(), "\n")
	return data, err
}

// MarshalWriter marshals any value in to a writer out.
func MarshalWriter(out io.Writer, in any) error {
	return marshaler(out, in)
}

// UnmarshalReader unmarshals a value out directly from a reader in.
func UnmarshalReader(out any, in io.Reader) error {
	return unmarshaler(out, in)
}
