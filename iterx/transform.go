// Copyright 2025~2026 xgfone
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

package iterx

import "iter"

// To returns a sequence containing convert(v) for each element v in seq.
func To[V, R any](seq iter.Seq[V], convert func(V) R) iter.Seq[R] {
	return func(yield func(R) bool) {
		for v := range seq {
			if !yield(convert(v)) {
				return
			}
		}
	}
}

// To2 returns a sequence containing convert(k, v) for each pair in seq.
// Both the input and output are two-value iterators.
func To2[K, V, K2, V2 any](seq iter.Seq2[K, V], convert func(K, V) (K2, V2)) iter.Seq2[K2, V2] {
	return func(yield func(K2, V2) bool) {
		for k, v := range seq {
			if !yield(convert(k, v)) {
				return
			}
		}
	}
}

// Keys returns a sequence of the first values in seq's pairs.
// The values need not be comparable.
func Keys[K, V any](seq iter.Seq2[K, V]) iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range seq {
			if !yield(k) {
				return
			}
		}
	}
}

// Values returns a sequence of the second values in seq's pairs.
func Values[K, V any](seq iter.Seq2[K, V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range seq {
			if !yield(v) {
				return
			}
		}
	}
}

// Filter returns a sequence containing only elements that match predicate.
func Filter[V any](seq iter.Seq[V], predicate func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if predicate(v) && !yield(v) {
				return
			}
		}
	}
}

// Filter2 returns a sequence containing only pairs that match predicate.
func Filter2[K, V any](seq iter.Seq2[K, V], predicate func(K, V) bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range seq {
			if predicate(k, v) && !yield(k, v) {
				return
			}
		}
	}
}

// FilterTo applies convert once to each input element and yields the converted
// value only when convert returns true.
func FilterTo[V, R any](seq iter.Seq[V], convert func(V) (R, bool)) iter.Seq[R] {
	return func(yield func(R) bool) {
		for v := range seq {
			if r, ok := convert(v); ok && !yield(r) {
				return
			}
		}
	}
}

// FilterTo2 applies convert once to each input pair and yields the converted
// pair only when convert returns true.
func FilterTo2[K, V, K2, V2 any](seq iter.Seq2[K, V], convert func(K, V) (K2, V2, bool)) iter.Seq2[K2, V2] {
	return func(yield func(K2, V2) bool) {
		for k, v := range seq {
			if k2, v2, ok := convert(k, v); ok && !yield(k2, v2) {
				return
			}
		}
	}
}

// Take returns a sequence containing at most the first n elements of seq.
// It stops the input immediately after the nth element, without requesting an
// extra element. If n is zero, it does not start seq. It panics if n is negative.
func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	if n < 0 {
		panic("iterx.Take: negative count")
	}

	return func(yield func(V) bool) {
		remaining := n
		if remaining == 0 {
			return
		}

		for v := range seq {
			if !yield(v) {
				return
			}

			remaining--
			if remaining == 0 {
				return
			}
		}
	}
}

// Drop returns a sequence that skips the first n elements of seq.
// It panics if n is negative.
func Drop[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	if n < 0 {
		panic("iterx.Drop: negative count")
	}

	return func(yield func(V) bool) {
		remaining := n
		for v := range seq {
			if remaining > 0 {
				remaining--
				continue
			}

			if !yield(v) {
				return
			}
		}
	}
}

// Concat returns a sequence that visits each input sequence in order.
// It does not start later sequences if the consumer stops early.
// With no inputs, it returns an empty sequence.
func Concat[V any](seqs ...iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, seq := range seqs {
			for v := range seq {
				if !yield(v) {
					return
				}
			}
		}
	}
}
