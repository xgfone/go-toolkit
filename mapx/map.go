// Copyright 2024~2026 xgfone
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

// Package mapx provides map operations with capacity-aware allocation.
// To converts entries into a new map preallocated to the size of the input.
package mapx

// To converts each entry into a new map with space reserved for len(maps)
// entries. It preserves nilness and does not modify maps. If converted keys
// collide, the last visited entry wins; because map iteration order is
// unspecified, the winning value is unspecified.
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
