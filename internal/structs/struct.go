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

type mapKey struct {
	typ reflect.Type
	tag string
}

type Struct struct {
	Fields []Field
}

type Field struct {
	Name     string
	Default  string
	SetValue func(root reflect.Value, s string) error
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
		if !sf.IsExported() {
			continue
		}

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

		if sf.Anonymous && name == "" && ft.Kind() == reflect.Struct {
			fields = append(fields, parseWithParent(ft, tag, index)...)
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

func makeValueSetter(index []int, rtype reflect.Type) func(root reflect.Value, s string) error {
	setter := CompileSetter(rtype)
	return func(root reflect.Value, s string) error {
		fv, err := fieldByIndexAlloc(root, index)
		if err != nil {
			return err
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

		switch f.Kind() {
		case reflect.Struct:
			cur = f

		case reflect.Pointer:
			if f.Type().Elem().Kind() != reflect.Struct {
				return reflect.Value{}, errors.New("non-struct pointer in field path")
			}
			if f.IsNil() {
				f.Set(reflect.New(f.Type().Elem()))
			}
			cur = f.Elem()

		default:
			return reflect.Value{}, errors.New("invalid field path")
		}
	}

	return reflect.Value{}, errors.New("empty field index")
}
