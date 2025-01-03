// Copyright 2025 xgfone
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

//go:build go1.23

package mapx

import "iter"

// All is the same as maps.All to return an iterator over key-value pairs from m,
// but uses Pair[K, V] to return iter.Seq instead of iter.Seq2.
func All[M ~map[K]V, K comparable, V any](m M) iter.Seq[Pair[K, V]] {
	return func(yield func(Pair[K, V]) bool) {
		for k, v := range m {
			if !yield(Pair[K, V]{Key: k, Value: v}) {
				return
			}
		}
	}
}
