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

package slicex

import (
	"iter"
	"slices"
)

// ValuesFunc returns an iterator that yields the slice elements in order.
func ValuesFunc[Slice ~[]E, E, V any](s Slice, f func(E) V) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range s {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// Collect collects the elements from the iterator seq into a slice
// with the given initial capacity and returns it.
func Collect[E any](cap int, seq iter.Seq[E]) []E {
	var s []E
	if cap > 0 {
		s = make([]E, 0, cap)
	}
	return slices.AppendSeq(s, seq)
}
