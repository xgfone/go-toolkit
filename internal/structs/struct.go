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

type (
	ValueGetter func(map[string]any) any
	FieldGetter func(root reflect.Value) reflect.Value

	FieldCompiler[Data any] func(reflect.StructField) Data
	OpaqueFieldFunc         func(reflect.StructField) bool
)

type Struct[Data any] struct {
	Fields []Field[Data]
}

type Field[Data any] struct {
	Type reflect.Type
	Name string
	Data Data

	// Read-Only
	Names   []string
	Indexes []int
}

// Missing values are returned as nil.
func (f *Field[Data]) GetValue(m map[string]any) any {
	return getValueFromMap(m, f.Names)
}

// GetField returns the field value, allocating nil intermediate struct
// pointers while traversing the field path.
//
// Notice: Use GetFieldByIndex with alloc=false for non-allocating lookup.
func (f *Field[Data]) GetField(root reflect.Value) reflect.Value {
	return GetFieldByIndex(root, f.Indexes, true)
}

type mapKey struct {
	typ reflect.Type
	tag string
}

type Parser[Data any] struct {
	structs sync.Map // map[mapKey]*Struct[Data]

	compileField  FieldCompiler[Data]
	fieldIsOpaque OpaqueFieldFunc
}

func NewParser[Data any](compile FieldCompiler[Data], isOpaque OpaqueFieldFunc) *Parser[Data] {
	if compile == nil {
		panic("structs.NewParser: field compile function is nil")
	}
	return &Parser[Data]{compileField: compile, fieldIsOpaque: isOpaque}
}

func (p *Parser[Data]) Parse(t reflect.Type, tag string) (s *Struct[Data]) {
	key := mapKey{typ: t, tag: tag}
	if v, ok := p.structs.Load(key); ok {
		return v.(*Struct[Data])
	}

	actual, _ := p.structs.LoadOrStore(key, p._Parse(t, tag))
	return actual.(*Struct[Data])
}

func (p *Parser[Data]) _Parse(t reflect.Type, tag string) *Struct[Data] {
	fields := p.parse(t, nil, nil, tag)
	return &Struct[Data]{Fields: fields}
}

func (p *Parser[Data]) parse(t reflect.Type, parentIndex []int, parentNames []string, tag string) (fields []Field[Data]) {
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		var name string
		var opaque bool
		if tag != "" {
			name, opaque = parseTag(sf.Tag.Get(tag))
			if name == "-" {
				continue
			}
		}
		if name == "" {
			name = sf.Name
		}

		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		index := appendSlice(parentIndex, i)
		names := appendSlice(parentNames, name)

		// Expand struct-typed fields by recursively parsing their
		// sub-fields so they are discoverable by callers.
		//
		// A struct field is expanded only when its immediate type has at
		// least one direct exported field (hasExportedField) AND it is either:
		//   - an anonymous embedded field, or
		//   - an exported named field.
		//
		// Anonymous embedded value structs are checked before the IsExported
		// test below because reflect.FieldByIndex permits traversing through
		// an unexported anonymous value struct to reach its direct exported
		// fields. This does not recursively pierce additional hidden
		// anonymous fields; those remain intentional visibility boundaries.
		// Unexported anonymous pointer structs are not expanded because nil
		// values cannot be allocated safely via reflection.
		//
		// Struct-typed fields without exported sub-fields, fields marked
		// opaque, and fields whose type is considered opaque by the parser
		// fall through to the normal path below and are added as single
		// fields when exported.
		if !opaque && ft.Kind() == reflect.Struct && p.canExpand(sf, ft) {
			if sf.Anonymous {
				names = parentNames
			}
			fields = append(fields, p.parse(ft, index, names, tag)...)
			continue
		}

		if !sf.IsExported() {
			continue
		}

		fields = append(fields, Field[Data]{
			Name: name,
			Type: sf.Type,
			Data: p.compileField(sf),

			Names:   names,
			Indexes: index,
		})
	}
	return
}

func (p *Parser[Data]) canExpand(sf reflect.StructField, ft reflect.Type) bool {
	if !hasExportedField(ft) {
		return false
	}
	if !sf.Anonymous && !sf.IsExported() {
		return false
	}
	if sf.Anonymous && !sf.IsExported() && sf.Type.Kind() == reflect.Pointer {
		return false
	}
	return !sf.IsExported() || p.fieldIsOpaque == nil || !p.fieldIsOpaque(sf)
}

// hasExportedField reports whether the struct type t has at least one direct
// exported field. It is intentionally shallow: unexported embedded fields are
// treated as visibility boundaries for deciding whether their parent should be
// expanded.
func hasExportedField(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() {
			return true
		}
	}
	return false
}

func parseTag(tag string) (name string, opaque bool) {
	if tag == "" {
		return
	}

	name, rest, ok := strings.Cut(tag, ",")
	if !ok {
		return
	}

	var option string
	for rest != "" {
		if option, rest, _ = strings.Cut(rest, ","); option == "opaque" {
			opaque = true
			break
		}
	}

	return
}

func appendSlice[T any](parent []T, i T) []T {
	s := make([]T, len(parent)+1)
	copy(s, parent)
	s[len(parent)] = i
	return s
}

func getValueFromMap(m map[string]any, names []string) any {
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

func GetFieldByIndex(v reflect.Value, index []int, alloc bool) reflect.Value {
	cur := v

	for i, _len := 0, len(index)-1; i <= _len; i++ {
		f := cur.Field(index[i])
		if i == _len {
			return f
		}

		// When traversing through a pointer field in an expanded path,
		// reflect.Value.Field returns a value that is NOT addressable/settable
		// even if the parent struct is addressable. We must use FieldByIndex
		// on the original addressable value instead.
		if f.Kind() == reflect.Pointer {
			if f.Type().Elem().Kind() != reflect.Struct {
				panic("non-struct pointer in field path")
			}

			if !f.IsNil() {
				f = f.Elem()
			} else if alloc {
				// Allocate through the original struct by creating the
				// intermediate pointer at index[:i+1].
				f = v.FieldByIndex(index[:i+1])
				f.Set(reflect.New(f.Type().Elem()))
				f = f.Elem()
			} else {
				return reflect.Value{}
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
