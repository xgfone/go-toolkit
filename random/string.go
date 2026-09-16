// Copyright 2024 xgfone
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

package random

import (
	"sync/atomic"
	"unsafe"
)

// Pre-define some charsets to generate the random string.
const (
	NumCharset      = "0123456789"
	HexCharset      = NumCharset + "abcdefABCDEF"
	HexLowerCharset = NumCharset + "abcdef"
	HexUpperCharset = NumCharset + "ABCDEF"

	AlphaLowerCharset = "abcdefghijklmnopqrstuvwxyz"
	AlphaUpperCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	AlphaCharset      = AlphaLowerCharset + AlphaUpperCharset

	AlphaNumLowerCharset = NumCharset + AlphaLowerCharset
	AlphaNumUpperCharset = NumCharset + AlphaUpperCharset
	AlphaNumCharset      = NumCharset + AlphaCharset
)

var defaultCharset atomic.Value

func init() {
	defaultCharset.Store(AlphaNumLowerCharset)
}

// DefaultCharset returns the default charset, initially [AlphaNumLowerCharset].
// It is safe to call concurrently with [SetDefaultCharset].
func DefaultCharset() string {
	return defaultCharset.Load().(string)
}

// SetDefaultCharset sets the default charset used by callers of [DefaultCharset].
// It panics if charset is empty and is safe to call concurrently.
func SetDefaultCharset(charset string) {
	if charset == "" {
		panic("random.SetDefaultCharset: charset must not be empty")
	}
	defaultCharset.Store(charset)
}

// String generates a random string with the length from the given charsets.
func String(n int, charset string) string {
	buf := make([]byte, n)
	Bytes(buf, charset)
	return unsafe.String(unsafe.SliceData(buf), len(buf))
}

// Bytes generates a random string with the length from the given charsets into buf.
func Bytes(buf []byte, charset string) {
	nlen := len(charset)
	for i, _len := 0, len(buf); i < _len; i++ {
		buf[i] = charset[IntN(nlen)]
	}
}
