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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xgfone/go-toolkit/httpx"
)

func TestResponseMiddleware(t *testing.T) {
	t.Run("no error in context", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := Response(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ctx := httpx.AcquireContext()
		defer httpx.ReleaseContext(ctx)

		ctx.Reset(rec, req)
		req = req.WithContext(httpx.SetContext(req.Context(), ctx))

		middleware.ServeHTTP(rec, req)

		if !called {
			t.Error("handler should be called")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("error in context with no status written", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			ctx := httpx.GetContext(r.Context())
			if ctx != nil {
				ctx.Error = errors.New("test error")
			}
		})

		middleware := Response(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ctx := httpx.AcquireContext()
		defer httpx.ReleaseContext(ctx)

		ctx.Reset(rec, req)
		req = req.WithContext(httpx.SetContext(req.Context(), ctx))

		middleware.ServeHTTP(rec, req)

		if !called {
			t.Error("handler should be called")
		}

		if rec.Code == 0 {
			t.Error("status code should be written")
		}
	})

	t.Run("error in context with status already written", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
			ctx := httpx.GetContext(r.Context())
			if ctx != nil {
				ctx.Error = errors.New("test error after write")
			}
		})

		middleware := Response(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ctx := httpx.AcquireContext()
		defer httpx.ReleaseContext(ctx)

		ctx.Reset(rec, req)
		req = req.WithContext(httpx.SetContext(req.Context(), ctx))

		middleware.ServeHTTP(rec, req)

		if !called {
			t.Error("handler should be called")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})

	t.Run("no context", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := Response(handler)
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
}
