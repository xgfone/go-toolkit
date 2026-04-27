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

// Package errorx provides some extra errors.
package errorx

import "strings"

const defaultSensitiveMessage = "[REDACTED]"

// Sensitive wraps an error with a safe message and returns a *SensitiveError,
// or nil if err is nil.
func Sensitive(err error, safe string) error {
	if err == nil {
		return nil
	}
	return NewSensitiveError(err, safe)
}

// NewSensitiveError returns a new SensitiveError from err and a safe message.
//
// If err is nil, it returns nil.
// If safe is empty, "[REDACTED]" is used.
func NewSensitiveError(err error, safe string) *SensitiveError {
	if err == nil {
		return nil
	}

	if safe = strings.TrimSpace(safe); safe == "" {
		safe = defaultSensitiveMessage
	}

	return &SensitiveError{
		safe: safe,
		err:  err,
	}
}

// SensitiveError is used to wrap an error that maybe contain the sensitive information.
type SensitiveError struct {
	safe string
	err  error
}

// Error returns the safe message of the error.
func (e *SensitiveError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.safe
}

// Unwrap returns the original underlying error.
func (e *SensitiveError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// SensitiveError is similar to Unwrap, but it more explicitly indicates
// that the returned error may contain sensitive information.
func (e *SensitiveError) SensitiveError() error {
	return e.Unwrap()
}
