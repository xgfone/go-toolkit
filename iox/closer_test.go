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

package iox

import (
	"errors"
	"testing"
)

func TestCloserFunc(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		err := CloserFunc(func() error { return nil }).Close()
		if err != nil {
			t.Errorf("expect nil, but got an error: %v", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		expect := errors.New("test")
		err := CloserFunc(func() error { return expect }).Close()
		if err == nil {
			t.Error("expect an error, but got nil")
		} else if expect != err {
			t.Errorf("expect error %v, but got %v", expect, err)
		}
	})

	t.Run("nil panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expect a panic, but nil")
			}
		}()
		CloserFunc(nil).Close()
	})
}
