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
	t.Run("success with no response written", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := httpx.AcquireContext()
		defer httpx.ReleaseContext(ctx)
		ctx.Reset(rec, req)
		req = req.WithContext(httpx.SetContext(req.Context(), ctx))

		handler := httpx.ContextHandler(func(*httpx.Context) error {
			return nil
		})
		Response(handler).ServeHTTP(ctx.ResponseWriter, req)

		if ctx.StatusCode() != http.StatusOK {
			t.Fatalf("context status = %d, want %d", ctx.StatusCode(), http.StatusOK)
		}
		if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
			t.Fatalf("response = %d %q, want empty 200", rec.Code, rec.Body.String())
		}
	})

	t.Run("no error in context", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		})

		middleware := Response(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		ctx := httpx.AcquireContext()
		defer httpx.ReleaseContext(ctx)

		ctx.Reset(rec, req)
		req = req.WithContext(httpx.SetContext(req.Context(), ctx))

		middleware.ServeHTTP(ctx.ResponseWriter, req)

		if !called {
			t.Error("handler should be called")
		}
		if rec.Code != http.StatusNoContent ||
			ctx.StatusCode() != http.StatusNoContent {
			t.Errorf("response/context status = %d/%d, want 204",
				rec.Code, ctx.StatusCode())
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

		middleware.ServeHTTP(ctx.ResponseWriter, req)

		if !called {
			t.Error("handler should be called")
		}

		if rec.Code != http.StatusInternalServerError ||
			ctx.StatusCode() != http.StatusInternalServerError {
			t.Errorf("response/context status = %d/%d, want 500",
				rec.Code, ctx.StatusCode())
		}
	})

	t.Run("error in context with status already written", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("accepted"))
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

		middleware.ServeHTTP(ctx.ResponseWriter, req)

		if !called {
			t.Error("handler should be called")
		}

		if rec.Code != http.StatusAccepted || ctx.StatusCode() != http.StatusAccepted {
			t.Errorf("response/context status = %d/%d, want 202", rec.Code, ctx.StatusCode())
		}
		if rec.Body.String() != "accepted" {
			t.Errorf("response body = %q, want accepted", rec.Body.String())
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
