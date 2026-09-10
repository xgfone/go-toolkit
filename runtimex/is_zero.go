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

import "reflect"

// EqualZero reports whether value equals the zero value of T using ==.
// Unlike IsZero, it does not call an IsZero method. If T is an interface,
// only a nil interface is equal to zero; an interface containing a typed
// nil is not.
func EqualZero[T comparable](value T) bool {
	var zero T
	return value == zero
}

// IsZero reports whether the value is ZERO of type.
func IsZero(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case bool:
		return !v
	case string:
		return v == ""
	case int:
		return v == 0
	case int8:
		return v == 0
	case int16:
		return v == 0
	case int32:
		return v == 0
	case int64:
		return v == 0
	case uint:
		return v == 0
	case uint8:
		return v == 0
	case uint16:
		return v == 0
	case uint32:
		return v == 0
	case uint64:
		return v == 0
	case uintptr:
		return v == 0
	case float32:
		return v == 0
	case float64:
		return v == 0
	case []byte:
		return v == nil
	case interface{ IsZero() bool }:
		switch rvalue := reflect.ValueOf(value); rvalue.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
			reflect.Pointer, reflect.Slice:
			if rvalue.IsNil() {
				return true
			}
		}
		return v.IsZero()
	default:
		rvalue := reflect.ValueOf(value)
		return !rvalue.IsValid() || rvalue.IsZero()
	}
}
