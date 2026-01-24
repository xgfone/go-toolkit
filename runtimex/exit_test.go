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
	"os"
	"sync"
	"testing"
)

func TestExit(t *testing.T) {
	// Save original exit function to restore after test
	originalExit := GetExitFunc()
	defer SetExitFunc(originalExit)

	t.Run("Exit calls configured function", func(t *testing.T) {
		var called bool
		var exitCode int

		SetExitFunc(func(code int) {
			called = true
			exitCode = code
		})

		Exit(42)

		if !called {
			t.Error("Exit should call the configured function")
		}
		if exitCode != 42 {
			t.Errorf("Exit should pass exit code to function, got %d, want 42", exitCode)
		}
	})

	t.Run("Exit with different codes", func(t *testing.T) {
		var codes []int
		SetExitFunc(func(code int) {
			codes = append(codes, code)
		})

		Exit(0)
		Exit(1)
		Exit(-1)
		Exit(255)

		if len(codes) != 4 {
			t.Errorf("Exit should be called 4 times, got %d", len(codes))
		}
		if codes[0] != 0 || codes[1] != 1 || codes[2] != -1 || codes[3] != 255 {
			t.Errorf("Exit codes mismatch, got %v", codes)
		}
	})
}

func TestSetExitFunc(t *testing.T) {
	originalExit := GetExitFunc()
	defer SetExitFunc(originalExit)

	t.Run("SetExitFunc with nil panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("SetExitFunc should panic when called with nil")
			}
			// Restore original after panic
			SetExitFunc(originalExit)
		}()

		SetExitFunc(nil)
	})

	t.Run("SetExitFunc overrides previous function", func(t *testing.T) {
		firstCalled := false
		secondCalled := false

		SetExitFunc(func(int) { firstCalled = true })
		Exit(1)

		if !firstCalled {
			t.Error("First function should be called")
		}

		// Reset firstCalled to check it's not called again
		firstCalled = false
		SetExitFunc(func(int) { secondCalled = true })
		Exit(2)

		if firstCalled {
			t.Error("First function should not be called after override")
		}
		if !secondCalled {
			t.Error("Second function should be called")
		}
	})

	t.Run("SetExitFunc can restore os.Exit", func(t *testing.T) {
		SetExitFunc(os.Exit)
		// Can't test os.Exit actually works without terminating process,
		// but we can verify GetExitFunc returns non-nil
		if GetExitFunc() == nil {
			t.Error("GetExitFunc should not return nil after setting os.Exit")
		}
	})
}

func TestGetExitFunc(t *testing.T) {
	originalExit := GetExitFunc()
	defer SetExitFunc(originalExit)

	t.Run("GetExitFunc returns configured function", func(t *testing.T) {
		var called bool
		customFunc := func(int) { called = true }

		SetExitFunc(customFunc)
		exitFunc := GetExitFunc()

		if exitFunc == nil {
			t.Error("GetExitFunc should not return nil")
		}

		// Call the returned function to verify it's the configured one
		exitFunc(0)
		if !called {
			t.Error("Function returned by GetExitFunc should be the configured function")
		}
	})

	t.Run("GetExitFunc reflects SetExitFunc changes", func(t *testing.T) {
		callCount := 0
		func1 := func(int) { callCount = 1 }
		func2 := func(int) { callCount = 2 }

		SetExitFunc(func1)
		GetExitFunc()(0)
		if callCount != 1 {
			t.Errorf("GetExitFunc should return first function, got callCount=%d", callCount)
		}

		SetExitFunc(func2)
		GetExitFunc()(0)
		if callCount != 2 {
			t.Errorf("GetExitFunc should return second function, got callCount=%d", callCount)
		}
	})
}

func TestExitConcurrent(t *testing.T) {
	originalExit := GetExitFunc()
	defer SetExitFunc(originalExit)

	t.Run("Concurrent Exit calls", func(t *testing.T) {
		const goroutines = 100
		var mu sync.Mutex
		callCount := 0

		SetExitFunc(func(int) {
			mu.Lock()
			callCount++
			mu.Unlock()
		})

		var wg sync.WaitGroup
		wg.Add(goroutines)
		for range goroutines {
			go func() {
				defer wg.Done()
				Exit(0)
			}()
		}
		wg.Wait()

		if callCount != goroutines {
			t.Errorf("Exit should be called %d times, got %d", goroutines, callCount)
		}
	})

	t.Run("Concurrent SetExitFunc and GetExitFunc", func(t *testing.T) {
		const goroutines = 50
		var wg sync.WaitGroup

		wg.Add(goroutines * 2)
		for range goroutines {
			go func() {
				defer wg.Done()
				SetExitFunc(func(int) {})
			}()
			go func() {
				defer wg.Done()
				_ = GetExitFunc()
			}()
		}
		wg.Wait()
		// Test passes if no panic occurs
	})
}

func TestExitEdgeCases(t *testing.T) {
	originalExit := GetExitFunc()
	defer SetExitFunc(originalExit)

	t.Run("Exit function can panic", func(t *testing.T) {
		SetExitFunc(func(int) {
			panic("exit panic")
		})

		defer func() {
			if r := recover(); r == nil {
				t.Error("Exit should panic when exit function panics")
			}
		}()

		Exit(1)
	})

	t.Run("Exit with function that changes itself", func(t *testing.T) {
		var firstCalled, secondCalled bool

		SetExitFunc(func(int) {
			firstCalled = true
			// Change exit function during execution
			SetExitFunc(func(int) {
				secondCalled = true
			})
		})

		Exit(1)
		if !firstCalled {
			t.Error("First function should be called")
		}

		Exit(2)
		if !secondCalled {
			t.Error("Second function should be called on subsequent Exit")
		}
	})
}
