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

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xgfone/go-toolkit/httpx"
	"github.com/xgfone/go-toolkit/runtimex"
)

func TestRecoverMiddleware(t *testing.T) {
	t.Run("no panic", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := Recover(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		middleware.ServeHTTP(rec, req)

		if !called {
			t.Error("handler should be called")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("panic without context", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		middleware := Recover(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
		if body := rec.Body.String(); body != "panic\n" {
			t.Errorf("expected body 'panic\\n', got %q", body)
		}
	})

	t.Run("panic with context", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic with context")
		})

		middleware := Recover(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ctx := httpx.AcquireContext()
		defer httpx.ReleaseContext(ctx)
		req = req.WithContext(httpx.SetContext(req.Context(), ctx))

		middleware.ServeHTTP(rec, req)

		if ctx.Error == nil {
			t.Error("error should be set in context")
		}
		if errStr := ctx.Error.Error(); errStr != "panic: test panic with context" {
			t.Errorf("expected error 'panic: test panic with context', got %q", errStr)
		}
	})

	t.Run("panic with httpx.ResponseWriter already written", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			panic("test panic after write")
		})

		middleware := Recover(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("panic with httpx.ResponseWriter already written status", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rw, ok := w.(httpx.ResponseWriter); ok {
				w.WriteHeader(http.StatusOK)
				if rw.StatusCode() == 0 {
					t.Error("StatusCode() should not be 0 after WriteHeader")
				}
				panic("test panic after status written")
			} else {
				t.Error("w should be httpx.ResponseWriter")
			}
		})

		middleware := Context(Recover(handler))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		if body := rec.Body.String(); body == "panic\n" {
			t.Error("should not send panic error page when w is httpx.ResponseWriter and status already written")
		}
	})
}

func TestPanicError(t *testing.T) {
	stacks := []struct {
		File, Func string
		Line       int
	}{
		{"file1.go", "Func1", 10},
		{"file2.go", "Func2", 20},
	}

	var frames []runtimex.Frame
	for _, s := range stacks {
		frames = append(frames, runtimex.Frame{
			File: s.File,
			Func: s.Func,
			Line: s.Line,
		})
	}

	err := panicerror{
		panicv: "test panic",
		stacks: frames,
	}

	expectedErr := "panic: test panic"
	if errStr := err.Error(); errStr != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, errStr)
	}

	if err.Stacks() == nil {
		t.Error("stacks should not be nil")
	}
}
