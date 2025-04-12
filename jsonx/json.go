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
	"io"
	"strings"
)

var (
	marshaler   func(out io.Writer, in any) error
	unmarshaler func(out any, in io.Reader) error
)

func init() {
	SetMarshalWriterFunc(marshal)
	SetUnmarshalReaderFunc(unmarshal)
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

// Marshal is short for MarshalWriter.
func Marshal(out io.Writer, in any) error {
	return MarshalWriter(out, in)
}

// Unmarshal is short for UnmarshalReader.
func Unmarshal(out any, in io.Reader) error {
	return UnmarshalReader(out, in)
}

// MarshalWriter marshals any value to a writer.
func MarshalWriter(out io.Writer, in any) error {
	return marshaler(out, in)
}

// UnmarshalReader unmarshals a value directly from a reader.
func UnmarshalReader(out any, in io.Reader) error {
	return unmarshaler(out, in)
}

// UnmarshalBytes is similar to Unmarshal, but unmarshals a value directly
// from a []byte instead of reading from an io.Reader.
func UnmarshalBytes(data []byte, dst any) error {
	return Unmarshal(dst, bytes.NewReader(data))
}

// UnmarshalString is similar to Unmarshal, but unmarshals a value directly
// from a string instead of reading from an io.Reader.
func UnmarshalString(data string, dst any) error {
	return Unmarshal(dst, strings.NewReader(data))
}

// MarshalBytes is similar to Marshal, but marshals a value directly
// to a []byte instead of writing to an io.Writer.
func MarshalBytes(v any) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(256)
	err := Marshal(&buf, v)
	data := bytes.TrimRight(buf.Bytes(), "\n")
	return data, err
}

// MarshalString is similar to Marshal, but marshals a value directly
// to a string instead of writing to an io.Writer.
func MarshalString(v any) (string, error) {
	var buf strings.Builder
	buf.Grow(256)
	err := Marshal(&buf, v)
	data := strings.TrimRight(buf.String(), "\n")
	return data, err
}
