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

package iterx

import (
	"iter"

	"github.com/xgfone/go-toolkit/iterx"
)

// Stream provides chainable, lazy transformations of a one-value iterator.
// Use Seq to iterate or pass the result to standard library functions.
// Transformations return new wrappers without consuming or caching the input.
// Repeated iteration depends on the input, just as with iter.Seq.
// The zero value contains a nil iterator and is not an empty stream.
type Stream[V any] struct {
	seq iter.Seq[V]
}

// Stream2 provides chainable, lazy transformations of a two-value iterator.
// K and V may be any types; the first value need not be a comparable map key.
// Like Stream, it preserves the input's iteration behavior and has an invalid
// zero value. Use Keys or Values to continue with a one-value Stream.
type Stream2[K, V any] struct {
	seq iter.Seq2[K, V]
}

// FromSeq wraps seq without starting it. The input must not be nil.
func FromSeq[V any](seq iter.Seq[V]) Stream[V] {
	return Stream[V]{seq: seq}
}

// FromSeq2 wraps seq without starting it. The input must not be nil.
func FromSeq2[K, V any](seq iter.Seq2[K, V]) Stream2[K, V] {
	return Stream2[K, V]{seq: seq}
}

// Seq returns the underlying iterator without starting it.
func (s Stream[V]) Seq() iter.Seq[V] {
	return s.seq
}

// Seq returns the underlying two-value iterator without starting it.
func (s Stream2[K, V]) Seq() iter.Seq2[K, V] {
	return s.seq
}

// Map transforms each value. It is the chainable form of [iterx.To].
func (s Stream[V]) Map[R any](convert func(V) R) Stream[R] {
	return FromSeq(iterx.To(s.seq, convert))
}

// Map2 transforms each pair into another pair. It is the chainable form of [iterx.To2].
func (s Stream2[K, V]) Map2[K2, V2 any](convert func(K, V) (K2, V2)) Stream2[K2, V2] {
	return FromSeq2(iterx.To2(s.seq, convert))
}

// Filter keeps matching values. It is the chainable form of [iterx.Filter].
func (s Stream[V]) Filter(predicate func(V) bool) Stream[V] {
	return FromSeq(iterx.Filter(s.seq, predicate))
}

// Filter keeps matching pairs. It is the chainable form of [iterx.Filter2].
func (s Stream2[K, V]) Filter(predicate func(K, V) bool) Stream2[K, V] {
	return FromSeq2(iterx.Filter2(s.seq, predicate))
}

// FilterMap transforms values and keeps results for which convert returns true.
// It is the chainable form of [iterx.FilterTo].
func (s Stream[V]) FilterMap[R any](convert func(V) (R, bool)) Stream[R] {
	return FromSeq(iterx.FilterTo(s.seq, convert))
}

// FilterMap2 transforms pairs and keeps results for which convert returns true.
// It is the chainable form of [iterx.FilterTo2].
func (s Stream2[K, V]) FilterMap2[K2, V2 any](convert func(K, V) (K2, V2, bool)) Stream2[K2, V2] {
	return FromSeq2(iterx.FilterTo2(s.seq, convert))
}

// Take keeps at most n values. Like [iterx.Take], it panics if n is negative.
func (s Stream[V]) Take(n int) Stream[V] {
	return FromSeq(iterx.Take(s.seq, n))
}

// Skip skips the first n values. Like [iterx.Drop], it panics if n is negative.
func (s Stream[V]) Skip(n int) Stream[V] {
	return FromSeq(iterx.Drop(s.seq, n))
}

// Keys projects the first value of each pair. It is the chainable form of [iterx.Keys].
func (s Stream2[K, V]) Keys() Stream[K] {
	return FromSeq(iterx.Keys(s.seq))
}

// Values projects the second value of each pair. It is the chainable form of [iterx.Values].
func (s Stream2[K, V]) Values() Stream[V] {
	return FromSeq(iterx.Values(s.seq))
}

// To is an alias for [Stream.Map].
func (s Stream[V]) To[R any](convert func(V) R) Stream[R] {
	return s.Map(convert)
}

// To2 is an alias for [Stream2.Map2].
func (s Stream2[K, V]) To2[K2, V2 any](convert func(K, V) (K2, V2)) Stream2[K2, V2] {
	return s.Map2(convert)
}

// FilterTo is an alias for [Stream.FilterMap].
func (s Stream[V]) FilterTo[R any](convert func(V) (R, bool)) Stream[R] {
	return s.FilterMap(convert)
}

// FilterTo2 is an alias for [Stream2.FilterMap2].
func (s Stream2[K, V]) FilterTo2[K2, V2 any](convert func(K, V) (K2, V2, bool)) Stream2[K2, V2] {
	return s.FilterMap2(convert)
}

// Drop is an alias for [Stream.Skip].
func (s Stream[V]) Drop(n int) Stream[V] {
	return s.Skip(n)
}
