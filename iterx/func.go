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

package iterx

import (
	"iter"
)

// Integer is the integer type.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Number is the integer or float type.
type Number interface {
	Integer | ~float32 | ~float64
}

func _predicate[V any](seq iter.Seq[V], predicate func(V) bool, result bool) bool {
	for v := range seq {
		if predicate(v) == result {
			return result
		}
	}
	return !result
}

// All returns true if all elements in the sequences match the predicate.
func All[V any](seq iter.Seq[V], predicate func(V) bool) bool {
	return _predicate(seq, predicate, false)
}

// Any returns true if any element in the sequences matches the predicate.
func Any[V any](seq iter.Seq[V], predicate func(V) bool) bool {
	return _predicate(seq, predicate, true)
}

// Sum returns the sum of the elements in the sequence.
func Sum[V any, R Number](seq iter.Seq[V], f func(V) R) R {
	var r R
	for v := range seq {
		r += f(v)
	}
	return r
}

// Filter returns a new sequence that only contains the elements that match the predicate.
func Filter[V any](seq iter.Seq[V], predicate func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if predicate(v) && !yield(v) {
				return
			}
		}
	}
}

// Filter2 returns a new sequence that only contains the elements that match the predicate.
func Filter2[K, V any](seq iter.Seq2[K, V], predicate func(K, V) bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range seq {
			if predicate(k, v) && !yield(k, v) {
				return
			}
		}
	}
}

// Map returns a new sequence that contains the results of applying the mapper function to the elements.
func Map[T any, R any](seq iter.Seq[T], mapper func(T) R) iter.Seq[R] {
	return func(yield func(R) bool) {
		for v := range seq {
			if !yield(mapper(v)) {
				return
			}
		}
	}
}

// Seq returns a new sequence that converts iter.Seq2 to iter.Seq.
func Seq[K, V, T any](seq2 iter.Seq2[K, V], mapper func(K, V) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for k, v := range seq2 {
			if !yield(mapper(k, v)) {
				return
			}
		}
	}
}

// Seq2 returns a new sequence that converts iter.Seq to iter.Seq2.
func Seq2[T, K, V any](seq iter.Seq[T], mapper func(T) (K, V)) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for v := range seq {
			if !yield(mapper(v)) {
				return
			}
		}
	}
}
