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

package runtimex

import (
	"context"
	"testing"
)

func TestRecover(t *testing.T) {
	t.Run("no panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("unexpected panic: %v", r)
			}
		}()

		// 正常执行，没有 panic
		defer Recover(context.Background())
		// 正常执行一些操作
		_ = 1 + 1
	})

	t.Run("panic with background context", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic should have been recovered: %v", r)
				}
			}()

			defer Recover(context.Background())
			panic("test panic with background context")
		}()
	})

	t.Run("panic with custom context", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic should have been recovered: %v", r)
				}
			}()

			ctx := context.WithValue(context.Background(), "key", "value")
			defer Recover(ctx)
			panic("test panic with custom context")
		}()
	})

	t.Run("panic with log args", func(t *testing.T) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic should have been recovered: %v", r)
				}
			}()

			defer Recover(context.Background(), "key1", "value1", "key2", 123)
			panic("test panic with log args")
		}()
	})

	t.Run("panic with various panic values", func(t *testing.T) {
		testCases := []struct {
			name  string
			panic any
		}{
			{"string panic", "error message"},
			{"error panic", "error string"},
			{"int panic", 42},
			{"struct panic", struct{ msg string }{msg: "struct panic"}},
			{"nil panic", nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("panic should have been recovered: %v", r)
						}
					}()

					defer Recover(context.Background())
					panic(tc.panic)
				}()
			})
		}
	})

	t.Run("multiple defers with recover", func(t *testing.T) {
		recoverCount := 0
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic should have been recovered by Recover: %v", r)
				}
			}()

			defer func() {
				recoverCount++
			}()

			defer Recover(context.Background())
			panic("test multiple defers")
		}()

		if recoverCount != 1 {
			t.Errorf("expected recoverCount to be 1, got %d", recoverCount)
		}
	})

	t.Run("recover in goroutine", func(t *testing.T) {
		done := make(chan bool)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic should have been recovered: %v", r)
				}
				done <- true
			}()

			defer Recover(context.Background())
			panic("goroutine panic")
		}()

		<-done
	})
}
