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

package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMiddlewareFunc tests MiddlewareFunc
func TestMiddlewareFunc(t *testing.T) {
	mw := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "value")
			next.ServeHTTP(w, r)
		})
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := mw.HTTPHandler(handler)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	wrapped.ServeHTTP(rec, req)

	if rec.Header().Get("X-Test") != "value" {
		t.Errorf("expected header 'value', got '%s'", rec.Header().Get("X-Test"))
	}
}

// TestMiddlewares tests Middlewares
func TestMiddlewares(t *testing.T) {
	order := ""
	mw1 := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order += "1"
			next.ServeHTTP(w, r)
		})
	})

	mw2 := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order += "2"
			next.ServeHTTP(w, r)
		})
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order += "H"
		w.WriteHeader(http.StatusOK)
	})

	middlewares := Middlewares{mw1, mw2}
	wrapped := middlewares.HTTPHandler(handler)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	wrapped.ServeHTTP(rec, req)

	// Execution order should be: mw2 wraps handler, mw1 wraps mw2
	// So execution is: mw1 -> mw2 -> handler
	if order != "12H" {
		t.Errorf("expected execution order '12H', got '%s'", order)
	}
}

// TestMiddlewaresEmpty tests empty Middlewares
func TestMiddlewaresEmpty(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middlewares := Middlewares{}
	wrapped := middlewares.HTTPHandler(handler)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
}

// TestMiddlewaresSort tests Sort method
func TestMiddlewaresSort(t *testing.T) {
	order := ""

	low := PriorityMiddlewareFunc(1, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order += "L"
			next.ServeHTTP(w, r)
		})
	})

	high := PriorityMiddlewareFunc(10, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order += "H"
			next.ServeHTTP(w, r)
		})
	})

	// Unsorted order: low, high
	middlewares := Middlewares{low, high}
	middlewares.Sort()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order += "X"
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middlewares.HTTPHandler(handler)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	wrapped.ServeHTTP(rec, req)

	// After sorting: high (10), low (1)
	// Execution: high wraps low wraps handler
	// So order: H -> L -> X
	if order != "HLX" {
		t.Errorf("expected execution order 'HLX', got '%s'", order)
	}
}

// TestMiddlewaresSortWithDefaultPriority tests Sort with default priority
func TestMiddlewaresSortWithDefaultPriority(t *testing.T) {
	order := ""

	priority := PriorityMiddlewareFunc(100, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order += "P"
			next.ServeHTTP(w, r)
		})
	})

	// Middleware without priority (default 1)
	normal := MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order += "N"
			next.ServeHTTP(w, r)
		})
	})

	// Unsorted: normal, priority
	middlewares := Middlewares{normal, priority}
	middlewares.Sort()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order += "H"
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middlewares.HTTPHandler(handler)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	wrapped.ServeHTTP(rec, req)

	// After sorting: priority (100), normal (1)
	// Execution: priority wraps normal wraps handler
	// So order: P -> N -> H
	if order != "PNH" {
		t.Errorf("expected execution order 'PNH', got '%s'", order)
	}
}
