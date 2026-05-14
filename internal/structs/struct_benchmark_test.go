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
	"strconv"
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
	_ = Parse(typ, "q")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Parse(typ, "q")
	}
}

func BenchmarkParseMiss(b *testing.B) {
	typ := reflect.TypeFor[benchStruct]()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Parse(typ, "q"+strconv.Itoa(i))
	}
}

func BenchmarkFieldSetValueInt(b *testing.B) {
	rtype := reflect.TypeFor[int]()
	field := Field{
		Type:     rtype,
		SetField: CompileSetter(rtype),
		GetField: makeFieldGetter([]int{0}),
	}

	root := reflect.ValueOf(&struct{ N int }{}).Elem()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := field.SetValue(root, "123"); err != nil {
			b.Fatal(err)
		}
	}

}

func BenchmarkFieldSetValueText(b *testing.B) {
	rtype := reflect.TypeFor[benchText]()
	field := Field{
		Type:     rtype,
		SetField: CompileSetter(rtype),
		GetField: makeFieldGetter([]int{0}),
	}

	root := reflect.ValueOf(&struct{ T benchText }{}).Elem()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := field.SetValue(root, "abc"); err != nil {
			b.Fatal(err)
		}
	}
}
