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

// Package iterx provides experimental, chainable iterator transformations.
// It has no compatibility guarantees: APIs may change, be removed, or move
// outside exp at any time, including in v1 releases.
//
// FromSeq and FromSeq2 wrap standard iterators in Stream and Stream2. Map,
// FilterMap, Filter, Take, and Skip use Rust-inspired names with Go signatures.
// FilterMap callbacks return (value, ok). Map2 produces two values per element
// from either stream type; FilterMap2 transforms and selects existing pairs.
// Keys and Values switch to a one-value Stream. To, To2, FilterTo, FilterTo2,
// and Drop are aliases for the corresponding transformation methods.
//
// Call Seq on either wrapper to use range, standard library collectors, or
// terminal operations from github.com/xgfone/go-toolkit/iterx. Streams do not
// start the input until iteration begins, cache values, or add concurrency.
// They preserve the input's order and stop it when the consumer stops. Repeated
// iteration depends on the input; a single-use iterator remains single-use.
//
// A nil iterator is not an empty sequence. Zero-value wrappers contain nil
// iterators and must not be iterated. Callbacks must be non-nil when called.
package iterx
