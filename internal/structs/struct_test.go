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
	"slices"
	"testing"
	"time"
)

type parserData struct {
	fieldName string
	marker    string
}

func newTestParser(isOpaque OpaqueFieldFunc) *Parser[parserData] {
	return NewParser(func(sf reflect.StructField) parserData {
		return parserData{fieldName: sf.Name, marker: sf.Tag.Get("marker")}
	}, isOpaque)
}

func checkFieldNames[Data any](t *testing.T, s *Struct[Data], want ...string) {
	t.Helper()
	got := make([]string, len(s.Fields))
	for i, f := range s.Fields {
		got[i] = f.Name
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got fields %v, want %v", got, want)
	}
}

func fieldByName[Data any](t *testing.T, s *Struct[Data], name string) *Field[Data] {
	t.Helper()
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i]
		}
	}
	t.Fatalf("missing field %q", name)
	return nil
}

func TestNewParserPanicsWithoutCompiler(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = NewParser[string](nil, nil)
}

func TestParseCache(t *testing.T) {
	type target struct {
		Name string `q:"name"`
	}

	parser := newTestParser(nil)
	typ := reflect.TypeFor[target]()
	s1 := parser.Parse(typ, "q")
	s2 := parser.Parse(typ, "q")
	s3 := parser.Parse(typ, "")
	if s1 != s2 {
		t.Fatal("cache miss for same type and tag")
	}
	if s1 == s3 {
		t.Fatal("cache should distinguish parse tags")
	}
}

func TestParseStructFields(t *testing.T) {
	type ExportedEmbed struct {
		A int `q:"a"`
	}
	type unexportedEmbed struct {
		X int `q:"x"`
	}
	type opaqueStruct struct {
		Y int `q:"y"`
	}

	tests := []struct {
		name     string
		typ      reflect.Type
		isOpaque OpaqueFieldFunc
		want     []string
	}{
		{
			name: "anonymous and named structs expand",
			typ: reflect.TypeFor[struct {
				ExportedEmbed
				Inner ExportedEmbed `q:"inner"`
				T     time.Time     `q:"t"`
				Skip  string        `q:"-"`
			}](),
			want: []string{"a", "a", "t"},
		},
		{
			name: "exported pointer embed expands",
			typ: reflect.TypeFor[struct {
				*ExportedEmbed
				B int `q:"b"`
			}](),
			want: []string{"a", "b"},
		},
		{
			name: "unexported pointer embed is skipped",
			typ: reflect.TypeFor[struct {
				*unexportedEmbed
				A int `q:"a"`
			}](),
			want: []string{"a"},
		},
		{
			name: "unexported value embed can expose direct exported fields",
			typ: reflect.TypeFor[struct {
				unexportedEmbed
				A int `q:"a"`
			}](),
			want: []string{"x", "a"},
		},
		{
			name: "tag opaque keeps struct as one field",
			typ: reflect.TypeFor[struct {
				Inner ExportedEmbed `q:"inner,opaque"`
				A     int           `q:"a"`
			}](),
			want: []string{"inner", "a"},
		},
		{
			name: "type opaque keeps struct as one field",
			typ: reflect.TypeFor[struct {
				Opaque opaqueStruct `q:"opaque"`
				A      int          `q:"a"`
			}](),
			isOpaque: func(sf reflect.StructField) bool { return sf.Type == reflect.TypeFor[opaqueStruct]() },
			want:     []string{"opaque", "a"},
		},
		{
			name: "unexported named field is skipped",
			typ: reflect.TypeFor[struct {
				inner ExportedEmbed //nolint:unused
				A     int           `q:"a"`
			}](),
			want: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkFieldNames(t, newTestParser(tt.isOpaque).Parse(tt.typ, "q"), tt.want...)
		})
	}
}

func TestFieldCompilerReceivesLeafStructField(t *testing.T) {
	type inner struct {
		Value int `q:"value" marker:"inner-value"`
	}
	type outer struct {
		Name  string `q:"name" marker:"outer-name"`
		Inner inner  `q:"inner" marker:"outer-inner"`
	}

	s := newTestParser(nil).Parse(reflect.TypeFor[outer](), "q")
	checkFieldNames(t, s, "name", "value")

	if got := fieldByName(t, s, "name").Data.marker; got != "outer-name" {
		t.Fatalf("got name marker %q", got)
	}
	if got := fieldByName(t, s, "value").Data.marker; got != "inner-value" {
		t.Fatalf("got nested marker %q", got)
	}
}

func TestGetValue(t *testing.T) {
	type inner struct {
		Key string `q:"key"`
	}
	type outer struct {
		Inner inner `q:"inner"`
	}

	s := newTestParser(nil).Parse(reflect.TypeFor[outer](), "q")
	key := fieldByName(t, s, "key")

	if got := key.GetValue(map[string]any{"inner": map[string]any{"key": "v"}}); got != "v" {
		t.Fatalf("expected v, got %v", got)
	}
	if got := key.GetValue(nil); got != nil {
		t.Fatalf("expected nil for nil map, got %v", got)
	}
	if got := key.GetValue(map[string]any{"inner": "bad"}); got != nil {
		t.Fatalf("expected nil for wrong intermediate type, got %v", got)
	}
}

func TestGetFieldAllocatesPointerStructPath(t *testing.T) {
	type inner struct {
		N int `q:"n"`
	}
	type outer struct {
		Inner *inner `q:"inner"`
	}

	s := newTestParser(nil).Parse(reflect.TypeFor[outer](), "q")
	var dst outer
	field := fieldByName(t, s, "n")
	field.GetField(reflect.ValueOf(&dst).Elem()).SetInt(12)
	if dst.Inner == nil || dst.Inner.N != 12 {
		t.Fatalf("unexpected target: %#v", dst)
	}

	field.GetField(reflect.ValueOf(&dst).Elem()).SetInt(13)
	valueDst := struct{ Value inner }{}
	fieldByIndexAlloc(reflect.ValueOf(&valueDst).Elem(), []int{0, 0}).SetInt(14)
	if dst.Inner.N != 13 || valueDst.Value.N != 14 {
		t.Fatalf("unexpected target: %#v", dst)
	}
}

func TestFieldByIndexAllocPanics(t *testing.T) {
	tests := []struct {
		name  string
		value any
		index []int
		panic string
	}{
		{name: "empty", value: struct{}{}, panic: "empty field index"},
		{name: "invalid path", value: struct{ N int }{}, index: []int{0, 0}, panic: "invalid field path"},
		{name: "non struct pointer", value: struct{ P *int }{}, index: []int{0, 0}, panic: "non-struct pointer in field path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic")
				}
				if r != tt.panic {
					t.Fatalf("got panic %v, want %q", r, tt.panic)
				}
			}()
			fieldByIndexAlloc(reflect.ValueOf(tt.value), tt.index)
		})
	}
}

func TestParserCachesAreIndependent(t *testing.T) {
	type target struct {
		Name string `q:"name"`
	}

	newParser := func(prefix string) *Parser[parserData] {
		return NewParser(func(sf reflect.StructField) parserData {
			return parserData{fieldName: prefix + sf.Name}
		}, nil)
	}

	s1 := newParser("p1:").Parse(reflect.TypeFor[target](), "q")
	s2 := newParser("p2:").Parse(reflect.TypeFor[target](), "q")
	if s1 == s2 {
		t.Fatal("different parsers shared a cached struct")
	}
	if s1.Fields[0].Data.fieldName != "p1:Name" || s2.Fields[0].Data.fieldName != "p2:Name" {
		t.Fatalf("unexpected parser data: %#v %#v", s1.Fields[0].Data, s2.Fields[0].Data)
	}
}

func TestParseHelpers(t *testing.T) {
	if name, opaque := parseTag("name,omitempty,opaque"); name != "name" || !opaque {
		t.Fatalf("unexpected tag parse: %q %v", name, opaque)
	}
	if name, opaque := parseTag("name"); name != "name" || opaque {
		t.Fatalf("unexpected name-only tag parse: %q %v", name, opaque)
	}

	if got := appendSlice([]int{1, 2}, 3); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("unexpected index: %#v", got)
	}
	if got := makeMapValueGetter(nil)(map[string]any{"x": 1}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
