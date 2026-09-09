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

// Package slicex provides some extra slice functions.
package slicex

// Empty returns itself if vs is not nil, otherwise returns an empty slice.
//
// DEPRECATED. DO NOT USE IT.
func Empty[S ~[]E, E any](vs S) S {
	if vs == nil {
		return S{}
	}
	return vs
}

// Convert converts the slice from []E1 to []E2.
//
// DEPRECATED. Please use To instead.
func Convert[S1 ~[]E1, E1, E2 any](vs S1, convert func(E1) E2) []E2 {
	return To(vs, convert)
}

// Filter filters the elements of the slice s and converts them.
func Filter[S1 ~[]E1, E1, E2 any](s S1, filter func(E1) (E2, bool)) []E2 {
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

// To converts the slice from []E1 to []E2.
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

// To2 converts the slice from []E1 to []E2.
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

// Map converts a slice to a map.
func Map[S ~[]E, K comparable, V, E any](s S, convert func(E) (K, V)) map[K]V {
	return Map2(s, func(_ int, e E) (K, V) { return convert(e) })
}

// Map2 converts a slice with the index to a map.
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

// Merge concatenates multiple slices into a single slice.
//
// If no slices are provided, it returns nil.
// If all input slices are empty or nil, it returns an empty slice of type S.
//
// Note: If there is only one slice, it will return that slice directly
// without cloning as the performance optimization.
func Merge[S ~[]E, E any](ss ...S) S {
	switch len(ss) {
	case 0:
		return nil

	case 1:
		return ss[0]
	}

	var _len int
	for i := range ss {
		_len += len(ss[i])
	}

	if _len == 0 {
		return S{}
	}

	vs := make(S, 0, _len)
	for i := range ss {
		vs = append(vs, ss[i]...)
	}
	return vs
}

// ContainsAll reports whether subset is a subset of superset.
//
// It treats the slices as sets: the element order and repeated elements do not
// affect the result.
func ContainsAll[S1 ~[]E, S2 ~[]E, E comparable](superset S1, subset S2) bool {
	if len(subset) == 0 {
		return true
	}

	if len(superset) == 0 {
		return false
	}

	set := make(map[E]struct{}, len(superset))
	for _, value := range superset {
		set[value] = struct{}{}
	}

	for _, value := range subset {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}

// ContainsAllFunc reports whether subset is a subset of superset using equal.
//
// It treats the slices as sets: the element order and repeated elements do not
// affect the result.
func ContainsAllFunc[S1 ~[]E1, S2 ~[]E2, E1, E2 any](superset S1, subset S2, equal func(E1, E2) bool) bool {
	if len(subset) == 0 {
		return true
	}

	if len(superset) == 0 {
		return false
	}

	for _, sub := range subset {
		found := false
		for _, super := range superset {
			if equal(super, sub) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
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
