// Copyright 2024 xgfone
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

package random

import (
	"sync"
	"testing"
)

func TestString(t *testing.T) {
	for range 100 {
		s := String(16, DefaultCharset())
		if len(s) != 16 {
			t.Errorf("expect length of string is equal to 16, but got '%s'", s)
		}
	}
}

func TestSetDefaultCharset(t *testing.T) {
	orig := DefaultCharset()
	t.Cleanup(func() { SetDefaultCharset(orig) })

	SetDefaultCharset("x")
	if got := String(16, DefaultCharset()); got != "xxxxxxxxxxxxxxxx" {
		t.Fatalf("unexpected generated string: %q", got)
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected panic for empty charset")
			}
		}()
		SetDefaultCharset("")
	}()
	if got := DefaultCharset(); got != "x" {
		t.Fatalf("invalid update changed charset to %q", got)
	}
}

func TestDefaultCharsetConcurrent(t *testing.T) {
	orig := DefaultCharset()
	t.Cleanup(func() { SetDefaultCharset(orig) })
	SetDefaultCharset(NumCharset)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				SetDefaultCharset(HexLowerCharset)
				got := DefaultCharset()
				if got != NumCharset && got != HexLowerCharset {
					t.Errorf("unexpected charset: %q", got)
				}
				SetDefaultCharset(NumCharset)
			}
		})
	}
	wg.Wait()
}
