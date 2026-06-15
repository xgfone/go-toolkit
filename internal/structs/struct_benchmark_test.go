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
	"unsafe"
)

type benchText string

func (t *benchText) UnmarshalText(b []byte) error {
	*t = benchText(unsafe.String(unsafe.SliceData(b), len(b)))
	return nil
}

type benchStruct struct {
	Name string    `q:"name"`
	Age  *int      `q:"age"`
	Text benchText `q:"text"`
}

func BenchmarkParseHit(b *testing.B) {
	typ := reflect.TypeFor[benchStruct]()
	parser := NewStringSetterParser("")
	_ = parser.Parse(typ, "q")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.Parse(typ, "q")
	}
}

func BenchmarkRawParse(b *testing.B) {
	typ := reflect.TypeFor[benchStruct]()
	parser := NewParser(NewSetterFieldCompiler(CompileStringSetter, ""), isStringOpaqueField)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = parser._Parse(typ, "q")
	}
}

func BenchmarkGetValueNested(b *testing.B) {
	type inner struct {
		Key string `q:"key"`
	}
	type outer struct {
		Inner inner `q:"inner"`
	}

	s := NewStringSetterParser("").Parse(reflect.TypeFor[outer](), "q")
	field := s.Fields[0]
	values := map[string]any{"inner": map[string]any{"key": "value"}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := field.GetValue(values); got != "value" {
			b.Fatalf("got %v", got)
		}
	}
}
