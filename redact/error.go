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

package redact

import "errors"

// ErrorRedactor is implemented by errors that can produce a redacted message.
type ErrorRedactor interface {
	RedactError() string
}

// RedactError returns a redacted message for err.
//
// Behavior:
//
//   - If err is nil, it returns an empty string.
//   - If err itself implements ErrorRedactor, that implementation is used.
//   - Otherwise, the wrapped error chain is searched with errors.As.
//   - If no ErrorRedactor is found, err.Error() is returned.
func RedactError(err error) string {
	if err == nil {
		return ""
	}

	if redactor, ok := err.(ErrorRedactor); ok {
		return redactor.RedactError()
	}

	var redactor ErrorRedactor
	if errors.As(err, &redactor) {
		return redactor.RedactError()
	}

	return err.Error()
}
