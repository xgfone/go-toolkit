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

// Package iterx provides lazy transformations and terminal operations for
// [iter.Seq] and [iter.Seq2].
//
// To and To2 transform values while keeping the number of values yielded per
// element. Keys and Values project one side of a two-value iterator.
// Filter selects elements; FilterTo both selects and converts them.
//
// Adapters do not start the input until iteration begins and do not cache results.
// They preserve the input's order and stop it when the consumer stops. Whether
// they can be iterated again depends on the input: adapting a single-use iterator
// does not make it reusable. Callbacks run again when an input is iterated again.
//
// Terminal operations consume the input. Find stops at the first matching
// element; Sum, Count, their Func variants and Reduce exhaust the input.
// A nil iterator is not an empty sequence and must not be used. Callbacks must
// be non-nil when called. These functions add no concurrency or synchronization.
package iterx
