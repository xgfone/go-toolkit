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

package binder

import "testing"

type benchGetter map[string]string

func (g benchGetter) Get(k string) string { return g[k] }

type benchText string

func (t *benchText) UnmarshalText(b []byte) error {
	*t = benchText(b)
	return nil
}

type benchFlat struct {
	Name  string  `q:"name"`
	Age   int     `q:"age"`
	Flag  bool    `q:"flag"`
	Score float64 `q:"score"`
	Count uint    `q:"count"`
}

type benchPointers struct {
	Name  *string    `q:"name"`
	Age   *int       `q:"age"`
	Text  benchText  `q:"text"`
	PText *benchText `q:"ptext"`
}

func BenchmarkBindGetterFlat(b *testing.B) {
	src := benchGetter{
		"name":  "alice",
		"age":   "42",
		"flag":  "true",
		"score": "12.5",
		"count": "7",
	}
	var warm benchFlat
	_ = BindGetter(src, &warm, "q")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dst benchFlat
		if err := BindGetter(src, &dst, "q"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBindGetterPointersAndText(b *testing.B) {
	src := benchGetter{
		"name":  "alice",
		"age":   "42",
		"text":  "tv",
		"ptext": "ptv",
	}
	var warm benchPointers
	_ = BindGetter(src, &warm, "q")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dst benchPointers
		if err := BindGetter(src, &dst, "q"); err != nil {
			b.Fatal(err)
		}
	}
}
