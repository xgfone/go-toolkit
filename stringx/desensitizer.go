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

package stringx

import (
	"strings"
	"unicode/utf8"
)

var (
	// PhoneDesensitizer is used to desensitize the phone.
	PhoneDesensitizer = NewDesensitizer(3, 4)

	// DefaultDesensitizer is the common desensitizer.
	DefaultDesensitizer = NewDesensitizer(4, 4)
)

type Desensitizer struct {
	// The length of the left undesensitized characters.
	Left int

	// The length of the right undesensitized characters.
	Right int

	// The desensitized string
	//
	// Default: "****"
	Chars string
}

func NewDesensitizer(left, right int) Desensitizer {
	return Desensitizer{Left: left, Right: right, Chars: "****"}
}

// WithChars returns a new string desensitizer with the new desensitization chars.
func (d Desensitizer) WithChars(s string) Desensitizer {
	d.Chars = s
	return d
}

// Desensitize returns a desensitized string of s.
func (d Desensitizer) Desensitize(s string) string {
	if d.Chars == "" {
		d.Chars = "****"
	}

	total := utf8.RuneCountInString(s)
	if total <= d.Left+d.Right {
		return d.Chars
	}

	switch {
	case d.Left <= 0 && d.Right <= 0:
		return d.Chars

	case d.Left <= 0:
		d.Right = total - d.Right

		var n int
		for i := range s {
			if n == d.Right {
				s = d.Chars + s[i:]
			}
			n++
		}

	case d.Right <= 0:
		var n int
		for i := range s {
			if n == d.Left {
				s = s[:i] + d.Chars
			}
			n++
		}

	default:
		d.Right = total - d.Right

		var n, left, right int
		for i := range s {
			if n == d.Left {
				left = i
			}

			if n == d.Right {
				right = i
			}

			n++
		}

		s = strings.Join([]string{s[:left], d.Chars, s[right:]}, "")
	}

	return s
}
