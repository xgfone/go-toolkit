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

// Package iterx provides some extra iter functions.
package iterx

import "iter"

// All returns true if all elements in the sequences match the predicate.
func All[T any](seq iter.Seq[T], predicate func(T) bool) bool {
	for v := range seq {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// Any returns true if any element in the sequences matches the predicate.
func Any[T any](seq iter.Seq[T], predicate func(T) bool) bool {
	for v := range seq {
		if predicate(v) {
			return true
		}
	}
	return false
}
