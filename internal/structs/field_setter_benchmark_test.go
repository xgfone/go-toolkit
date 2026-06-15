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
)

func BenchmarkFieldSetValueInt(b *testing.B) {
	rtype := reflect.TypeFor[int]()
	field := Field[FieldSetter[string]]{
		Type: rtype,
		Data: FieldSetter[string]{SetField: CompileStringSetter(rtype)},

		Indexes: []int{0},
	}
	root := reflect.ValueOf(&struct{ N int }{}).Elem()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := field.Data.SetField(field.Type, field.GetField(root), "123"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFieldSetValueText(b *testing.B) {
	rtype := reflect.TypeFor[benchText]()
	field := Field[FieldSetter[string]]{
		Type: rtype,
		Data: FieldSetter[string]{SetField: CompileStringSetter(rtype)},

		Indexes: []int{0},
	}
	root := reflect.ValueOf(&struct{ T benchText }{}).Elem()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := field.Data.SetField(field.Type, field.GetField(root), "abc"); err != nil {
			b.Fatal(err)
		}
	}
}
