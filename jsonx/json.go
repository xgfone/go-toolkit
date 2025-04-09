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
	// Marshal is used to marshal a value by json to a writer.
	//
	// Default: use json.Encoder
	Marshal func(out io.Writer, in any) error = marshal

	// Unmarshal is used to unmarshal a value by json from a reader.
	//
	// Default: use json.Decoder
	Unmarshal func(out any, in io.Reader) error = unmarshal
)

// UnmarshalBytes is similar to Unmarshal, but decodes a value directly
// from a []byte instead of reading from an io.Reader.
func UnmarshalBytes(data []byte, dst any) error {
	return Unmarshal(dst, bytes.NewReader(data))
}

// MarshalBytes is similar to Marshal, but encodes a value directly
// to a []byte instead of writing to an io.Writer.
func MarshalBytes(v any) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(256)
	err := Marshal(&buf, v)
	data := bytes.TrimRight(buf.Bytes(), "\n")
	return data, err
}

// MarshalString is similar to Marshal, but encodes a value directly
// to a string instead of writing to an io.Writer.
func MarshalString(v any) (string, error) {
	var buf strings.Builder
	buf.Grow(256)
	err := Marshal(&buf, v)
	data := strings.TrimRight(buf.String(), "\n")
	return data, err
}
