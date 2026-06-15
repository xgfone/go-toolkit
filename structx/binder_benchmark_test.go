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

type benchValuesGetter map[string]string

func (g benchValuesGetter) Get(k string) string { return g[k] }

type benchValuesText string

func (t *benchValuesText) UnmarshalText(b []byte) error {
	*t = benchValuesText(b)
	return nil
}

type benchValuesFlat struct {
	Name  string  `q:"name"`
	Age   int     `q:"age"`
	Flag  bool    `q:"flag"`
	Score float64 `q:"score"`
	Count uint    `q:"count"`
}

type benchValuesPointers struct {
	Name  *string          `q:"name"`
	Age   *int             `q:"age"`
	Text  benchValuesText  `q:"text"`
	PText *benchValuesText `q:"ptext"`
}

type benchMapBindValue string

func (b *benchMapBindValue) Bind(v any) error {
	*b = benchMapBindValue("bind:" + v.(string))
	return nil
}

type benchMapFlat struct {
	Name  string  `json:"name"`
	Age   int     `json:"age"`
	Flag  bool    `json:"flag"`
	Score float64 `json:"score"`
	Count uint    `json:"count"`
}

type benchMapNestedInner struct {
	Age   *int              `json:"age"`
	Text  benchMapBindValue `json:"text"`
	Score float64           `json:"score"`
}

type benchMapNested struct {
	Name  string               `json:"name"`
	Flag  bool                 `json:"flag"`
	Inner benchMapNestedInner  `json:"inner"`
	PText *benchMapBindValue   `json:"ptext"`
	Meta  map[string][]float64 `json:"meta"`
}

func BenchmarkBindValuesFlat(b *testing.B) {
	source := benchValuesGetter{
		"name":  "alice",
		"age":   "42",
		"flag":  "true",
		"score": "12.5",
		"count": "7",
	}
	target := new(benchValuesFlat)
	_ = BindValues(target, source, "q")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*target = benchValuesFlat{}
		if err := BindValues(target, source, "q"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBindMapFlat(b *testing.B) {
	source := map[string]any{
		"name":  "alice",
		"age":   42,
		"flag":  true,
		"score": 12.5,
		"count": uint(7),
	}
	target := new(benchMapFlat)
	_ = BindMap(target, source, "json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*target = benchMapFlat{}
		if err := BindMap(target, source, "json"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBindMapNestedPointersAndBinders(b *testing.B) {
	source := map[string]any{
		"name": "alice",
		"flag": true,
		"inner": map[string]any{
			"age":   42,
			"text":  "tv",
			"score": 12.5,
		},
		"ptext": "ptv",
		"meta": map[string]any{
			"weights": []any{1.5, 2.5, 3.5},
		},
	}
	target := new(benchMapNested)
	_ = BindMap(target, source, "json")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*target = benchMapNested{}
		if err := BindMap(target, source, "json"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBindValuesPointersAndText(b *testing.B) {
	source := benchValuesGetter{
		"name":  "alice",
		"age":   "42",
		"text":  "tv",
		"ptext": "ptv",
	}
	target := new(benchValuesPointers)
	_ = BindValues(target, source, "q")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*target = benchValuesPointers{}
		if err := BindValues(target, source, "q"); err != nil {
			b.Fatal(err)
		}
	}
}
