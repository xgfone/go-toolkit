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

// Package redact provides small interfaces and helpers for producing
// redacted representations of values.
//
// The package is intentionally minimal:
//
//   - Redactor is for ordinary values that need to expose a redacted view.
//   - ErrorRedactor is for errors that need to expose a redacted message.
//   - Redact and RedactError apply those interfaces when present and otherwise
//     fall back to the original value or error text.
//
// Redaction behavior is controlled by Level. Each level describes the intended
// exposure range of the output, from fully trusted internal use to public use.
package redact
