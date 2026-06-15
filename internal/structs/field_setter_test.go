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
	"testing"
	"time"
)

type textSetterStruct struct {
	N int `q:"n"`
}

func (t *textSetterStruct) UnmarshalText(b []byte) error {
	t.N = len(b)
	return nil
}

type anySetterStruct struct {
	V any `q:"v"`
}

func (b *anySetterStruct) Bind(v any) error {
	b.V = v
	return nil
}

func onlySetterField[T any](t *testing.T, s *Struct[FieldSetter[T]], name string) *Field[FieldSetter[T]] {
	t.Helper()
	checkFieldNames(t, s, name)
	return &s.Fields[0]
}

func setSetterField[T any](t *testing.T, f *Field[FieldSetter[T]], dst any, value T) {
	t.Helper()
	root := reflect.ValueOf(dst).Elem()
	if err := f.Data.SetField(f.Type, f.GetField(root), value); err != nil {
		t.Fatalf("SetField(%v): %v", value, err)
	}
}

func TestFieldSetterParsers(t *testing.T) {
	t.Run("string value tag", func(t *testing.T) {
		type target struct {
			Name string `q:"name" default:"alice"`
		}

		field := fieldByName(t, NewStringSetterParser("default").Parse(reflect.TypeFor[target](), "q"), "name")
		if field.Data.TagValue != "alice" {
			t.Fatalf("got tag value %q", field.Data.TagValue)
		}

		var dst target
		setSetterField(t, field, &dst, "bob")
		if dst.Name != "bob" {
			t.Fatalf("got name %q", dst.Name)
		}
	})

	t.Run("string opaque text unmarshaler", func(t *testing.T) {
		type target struct {
			Value textSetterStruct `q:"value"`
		}

		field := onlySetterField(t, NewStringSetterParser("").Parse(reflect.TypeFor[target](), "q"), "value")
		var dst target
		setSetterField(t, field, &dst, "hello")
		if dst.Value.N != 5 {
			t.Fatalf("got value %#v", dst.Value)
		}
	})

	t.Run("any opaque binder", func(t *testing.T) {
		type target struct {
			Value anySetterStruct `q:"value"`
		}

		field := onlySetterField(t, NewAnySetterParser("").Parse(reflect.TypeFor[target](), "q"), "value")
		var dst target
		setSetterField(t, field, &dst, 123)
		if dst.Value.V != 123 {
			t.Fatalf("got value %#v", dst.Value.V)
		}
	})

	t.Run("any time string conversion", func(t *testing.T) {
		type target struct {
			At time.Time `q:"at"`
		}

		field := onlySetterField(t, NewAnySetterParser("").Parse(reflect.TypeFor[target](), "q"), "at")
		ts := time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)

		var dst target
		setSetterField(t, field, &dst, any(ts.Format(time.RFC3339)))
		if !dst.At.Equal(ts) {
			t.Fatalf("got %v, want %v", dst.At, ts)
		}
	})
}
