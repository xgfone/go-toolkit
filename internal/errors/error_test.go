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

package errors

import (
	stderrs "errors"
	"fmt"
	"testing"
)

func TestAsType(t *testing.T) {
	testAsType(t, "nil", error(nil), (*pointerError)(nil), false)
	testAsType(t, "direct concrete value", valueError{"direct"}, valueError{"direct"}, true)
	testAsType(t, "wrapped concrete value", fmt.Errorf("wrap: %w", valueError{"wrapped"}), valueError{"wrapped"}, true)
	testAsType(t, "not found", stderrs.New("plain"), valueError{}, false)
}

type comparableError interface {
	comparable
	error
}

func testAsType[E comparableError](t *testing.T, name string, err error, want E, wantOK bool) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		got, gotOK := AsType[E](err)
		if gotOK != wantOK {
			t.Fatalf("ok = %v, want %v", gotOK, wantOK)
		}
		if got != want {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

type valueError struct {
	msg string
}

func (e valueError) Error() string { return e.msg }

type pointerError struct {
	msg string
}

func (e *pointerError) Error() string { return e.msg }
