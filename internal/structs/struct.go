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
	"reflect"
	"strings"
	"sync"
)

type mapKey struct {
	vtype reflect.Type
	stype reflect.Type
	tag   string
}

type Struct[T any] struct {
	Fields []Field[T]
}

type Field[T any] struct {
	Type    reflect.Type
	Name    string
	Default string

	SetField SetterFunc[T]
	GetField FieldGetter
	GetValue func(map[string]any) any // Missing values are returned as nil.
}

type FieldGetter func(root reflect.Value) reflect.Value

func (f *Field[T]) SetValue(root reflect.Value, value T) error {
	rvalue := f.GetField(root)
	return f.SetField(f.Type, rvalue, value)
}

var structs sync.Map // map[mapKey]*Struct

func Parse[T any](t reflect.Type, tag string, compileSetter SetterCompiler[T]) (s *Struct[T]) {
	key := mapKey{vtype: reflect.TypeFor[T](), stype: t, tag: tag}
	if v, ok := structs.Load(key); ok {
		return v.(*Struct[T])
	}

	parser := _Parser[T]{CompileSetter: compileSetter, Tag: tag}
	actual, _ := structs.LoadOrStore(key, parser.Parse(t))
	return actual.(*Struct[T])
}

type _Parser[T any] struct {
	CompileSetter SetterCompiler[T]

	Tag string
}

func (p *_Parser[T]) Parse(t reflect.Type) *Struct[T] {
	fields := p.parse(t, nil, nil)
	return &Struct[T]{Fields: fields}
}

func (p *_Parser[T]) parse(t reflect.Type, parentIndex []int, parentNames []string) (fields []Field[T]) {
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		var name string
		if p.Tag != "" {
			name = parseTagName(sf.Tag.Get(p.Tag))
			if name == "-" {
				continue
			}
		}
		if name == "" {
			name = sf.Name
		}

		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		index := appendSlice(parentIndex, i)

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
			var names []string
			if sf.Anonymous {
				names = parentNames
			} else {
				names = appendSlice(parentNames, name)
			}
			fields = append(fields, p.parse(ft, index, names)...)
			continue
		}

		if !sf.IsExported() {
			continue
		}

		fields = append(fields, Field[T]{
			Name:     name,
			Type:     sf.Type,
			Default:  sf.Tag.Get("default"),
			SetField: p.CompileSetter(sf.Type),
			GetField: makeFieldGetter(index),
			GetValue: makeMapValueGetter(appendSlice(parentNames, name)),
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

func appendSlice[T any](parent []T, i T) []T {
	s := make([]T, len(parent)+1)
	copy(s, parent)
	s[len(parent)] = i
	return s
}

func makeFieldGetter(index []int) FieldGetter {
	return func(root reflect.Value) reflect.Value {
		return fieldByIndexAlloc(root, index)
	}
}

func makeMapValueGetter(names []string) func(map[string]any) any {
	return func(m map[string]any) any {
		if m == nil {
			return nil
		}

		for i := range names {
			name := names[i]

			if i == len(names)-1 {
				return m[name]
			}

			if v, ok := m[name].(map[string]any); ok {
				m = v
			} else {
				return nil
			}
		}

		return nil
	}
}

func fieldByIndexAlloc(v reflect.Value, index []int) reflect.Value {
	cur := v

	for i, x := range index {
		f := cur.Field(x)
		if i == len(index)-1 {
			return f
		}

		// When traversing through an anonymous (embedded) pointer field,
		// reflect.Value.Field returns a value that is NOT addressable/settable
		// even if the parent struct is addressable. We must use FieldByIndex
		// on the original addressable value instead.
		if f.Kind() == reflect.Pointer {
			if f.Type().Elem().Kind() != reflect.Struct {
				panic("non-struct pointer in field path")
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
			panic("invalid field path")
		}

		cur = f
	}

	panic("empty field index")
}
