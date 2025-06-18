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
