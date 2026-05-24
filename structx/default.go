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

package structx

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/xgfone/go-toolkit/internal/structs"
)

var (
	errDefaultNilPointer = errors.New("SetDefault: structptr is nil")
	errDefaultNotStruct  = errors.New("SetDefault: structptr is not a pointer to struct")
)

// SetDefaultAny is like SetDefault, but accepts a value of type any.
//
// structptr must be a non-nil pointer to struct.
func SetDefaultAny(structptr any) (err error) {
	if structptr == nil {
		return errDefaultNilPointer
	}

	rtype := reflect.TypeOf(structptr)
	if rtype.Kind() != reflect.Pointer {
		return errDefaultNotStruct
	}

	rtype = rtype.Elem()
	if rtype.Kind() != reflect.Struct {
		return errDefaultNotStruct
	}

	root := reflect.ValueOf(structptr)
	if root.IsNil() {
		return errDefaultNilPointer
	}

	return _setdefault(rtype, root.Elem())
}

// SetDefault sets the default values of the struct fields tagged with "default".
//
// If a field has a "default" tag and its current value is the zero value
// of its type, SetDefault will set it to the value parsed from the tag.
// Otherwise, the field is left unchanged.
//
// Defaults are intended for fields that can be parsed from strings, such as
// bools, integers, floats, strings, their named types, pointers to those types,
// and fields whose pointer type implements encoding.TextUnmarshaler.
//
// Struct fields are normally expanded when their immediate type has direct
// exported fields, and defaults are applied to their expanded leaf fields.
// Struct fields whose pointer type implements encoding.TextUnmarshaler are
// treated as whole fields automatically and may receive defaults as a whole.
// Nested struct expansion supports struct values and single pointers to
// structs only. Self-referential struct graphs are not cycle detected; avoid
// applying SetDefault to those graphs or make the recursive field type
// implement encoding.TextUnmarshaler so it is treated as a whole field.
func SetDefault[Struct any](structptr *Struct) (err error) {
	if structptr == nil {
		return errDefaultNilPointer
	}

	rtype := reflect.TypeFor[Struct]()
	if rtype.Kind() != reflect.Struct {
		return errDefaultNotStruct
	}

	root := reflect.ValueOf(structptr).Elem()
	return _setdefault(rtype, root)
}

// _setdefault is the backend function implementing the SetDefault functionality,
// designed for convenient replacement during testing.
var _setdefault = setDefault

func setDefault(rtype reflect.Type, root reflect.Value) (err error) {
	for _, f := range structs.StringParser.Parse(rtype, "").Fields {
		if f.Default == "" {
			continue
		}

		rvalue := f.GetField(root)
		if !rvalue.IsZero() {
			continue
		}

		if err = f.SetField(f.Type, rvalue, f.Default); err != nil {
			return fmt.Errorf("%q: %w", f.Name, err)
		}
	}
	return
}
