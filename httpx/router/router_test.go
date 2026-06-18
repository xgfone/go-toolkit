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

package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xgfone/go-toolkit/httpx"
)

func TestRouter_SetBackend(t *testing.T) {
	router := New()

	// Test valid backend
	router.SetBackend(func(routes []httpx.Route, notfound http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	// Test nil backend should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetBackend with nil should panic")
		}
	}()
	router2 := New()
	router2.SetBackend(nil)
}

func TestRouter_SetNotFound(t *testing.T) {
	router := New()

	// Test valid handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	router.SetNotFound(handler)

	// Test nil handler should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetNotFound with nil should panic")
		}
	}()
	router2 := New()
	router2.SetNotFound(nil)

}

func TestRouter_Use(t *testing.T) {
	router := New()

	// Add middleware
	middleware := httpx.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})

	router.Use(middleware)

	// Test that middleware is added (indirectly by testing ServeHTTP)
	router.Register(httpx.Route{
		Path: "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestRouter_Register(t *testing.T) {
	router := New()

	// Test valid route
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Register(httpx.Route{
		Path:    "/test",
		Handler: handler,
	})

	// Test that route was added
	routes := router.Routes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route after registration, got %d", len(routes))
	}

	// Test empty path should panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Register with empty path should panic")
			}
		}()
		router.Register(httpx.Route{
			Path:    "",
			Handler: handler,
		})
	}()

	// Test nil handler should panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Register with nil handler should panic")
			}
		}()
		router.Register(httpx.Route{
			Path:    "/test2",
			Handler: nil,
		})
	}()
}

func TestRouter_Routes(t *testing.T) {
	router := New()

	// Initially should be empty
	routes := router.Routes()
	if len(routes) != 0 {
		t.Errorf("expected 0 routes initially, got %d", len(routes))
	}

	// Add a route
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Register(httpx.Route{
		Path:    "/test",
		Handler: handler,
	})

	routes = router.Routes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route after registration, got %d", len(routes))
	}

	if routes[0].Path != "/test" {
		t.Errorf("expected path '/test', got '%s'", routes[0].Path)
	}
}

func TestRouter_ServeHTTP(t *testing.T) {
	router := New()

	// Register a test route
	called := false
	router.Register(httpx.Route{
		Path: "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
	})

	// Test registered route
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if !called {
		t.Error("handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Test not found
	req = httptest.NewRequest("GET", "/notfound", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}
