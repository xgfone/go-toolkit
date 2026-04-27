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

import "testing"

// ---- Benchmark test structs ----

type benchDefaultSimple struct {
	Name string `default:"hello"`
	Age  int    `default:"18"`
}

type benchDefaultAllFields struct {
	Name    string  `default:"alice"`
	Age     int     `default:"25"`
	Score   float64 `default:"99.5"`
	Active  bool    `default:"true"`
	Tag     string  `default:"dev"`
	Counter int64   `default:"1000"`
}

type benchDefaultPartial struct {
	Name    string  `default:"bob"`
	Age     int     `default:"30"`
	Score   float64 // no default tag
	Active  bool    // no default tag
	Country string  `default:"CN"`
}

type benchDefaultNested struct {
	Inner struct {
		X string `default:"nested_x"`
		Y int    `default:"42"`
	}
	Outer string `default:"outer_val"`
}

type benchDefaultAlreadySet struct {
	Name string `default:"default_name"`
	Age  int    `default:"99"`
}

// ---- Benchmarks ----

func BenchmarkSetDefaultSimple(b *testing.B) {
	v := &benchDefaultSimple{}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := SetDefault(v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetDefaultAllFields(b *testing.B) {
	v := &benchDefaultAllFields{}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := SetDefault(v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetDefaultPartialFields(b *testing.B) {
	v := &benchDefaultPartial{}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := SetDefault(v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetDefaultNested(b *testing.B) {
	v := &benchDefaultNested{}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := SetDefault(v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetDefaultAlreadySet(b *testing.B) {
	v := &benchDefaultAlreadySet{
		Name: "existing",
		Age:  50,
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := SetDefault(v); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSetDefaultNilPointer(b *testing.B) {
	var v *benchDefaultSimple
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = SetDefault(v) // returns error, no panic
	}
}

func BenchmarkSetDefaultNonStruct(b *testing.B) {
	v := 42
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = SetDefault(&v) // returns error, no panic
	}
}

func BenchmarkSetDefaultNoDefaults(b *testing.B) {
	type noDefault struct {
		Name string
		Age  int
	}
	v := &noDefault{}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := SetDefault(v); err != nil {
			b.Fatal(err)
		}
	}
}
