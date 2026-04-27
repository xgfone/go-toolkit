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

package structs

import (
	"errors"
	"reflect"
	"strings"
	"sync"
)

const (
	SetFlagOnlyZero int8 = 0
	SetFlagForce    int8 = 1
)

type mapKey struct {
	typ reflect.Type
	tag string
}

type Struct struct {
	Fields []Field
}

type Field struct {
	Name    string
	Default string

	// flag: SetFlagOnlyZero or SetFlagForce
	SetValue func(root reflect.Value, s string, flag int8) error
}

var structs sync.Map // map[mapKey]*Struct

func Parse(t reflect.Type, tag string) (s *Struct) {
	key := mapKey{typ: t, tag: tag}
	if v, ok := structs.Load(key); ok {
		return v.(*Struct)
	}

	actual, _ := structs.LoadOrStore(key, parse(t, tag))
	return actual.(*Struct)
}

func parse(t reflect.Type, tag string) (s *Struct) {
	fields := parseWithParent(t, tag, nil)
	return &Struct{Fields: fields}
}

func parseWithParent(t reflect.Type, tag string, parent []int) (fields []Field) {
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		var name string
		if tag != "" {
			name = parseTagName(sf.Tag.Get(tag))
			if name == "-" {
				continue
			}
		}

		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		index := appendIndex(parent, i)

		// Expand struct-typed fields by recursively parsing their
		// sub-fields so they are discoverable by callers.
		//
		// A struct field is expanded only when it has at least one exported
		// sub-field (hasExportedField) AND it is either:
		//   - an anonymous embedded field, or
		//   - an exported named field.
		//
		// Anonymous embedded structs are checked before the IsExported test
		// below because reflect.FieldByIndex permits traversing through an
		// unexported anonymous struct to reach its exported sub-fields.
		//
		// Struct-typed fields without exported sub-fields fall through to
		// the normal path below and are added as a single opaque field.
		if ft.Kind() == reflect.Struct && hasExportedField(ft) && (sf.Anonymous || sf.IsExported()) {
			fields = append(fields, parseWithParent(ft, tag, index)...)
			continue
		}

		if !sf.IsExported() {
			continue
		}

		if name == "" {
			name = sf.Name
		}

		fields = append(fields, Field{
			Name:     name,
			Default:  sf.Tag.Get("default"),
			SetValue: makeValueSetter(index, sf.Type),
		})
	}

	return
}

// hasExportedField reports whether the struct type t has at least one
// exported field. It is used to decide whether to expand an anonymous
// struct field: only structs with at least one exported field are
// candidates for expansion.
func hasExportedField(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() {
			return true
		}
	}
	return false
}

func parseTagName(tag string) string {
	if tag == "" {
		return ""
	}

	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	return tag
}

func appendIndex(parent []int, i int) []int {
	index := make([]int, len(parent)+1)
	copy(index, parent)
	index[len(parent)] = i
	return index
}

func makeValueSetter(index []int, rtype reflect.Type) func(root reflect.Value, s string, flag int8) error {
	setter := CompileSetter(rtype)
	return func(root reflect.Value, s string, flag int8) error {
		fv, err := fieldByIndexAlloc(root, index)
		if err != nil {
			return err
		}
		if flag == SetFlagOnlyZero && !fv.IsZero() {
			return nil
		}
		return setter(rtype, fv, s)
	}
}

func fieldByIndexAlloc(v reflect.Value, index []int) (reflect.Value, error) {
	cur := v

	for i, x := range index {
		f := cur.Field(x)

		if i == len(index)-1 {
			return f, nil
		}

		// When traversing through an anonymous (embedded) pointer field,
		// reflect.Value.Field returns a value that is NOT addressable/settable
		// even if the parent struct is addressable. We must use FieldByIndex
		// on the original addressable value instead.
		if f.Kind() == reflect.Pointer {
			if f.Type().Elem().Kind() != reflect.Struct {
				return reflect.Value{}, errors.New("non-struct pointer in field path")
			}

			if f.IsNil() {
				// Allocate through the original struct by creating the
				// intermediate pointer at index[:i+1].
				f = v.FieldByIndex(index[:i+1])
				f.Set(reflect.New(f.Type().Elem()))
				f = f.Elem()
			} else {
				f = f.Elem()
			}

			cur = f
			continue
		}

		if f.Kind() != reflect.Struct {
			return reflect.Value{}, errors.New("invalid field path")
		}

		cur = f
	}

	return reflect.Value{}, errors.New("empty field index")
}
