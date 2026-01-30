// Copyright 2023~2025 xgfone
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
