// Copyright 2024~2025 xgfone
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

// Package mapx provides some extra map functions.
package mapx

// Pair represents a key-value pair of map.
type Pair[K comparable, V any] struct {
	Key   K
	Value V
}

// Convert converts the map from map[K1]V1 to map[K2]V2.
//
// DEPRECATED. Please use To instead.
func Convert[M ~map[K1]V1, K1, K2 comparable, V1, V2 any](maps M, convert func(K1, V1) (K2, V2)) map[K2]V2 {
	return To(maps, convert)
}

// Filter filters the elements of the map m and converts them.
func Filter[M ~map[K1]V1, K1, K2 comparable, V1, V2 any](m M, filter func(K1, V1) (K2, V2, bool)) map[K2]V2 {
	if m == nil {
		return nil
	}

	newmap := make(map[K2]V2, len(m))
	for k1, v1 := range m {
		if k2, v2, ok := filter(k1, v1); ok {
			newmap[k2] = v2
		}
	}

	return newmap
}

// To converts the map from map[K1]V1 to map[K2]V2.
func To[M ~map[K1]V1, K1, K2 comparable, V1, V2 any](maps M, convert func(K1, V1) (K2, V2)) map[K2]V2 {
	if maps == nil {
		return nil
	}

	newmap := make(map[K2]V2, len(maps))
	for k1, v1 := range maps {
		k2, v2 := convert(k1, v1)
		newmap[k2] = v2
	}
	return newmap
}

// Values returns all the values of the map.
func Values[M ~map[K]V, K comparable, V any](maps M) []V {
	values := make([]V, 0, len(maps))
	for _, v := range maps {
		values = append(values, v)
	}
	return values
}

// Keys returns all the keys of the map.
func Keys[M ~map[K]V, K comparable, V any](maps M) []K {
	keys := make([]K, 0, len(maps))
	for k := range maps {
		keys = append(keys, k)
	}
	return keys
}

// KeysFunc returns all the keys of the map by the conversion function.
func KeysFunc[M ~map[K]V, T any, K comparable, V any](maps M, convert func(K) T) []T {
	keys := make([]T, 0, len(maps))
	for k := range maps {
		keys = append(keys, convert(k))
	}
	return keys
}

// Values returns all the values of the map by the conversion function.
func ValuesFunc[M ~map[K]V, T any, K comparable, V any](maps M, convert func(V) T) []T {
	values := make([]T, 0, len(maps))
	for _, v := range maps {
		values = append(values, convert(v))
	}
	return values
}
