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

package slicex_test

import (
	"slices"
	"testing"

	"github.com/xgfone/go-toolkit/iterx"
	"github.com/xgfone/go-toolkit/slicex"
)

var collectedInts []int

func BenchmarkCollect(b *testing.B) {
	input := make([]int, 1024)
	b.Run("Standard", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			collectedInts = slices.Collect(slices.Values(input))
		}
	})

	b.Run("Preallocated", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			collectedInts = slices.AppendSeq(make([]int, 0, len(input)), slices.Values(input))
		}
	})
}

func BenchmarkTo(b *testing.B) {
	input := make([]int, 1024)
	convert := func(v int) int { return v + 1 }
	b.Run("IteratorCollect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			collectedInts = slices.Collect(iterx.To(slices.Values(input), convert))
		}
	})

	b.Run("Preallocated", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			collectedInts = slicex.To(input, convert)
		}
	})
}
