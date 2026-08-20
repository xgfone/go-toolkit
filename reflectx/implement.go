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

package reflectx

import (
	"reflect"
	"sync"
)

var typeImplements = new(sync.Map)

type mapkey struct {
	source reflect.Type
	target reflect.Type
}

// Implements checks if typ implements the target interface type.
// The result is cached to improve performance for repeated checks.
//
// Note: target must be an interface type. Or, it will panic.
func Implements(typ, target reflect.Type) bool {
	key := mapkey{source: typ, target: target}
	if value, ok := typeImplements.Load(key); ok {
		return value.(bool)
	}

	ok := typ.Implements(target)
	actual, _ := typeImplements.LoadOrStore(key, ok)
	return actual.(bool)
}

// As returns the first value in source's unwrap chain that can be asserted to Target.
// The chain consists of source followed by the values obtained by repeatedly calling
// Unwrap() Source. The method's return type must be exactly Source.
//
// If Source is a concrete type, only source itself is examined.
// If no value matches, As returns the zero value of Target and false.
func As[Target any, Source any](source Source) (Target, bool) {
	sourceType := reflect.TypeFor[Source]()
	if sourceType.Kind() != reflect.Interface {
		target, ok := any(source).(Target)
		return target, ok
	}

	var zero Target

	targetType := reflect.TypeFor[Target]()
	if targetType.Kind() != reflect.Interface && !targetType.Implements(sourceType) {
		return zero, false
	}

	src := any(source)
	for {
		switch t := src.(type) {
		case Target:
			return t, true

		case interface{ Unwrap() Source }:
			src = t.Unwrap()

		default:
			return zero, false
		}
	}
}
