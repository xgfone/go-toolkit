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

package mapx_test

import (
	"maps"
	"slices"
	"testing"
)

var collectedKeys []int

func BenchmarkKeys(b *testing.B) {
	input := make(map[int]int, 1024)
	for i := range 1024 {
		input[i] = i
	}

	b.Run("StandardCollect", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			collectedKeys = slices.Collect(maps.Keys(input))
		}
	})

	b.Run("StandardPreallocated", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			collectedKeys = slices.AppendSeq(make([]int, 0, len(input)), maps.Keys(input))
		}
	})
}
