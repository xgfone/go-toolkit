// Copyright 2026 xgfone
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

package strconvx

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func ExampleParseInt64Slice() {
	values, err := ParseInt64Slice("1, -2 , +3")
	fmt.Println(values, err)
	// Output: [1 -2 3] <nil>
}

func ExampleParseUint64Slice() {
	values, err := ParseUint64Slice(" 1, 2 , 18446744073709551615 ")
	fmt.Println(values, err)
	// Output: [1 2 18446744073709551615] <nil>
}

func TestParseInt64Slice(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []int64
	}{
		{"empty", "", nil},
		{"whitespace only", " \t\r\n\u00a0\u2003 ", nil},
		{"single", "42", []int64{42}},
		{"comma separated", "1,2,3", []int64{1, 2, 3}},
		{"whitespace", " \t1 \r\n,\u00a0 -2\u2003, +3 \t", []int64{1, -2, 3}},
		{"zero and signs", "0,-0,+0,-1,+1", []int64{0, 0, 0, -1, 1}},
		{"decimal leading zeros", "010,-020,+030", []int64{10, -20, 30}},
		{"order and duplicates", "3,1,3,2", []int64{3, 1, 3, 2}},
		{"limits", "-9223372036854775808,9223372036854775807", []int64{-1 << 63, 1<<63 - 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInt64Slice(tt.s)
			if err != nil {
				t.Fatalf("ParseInt64Slice(%q): %v", tt.s, err)
			}
			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("ParseInt64Slice(%q) = %#v, want %#v", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseUint64Slice(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []uint64
	}{
		{"empty", "", nil},
		{"whitespace only", " \t\r\n\u00a0\u2003 ", nil},
		{"single", "42", []uint64{42}},
		{"comma separated", "1,2,3", []uint64{1, 2, 3}},
		{"whitespace", " \t1 \r\n,\u00a0 2\u2003, 3 \t", []uint64{1, 2, 3}},
		{"decimal leading zeros", "010,020,030", []uint64{10, 20, 30}},
		{"order and duplicates", "3,1,3,2", []uint64{3, 1, 3, 2}},
		{"limits", "0,9223372036854775808,18446744073709551615", []uint64{0, 1 << 63, 1<<64 - 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUint64Slice(tt.s)
			if err != nil {
				t.Fatalf("ParseUint64Slice(%q): %v", tt.s, err)
			}
			if !slices.Equal(got, tt.want) || (got == nil) != (tt.want == nil) {
				t.Errorf("ParseUint64Slice(%q) = %#v, want %#v", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseIntegerSliceErrors(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		position int
		num      string
	}{
		{"leading comma", ",1", 1, ""},
		{"trailing comma", "1,", 2, ""},
		{"consecutive commas", "1,,2", 2, ""},
		{"whitespace element", "1, \t\u2003,2", 2, ""},
		{"comma only", ",", 1, ""},
		{"invalid first element", "bad,1", 1, "bad"},
		{"invalid last element", "1,2, bad ", 3, "bad"},
		{"first error", "1, bad ,also-bad", 2, "bad"},
		{"internal whitespace", "1,2 3", 2, "2 3"},
		{"float", "1,2.5", 2, "2.5"},
		{"exponent", "1,2e3", 2, "2e3"},
		{"hex prefix", "1,0x10", 2, "0x10"},
		{"underscore", "1,1_000", 2, "1_000"},
		{"fullwidth comma", "1，2", 1, "1，2"},
		{"fullwidth digit", "1,２", 2, "２"},
		{"plus only", "1,+", 2, "+"},
		{"minus only", "1,-", 2, "-"},
		{"double sign", "1,--2", 2, "--2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ints, err := ParseInt64Slice(tt.s)
			if ints != nil {
				t.Errorf("ParseInt64Slice(%q) = %v, want nil", tt.s, ints)
			}
			checkParseError(t, err, "ParseInt", tt.num, tt.position, strconv.ErrSyntax)

			uints, err := ParseUint64Slice(tt.s)
			if uints != nil {
				t.Errorf("ParseUint64Slice(%q) = %v, want nil", tt.s, uints)
			}
			checkParseError(t, err, "ParseUint", tt.num, tt.position, strconv.ErrSyntax)
		})
	}
}

func TestParseInt64SliceRangeErrors(t *testing.T) {
	for _, s := range []string{"9223372036854775808", "-9223372036854775809"} {
		t.Run(s, func(t *testing.T) {
			got, err := ParseInt64Slice("1, " + s + " ,2")
			if got != nil {
				t.Errorf("ParseInt64Slice returned %v, want nil", got)
			}
			checkParseError(t, err, "ParseInt", s, 2, strconv.ErrRange)
		})
	}
}

func TestParseUint64SliceSignAndRangeErrors(t *testing.T) {
	tests := []struct {
		s   string
		err error
	}{
		{"-1", strconv.ErrSyntax},
		{"-0", strconv.ErrSyntax},
		{"+1", strconv.ErrSyntax},
		{"18446744073709551616", strconv.ErrRange},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got, err := ParseUint64Slice("1, " + tt.s + " ,2")
			if got != nil {
				t.Errorf("ParseUint64Slice returned %v, want nil", got)
			}
			checkParseError(t, err, "ParseUint", tt.s, 2, tt.err)
		})
	}
}

func checkParseError(t *testing.T, err error, fn, num string, position int, want error) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}

	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("error = %v, want a wrapped *strconv.NumError", err)
	}
	if numErr.Func != fn || numErr.Num != num {
		t.Errorf("NumError = %#v, want Func=%q and Num=%q", numErr, fn, num)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("element %d:", position)) {
		t.Errorf("error = %v, want element position %d", err, position)
	}
}
