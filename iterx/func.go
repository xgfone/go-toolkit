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

type number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Sum returns the sum of the integer or floating-point elements in seq,
// or zero if seq is empty. It accumulates in V using Go's addition semantics.
func Sum[V number](seq iter.Seq[V]) (sum V) {
	for v := range seq {
		sum += v
	}
	return
}

// SumFunc returns the sum of f(v) for every element v in seq, or zero if seq is
// empty. It calls f once per element and accumulates in R using Go's addition
// semantics. R may be an integer or floating-point type.
func SumFunc[V any, R number](seq iter.Seq[V], f func(V) R) (sum R) {
	for v := range seq {
		sum += f(v)
	}
	return
}

// Count returns the number of elements in seq, or zero if seq is empty.
// It consumes the entire sequence.
func Count[V any](seq iter.Seq[V]) (count int) {
	for range seq {
		count++
	}
	return
}

// CountFunc returns the number of elements that match predicate, or zero if seq
// is empty. It calls predicate once per element and consumes the entire sequence.
func CountFunc[V any](seq iter.Seq[V], predicate func(V) bool) (count int) {
	for v := range seq {
		if predicate(v) {
			count++
		}
	}
	return
}

// Find returns the first element that matches predicate and stops the sequence.
// If no element matches, it returns the zero value of V and false.
func Find[V any](seq iter.Seq[V], predicate func(V) bool) (V, bool) {
	for v := range seq {
		if predicate(v) {
			return v, true
		}
	}
	var zero V
	return zero, false
}

// Reduce folds seq from left to right, starting with initial. It calls f once
// per element with the current accumulator and element, and returns the final
// accumulator. If seq is empty, it returns initial without calling f.
func Reduce[V, R any](seq iter.Seq[V], initial R, f func(R, V) R) R {
	for v := range seq {
		initial = f(initial, v)
	}
	return initial
}
