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

// Package stringx provides string helpers, desensitizers, and generators.
//
// [Desensitizer] abstracts string masking. [NewDesensitizer] returns an immutable
// [MaskDesensitizer] configured in rune counts; [MaskDesensitizer.WithLeft],
// [MaskDesensitizer.WithRight], and [MaskDesensitizer.WithChars] return copies.
// [PhoneDesensitizer] and the other presets expose only the [Desensitizer] interface.
// Setters such as [SetPhoneDesensitizer] replace the implementations atomically;
// previously returned instances are unaffected. Both retrieval and replacement
// support concurrent calls, but custom implementations must provide their own
// concurrency guarantees.
// [EmailDesensitizer] masks the local part of a single email address while
// preserving its domain; its implementation is replaceable with [SetEmailDesensitizer].
// [NewEmailDesensitizer] wraps a custom [MaskDesensitizer] for the local part and
// returns the [Desensitizer] interface.
//
// [Generator] produces strings and appends to byte buffers using byte lengths.
// [NewGenerator] adapts an append callback, and [NewAffixGenerator] adds a prefix
// and suffix to any generator. [DefaultGenerator] and [SetDefaultGenerator] provide
// synchronized access to a replaceable default implementation. Callers are
// responsible for the concurrency safety of their custom implementations.
// The time presets support concurrent generation while the [timex] configuration
// is stable and its configured clock function is safe for concurrent use.
package stringx
