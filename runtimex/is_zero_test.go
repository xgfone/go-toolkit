// Copyright 2024~2026 xgfone
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

package runtimex

import "testing"

func TestIsZero(t *testing.T) {
	testbool(t, "nil", IsZero(nil), true)
	testbool(t, "nil", IsZero((*int)(nil)), true)

	testbool(t, "bool", IsZero(false), true)
	testbool(t, "bool", IsZero(true), false)

	testbool(t, "string", IsZero(""), true)
	testbool(t, "string", IsZero("a"), false)

	testbool(t, "bytes", IsZero([]byte(nil)), true)
	testbool(t, "bytes", IsZero([]byte("")), false)

	testbool(t, "__int", IsZero(int(0)), true)
	testbool(t, "__int", IsZero(int(1)), false)
	testbool(t, "_int8", IsZero(int8(0)), true)
	testbool(t, "_int8", IsZero(int8(1)), false)
	testbool(t, "int16", IsZero(int16(0)), true)
	testbool(t, "int16", IsZero(int16(1)), false)
	testbool(t, "int32", IsZero(int32(0)), true)
	testbool(t, "int32", IsZero(int32(1)), false)
	testbool(t, "int64", IsZero(int64(0)), true)
	testbool(t, "int64", IsZero(int64(1)), false)

	testbool(t, "__uint", IsZero(uint(0)), true)
	testbool(t, "__uint", IsZero(uint(1)), false)
	testbool(t, "_uint8", IsZero(uint8(0)), true)
	testbool(t, "_uint8", IsZero(uint8(1)), false)
	testbool(t, "uint16", IsZero(uint16(0)), true)
	testbool(t, "uint16", IsZero(uint16(1)), false)
	testbool(t, "uint32", IsZero(uint32(0)), true)
	testbool(t, "uint32", IsZero(uint32(1)), false)
	testbool(t, "uint64", IsZero(uint64(0)), true)
	testbool(t, "uint64", IsZero(uint64(1)), false)

	testbool(t, "uintptr", IsZero(uintptr(0)), true)
	testbool(t, "uintptr", IsZero(uintptr(1)), false)

	testbool(t, "float32", IsZero(float32(0)), true)
	testbool(t, "float32", IsZero(float32(1)), false)
	testbool(t, "float64", IsZero(float64(0)), true)
	testbool(t, "float64", IsZero(float64(1)), false)

	testbool(t, "interface", IsZero(_iszero(false)), true)
	testbool(t, "interface", IsZero(_iszero(true)), false)

	testbool(t, "reflect", IsZero(any(false)), true)
	testbool(t, "reflect", IsZero(any(true)), false)
}

func testbool(t *testing.T, kind string, value, expect bool) {
	if value != expect {
		t.Errorf("%s: expect %v, but got %v", kind, expect, value)
	}
}

type _iszero bool

func (v _iszero) IsZero() bool { return !bool(v) }

func TestEqualZero(t *testing.T) {
	testbool(t, "zero int", EqualZero(0), true)
	testbool(t, "nonzero int", EqualZero(1), false)
	testbool(t, "empty string", EqualZero(""), true)
	testbool(t, "nonempty string", EqualZero("x"), false)
	testbool(t, "zero array", EqualZero([2]int{}), true)
	testbool(t, "nonzero array", EqualZero([2]int{0, 1}), false)
	testbool(t, "zero struct", EqualZero(struct{ N int }{}), true)
	testbool(t, "nonzero struct", EqualZero(struct{ N int }{N: 1}), false)

	var ptr *int
	testbool(t, "nil pointer", EqualZero(ptr), true)
	testbool(t, "pointer to zero", EqualZero(new(int)), false)

	var value any
	testbool(t, "nil interface", EqualZero(value), true)
	testbool(t, "interface containing zero", EqualZero[any](0), false)
	testbool(t, "interface containing typed nil", EqualZero[any](ptr), false)
	testbool(t, "interface containing slice", EqualZero[any]([]int{}), false)
	testbool(t, "struct containing interface", EqualZero(struct{ V any }{V: []int{}}), false)
}

type equalZeroOverride int

func (v equalZeroOverride) IsZero() bool { return v == 1 }

func TestEqualZeroIgnoresIsZeroMethod(t *testing.T) {
	testbool(t, "underlying zero", EqualZero(equalZeroOverride(0)), true)
	testbool(t, "custom zero", EqualZero(equalZeroOverride(1)), false)
	testbool(t, "IsZero custom zero", IsZero(equalZeroOverride(1)), true)
}
