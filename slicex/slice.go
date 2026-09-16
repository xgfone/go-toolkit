// Copyright 2023~2026 xgfone
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

// Package slicex provides slice operations with capacity-aware allocation.
// To converts elements, and FilterTo selects and converts them.
// Map and Map2 collect converted elements into a Go map.
package slicex

// FilterTo converts each element and keeps the result only when filter returns
// true. It preserves order and nilness, does not modify s, and reserves space
// for len(s) output elements.
func FilterTo[S1 ~[]E1, E1, E2 any](s S1, filter func(E1) (E2, bool)) []E2 {
	if s == nil {
		return nil
	}

	newslice := make([]E2, 0, len(s))
	for i := range s {
		if e2, ok := filter(s[i]); ok {
			newslice = append(newslice, e2)
		}
	}

	return newslice
}

// To converts each element of vs, allocating the output at its final length.
// It preserves order and nilness and does not modify vs.
func To[S1 ~[]E1, E1, E2 any](vs S1, convert func(E1) E2) []E2 {
	if vs == nil {
		return nil
	}

	newslice := make([]E2, len(vs))
	for i := range vs {
		newslice[i] = convert(vs[i])
	}
	return newslice
}

// To2 is like To, but passes both the index and value to convert.
func To2[S1 ~[]E1, E1, E2 any](vs S1, convert func(int, E1) E2) []E2 {
	if vs == nil {
		return nil
	}

	newslice := make([]E2, len(vs))
	for i := range vs {
		newslice[i] = convert(i, vs[i])
	}
	return newslice
}

// Map converts a slice to a map, reserving space for len(s) entries.
// Later elements overwrite earlier entries with the same converted key.
// Empty or nil input returns a non-nil empty map.
func Map[S ~[]E, K comparable, V, E any](s S, convert func(E) (K, V)) map[K]V {
	return Map2(s, func(_ int, e E) (K, V) { return convert(e) })
}

// Map2 is like Map, but passes both the index and value to convert.
func Map2[S ~[]E, K comparable, V, E any](s S, convert func(int, E) (K, V)) map[K]V {
	_len := len(s)
	maps := make(map[K]V, _len)
	for i := range _len {
		k, v := convert(i, s[i])
		maps[k] = v
	}
	return maps
}

// GroupBy groups the elements of s by key, preserving their order within each
// group. It calls key once for each element and does not modify s.
// The elements are shallow copies. Empty or nil input returns a non-nil empty map.
func GroupBy[S ~[]E, K comparable, E any](s S, key func(E) K) map[K][]E {
	groups := make(map[K][]E)
	for _, value := range s {
		k := key(value)
		groups[k] = append(groups[k], value)
	}
	return groups
}

// HasDuplicates reports whether s contains equal elements, including
// non-adjacent ones. It does not modify s. Empty or nil input returns false.
// Elements are compared using ==, so floating-point NaNs are never duplicates.
func HasDuplicates[S ~[]E, E comparable](s S) bool {
	if len(s) < 2 {
		return false
	}

	seen := make(map[E]struct{}, len(s))
	for _, value := range s {
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

// Unique returns a shallow copy of s with duplicate elements removed,
// preserving the order of their first occurrence. It does not modify s or
// share its backing array, and it preserves the nilness of s.
// Elements are compared using ==, so floating-point NaNs are all retained.
//
// Unlike slices.Compact, Unique also removes non-adjacent duplicates.
func Unique[S ~[]E, E comparable](s S) S {
	if s == nil {
		return nil
	}

	values := make(S, 0, len(s))
	seen := make(map[E]struct{}, len(s))
	for _, value := range s {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			values = append(values, value)
		}
	}
	return values
}
