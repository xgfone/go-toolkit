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

package mapx

// SMap is a map with string key and generic value.
type SMap[T any] map[string]T

// NewSMap returns a new SMap with the given capacity.
func NewSMap[T any](cap int) SMap[T] {
	return make(SMap[T], cap)
}

// Get returns the value of the given key, but ZERO if the key does not exist.
func (m SMap[T]) Get(key string) T {
	return m[key]
}
