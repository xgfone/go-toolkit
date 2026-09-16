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
	"sync/atomic"
	"unicode/utf8"
)

var (
	phoneDesensitizer    atomic.Value
	emailDesensitizer    atomic.Value
	shortDesensitizer    atomic.Value
	defaultDesensitizer  atomic.Value
	passwordDesensitizer atomic.Value
)

// Keep the stored type fixed when a setter receives a different implementation.
type desensitizerValue struct{ desensitizer Desensitizer }

func init() {
	SetPhoneDesensitizer(new(NewDesensitizer(3, 4)))
	SetShortDesensitizer(new(NewDesensitizer(2, 2)))
	SetDefaultDesensitizer(new(NewDesensitizer(4, 4)))
	SetPasswordDesensitizer(new(NewDesensitizer(0, 0).WithChars("********")))
	SetEmailDesensitizer(NewEmailDesensitizer(NewDesensitizer(1, 0)))
}

// PhoneDesensitizer returns the current phone desensitizer, initially retaining
// 3 leading and 4 trailing runes. It is safe to call concurrently with
// [SetPhoneDesensitizer].
func PhoneDesensitizer() Desensitizer {
	return phoneDesensitizer.Load().(desensitizerValue).desensitizer
}

// SetPhoneDesensitizer replaces the phone desensitizer.
//
// It panics if d is nil, including a typed nil. It is safe to call
// concurrently, but does not make d itself safe for concurrent use.
// Previously returned instances are unaffected.
func SetPhoneDesensitizer(d Desensitizer) {
	storeDesensitizer(&phoneDesensitizer, d)
}

// EmailDesensitizer returns the current email desensitizer.
//
// Initially it retains the first rune of the local part and the entire domain,
// replacing the remaining local part with "****". A single-rune local part is
// replaced entirely.
//
// The initial implementation parses a single address, discarding display names
// and comments. Empty input stays empty; unparseable input becomes "****".
// It is safe to call concurrently with [SetEmailDesensitizer].
func EmailDesensitizer() Desensitizer {
	return emailDesensitizer.Load().(desensitizerValue).desensitizer
}

// SetEmailDesensitizer replaces the email desensitizer.
//
// It panics if d is nil, including a typed nil. It is safe to call
// concurrently, but does not make d itself safe for concurrent use.
// Previously returned instances are unaffected.
func SetEmailDesensitizer(d Desensitizer) {
	storeDesensitizer(&emailDesensitizer, d)
}

// ShortDesensitizer returns the current short-string desensitizer, initially
// retaining 2 leading and 2 trailing runes. It is safe to call concurrently
// with [SetShortDesensitizer].
func ShortDesensitizer() Desensitizer {
	return shortDesensitizer.Load().(desensitizerValue).desensitizer
}

// SetShortDesensitizer replaces the short-string desensitizer.
//
// It panics if d is nil, including a typed nil. It is safe to call
// concurrently, but does not make d itself safe for concurrent use.
// Previously returned instances are unaffected.
func SetShortDesensitizer(d Desensitizer) {
	storeDesensitizer(&shortDesensitizer, d)
}

// DefaultDesensitizer returns the current default desensitizer, initially
// retaining 4 leading and 4 trailing runes. It is safe to call concurrently
// with [SetDefaultDesensitizer].
func DefaultDesensitizer() Desensitizer {
	return defaultDesensitizer.Load().(desensitizerValue).desensitizer
}

// SetDefaultDesensitizer replaces the default desensitizer.
//
// It panics if d is nil, including a typed nil. It is safe to call
// concurrently, but does not make d itself safe for concurrent use.
// Previously returned instances are unaffected.
func SetDefaultDesensitizer(d Desensitizer) {
	storeDesensitizer(&defaultDesensitizer, d)
}

// PasswordDesensitizer returns the current password desensitizer, initially
// replacing the entire string with "********". It is safe to call concurrently
// with [SetPasswordDesensitizer].
func PasswordDesensitizer() Desensitizer {
	return passwordDesensitizer.Load().(desensitizerValue).desensitizer
}

// SetPasswordDesensitizer replaces the password desensitizer.
//
// It panics if d is nil, including a typed nil. It is safe to call
// concurrently, but does not make d itself safe for concurrent use.
// Previously returned instances are unaffected.
func SetPasswordDesensitizer(d Desensitizer) {
	storeDesensitizer(&passwordDesensitizer, d)
}

func storeDesensitizer(value *atomic.Value, d Desensitizer) {
	checkNonNil(d, "stringx.Desensitizer: desensitizer must not be nil")
	value.Store(desensitizerValue{desensitizer: d})
}

// Desensitizer desensitizes a string.
type Desensitizer interface {
	Desensitize(string) string
}

// DesensitizerFunc adapts a function to [Desensitizer].
type DesensitizerFunc func(string) string

// Desensitize calls f(s).
func (f DesensitizerFunc) Desensitize(s string) string { return f(s) }

// MaskDesensitizer replaces the middle of a string while retaining leading and
// trailing runes. Its methods do not modify the receiver and may be called
// concurrently. The zero value replaces non-empty strings with "****".
type MaskDesensitizer struct {
	left  int
	right int
	chars string
}

// NewDesensitizer returns a desensitizer retaining left leading and right
// trailing runes, with "****" as the replacement.
//
// It panics if either count is negative.
func NewDesensitizer(left, right int) MaskDesensitizer {
	if left < 0 || right < 0 {
		panic("stringx.NewDesensitizer: rune counts must not be negative")
	}
	return MaskDesensitizer{left: left, right: right, chars: "****"}
}

// Left returns the number of leading runes to retain.
func (d MaskDesensitizer) Left() int { return d.left }

// Right returns the number of trailing runes to retain.
func (d MaskDesensitizer) Right() int { return d.right }

// Chars returns the configured replacement, which may be empty.
func (d MaskDesensitizer) Chars() string { return d.chars }

// WithLeft returns a copy with the given leading rune count.
//
// It panics if left is negative.
func (d MaskDesensitizer) WithLeft(left int) MaskDesensitizer {
	if left < 0 {
		panic("stringx.MaskDesensitizer.WithLeft: left must not be negative")
	}

	d.left = left
	return d
}

// WithRight returns a copy with the given trailing rune count.
//
// It panics if right is negative.
func (d MaskDesensitizer) WithRight(right int) MaskDesensitizer {
	if right < 0 {
		panic("stringx.MaskDesensitizer.WithRight: right must not be negative")
	}

	d.right = right
	return d
}

// WithChars returns a copy with the given replacement.
//
// An empty replacement preserves empty input and uses "****" for non-empty input.
func (d MaskDesensitizer) WithChars(s string) MaskDesensitizer {
	d.chars = s
	return d
}

// Desensitize returns a desensitized string of s.
//
// If s has no more runes than the sum returned by [MaskDesensitizer.Left] and
// [MaskDesensitizer.Right], it is replaced entirely. If both s and the result of
// [MaskDesensitizer.Chars] are empty, it returns ""; otherwise an empty replacement
// defaults to "****". Rune boundaries, rather than grapheme clusters, are used.
func (d MaskDesensitizer) Desensitize(s string) string {
	chars := d.chars
	if chars == "" {
		if s == "" {
			return ""
		}
		chars = "****"
	}

	total := utf8.RuneCountInString(s)
	// Subtract only after checking left to avoid overflowing left+right.
	if d.left >= total || d.right >= total-d.left {
		return chars
	}

	if d.left == 0 && d.right == 0 {
		return chars
	}

	left, right := 0, len(s)
	var n int
	for i := range s {
		if n == d.left {
			left = i
		}
		if n == total-d.right {
			right = i
			break
		}
		n++
	}

	return s[:left] + chars + s[right:]
}
