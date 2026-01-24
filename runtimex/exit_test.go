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
	"time"
)

func TestExit(t *testing.T) {
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

func TestExitContext(t *testing.T) {
	originalExit := GetExitFunc()
	defer SetExitFunc(originalExit)

	t.Run("ExitContext returns non-nil context", func(t *testing.T) {
		ctx := ExitContext()
		if ctx == nil {
			t.Error("ExitContext should not return nil")
		}
	})

	t.Run("ExitContext is cancelled when Exit is called", func(t *testing.T) {
		ctx := ExitContext()

		// Set up a function that signals when context is done
		done := make(chan bool)
		go func() {
			<-ctx.Done()
			done <- true
		}()

		// Set exit function that doesn't actually exit
		SetExitFunc(func(int) {})

		// Call Exit which should cancel the context
		Exit(1)

		// Wait for context to be cancelled
		select {
		case <-done:
			// Context was cancelled, good
		case <-time.After(100 * time.Millisecond):
			t.Error("ExitContext should be cancelled when Exit is called")
		}
	})

	t.Run("ExitContext can be used for cleanup coordination", func(t *testing.T) {
		cleanupDone := make(chan bool)
		cleanupStarted := make(chan bool)

		// Start a goroutine that waits for context cancellation
		go func() {
			ctx := ExitContext()
			cleanupStarted <- true
			<-ctx.Done()
			cleanupDone <- true
		}()

		// Wait for cleanup goroutine to start
		<-cleanupStarted

		// Set exit function that signals before returning
		exitCalled := make(chan bool)
		SetExitFunc(func(int) {
			exitCalled <- true
		})

		// Call Exit in a goroutine
		go Exit(1)

		// First the cleanup should complete
		select {
		case <-cleanupDone:
			// Good, cleanup completed
		case <-exitCalled:
			t.Error("Cleanup should complete before exit function is called")
			return
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for cleanup")
			return
		}

		// Then the exit function should be called
		select {
		case <-exitCalled:
			// Good, exit function called after cleanup
		case <-time.After(100 * time.Millisecond):
			t.Error("Exit function should be called after context cancellation")
		}
	})

	t.Run("Multiple Exit calls cancel context only once", func(t *testing.T) {
		ctx := ExitContext()
		cancelCount := 0
		var mu sync.Mutex

		// Monitor context cancellation
		done := make(chan bool)
		go func() {
			<-ctx.Done()
			mu.Lock()
			cancelCount++
			mu.Unlock()
			done <- true
		}()

		SetExitFunc(func(int) {})

		// Call Exit multiple times
		Exit(1)
		Exit(2)
		Exit(3)

		// Wait for context cancellation
		select {
		case <-done:
			// Context was cancelled
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for context cancellation")
			return
		}

		mu.Lock()
		count := cancelCount
		mu.Unlock()

		if count != 1 {
			t.Errorf("ExitContext should be cancelled only once, got %d cancellations", count)
		}
	})
}
