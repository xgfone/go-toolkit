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

import (
	"errors"
	"fmt"
	"testing"
)

type testErrorRedactor struct {
	msg string
}

func (e testErrorRedactor) Error() string {
	return "raw:" + e.msg
}

func (e testErrorRedactor) RedactError() string {
	return "redacted:" + e.msg
}

func TestRedactError(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if got := RedactError(nil); got != "" {
			t.Fatalf("RedactError(nil) = %q, want empty", got)
		}
	})

	t.Run("direct", func(t *testing.T) {
		err := testErrorRedactor{msg: "direct"}

		if got := RedactError(err); got != "redacted:direct" {
			t.Fatalf("RedactError(direct) = %q, want %q", got, "redacted:direct")
		}
	})

	t.Run("wrapped", func(t *testing.T) {
		err := fmt.Errorf("outer: %w", testErrorRedactor{msg: "wrapped"})

		if got := RedactError(err); got != "redacted:wrapped" {
			t.Fatalf("RedactError(wrapped) = %q, want %q", got, "redacted:wrapped")
		}
	})

	t.Run("fallback", func(t *testing.T) {
		err := errors.New("plain error")

		if got := RedactError(err); got != "plain error" {
			t.Fatalf("RedactError(fallback) = %q, want %q", got, "plain error")
		}
	})
}
