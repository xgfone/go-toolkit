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
)

func TestBlockPathMiddleware(t *testing.T) {
	t.Run("matched path", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := BlockPath("/health", http.StatusServiceUnavailable)(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health?ready=false", nil)
		middleware.ServeHTTP(rec, req)

		if called {
			t.Error("handler should not be called")
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})

	t.Run("unmatched path", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := BlockPath("/health", http.StatusServiceUnavailable)(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		middleware.ServeHTTP(rec, req)

		if !called {
			t.Error("handler should be called")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}

func TestBlockPathPrefixMiddleware(t *testing.T) {
	t.Run("matched path prefix", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := BlockPathPrefix("/api/private/", http.StatusForbidden)(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/private/users", nil)
		middleware.ServeHTTP(rec, req)

		if called {
			t.Error("handler should not be called")
		}
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
		}
	})

	t.Run("unmatched path prefix", func(t *testing.T) {
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		middleware := BlockPathPrefix("/api/private/", http.StatusForbidden)(handler)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/public/users", nil)
		middleware.ServeHTTP(rec, req)

		if !called {
			t.Error("handler should be called")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
		}
	})
}
