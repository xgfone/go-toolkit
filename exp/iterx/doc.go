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
// [FromSeq] and [FromSeq2] wrap standard iterators in [Stream] and [Stream2].
// [Stream.Map], [Stream.FilterMap], [Stream.Filter], [Stream.Take], and
// [Stream.Skip] use Rust-inspired names with Go signatures, as do their
// [Stream2.Map], [Stream2.FilterMap], [Stream2.Filter], [Stream2.Take], and
// [Stream2.Skip] counterparts.
// The mapping methods preserve the number of values yielded per element, while
// allowing their types to change. [Stream.FilterMap] callbacks return (value, ok),
// and [Stream2.FilterMap] callbacks return (key, value, ok).
// [Stream2.Take] and [Stream2.Skip] count each pair as one element and preserve
// the keys of remaining pairs, including slice indices.
//
// [Stream.To2] converts each value into a pair; [Stream2.To] converts each pair into
// one value. [Stream2.Keys] and [Stream2.Values] project either side of each pair.
// [Stream.Drop] and [Stream2.Drop] are aliases for [Stream.Skip] and [Stream2.Skip].
//
// Call [Stream.Seq] or [Stream2.Seq2] to use range, standard library collectors,
// or terminal operations from [github.com/xgfone/go-toolkit/iterx]. Streams do
// not start the input until iteration begins, cache values, or add concurrency.
// They preserve the input's order and stop it when the consumer stops. Repeated
// iteration depends on the input; a single-use iterator remains single-use.
//
// A nil iterator is not an empty sequence. Zero-value wrappers contain nil
// iterators and must not be iterated. Callbacks must be non-nil when called.
package iterx
