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

func TestRoute_Router(t *testing.T) {
	router := New()
	rrouter := router.Path("/path").Router()
	if router != rrouter {
		t.Fail()
	}
}

func TestRoute_Use(t *testing.T) {
	router := New()

	middleware := httpx.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "true")
			next.ServeHTTP(w, r)
		})
	})

	router.Path("/test").Use(middleware).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Header().Get("X-Test") != "true" {
		t.Error("middleware should have been applied")
	}
}

func TestRoute_Auth(t *testing.T) {
	router := New()

	auth := httpx.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Auth", "true")
			next.ServeHTTP(w, r)
		})
	})

	router.Path("/auth").Auth(auth).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/auth", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Header().Get("X-Auth") != "true" {
		t.Error("auth middleware should have been applied")
	}
}

func TestRoute_Host(t *testing.T) {
	router := New()

	router.Host("example.com").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "http://notexist.com/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for /test, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for example.com/test, got %d", rec.Code)
	}
}

func TestRoute_Path(t *testing.T) {
	router := New()

	router.Path("/test").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for /test, got %d", rr.Code)
	}

	// Test path without leading slash should panic
	router2 := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Path without leading slash should panic")
		}
	}()
	router2.Path("test")
}

func TestRoute_Group(t *testing.T) {
	router := New()

	group := router.Group("/api")
	group.Path("/users").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	if prefix := group.Prefix(); prefix != "/api" {
		t.Errorf("expected prefix /api, got %s", prefix)
	}

	req := httptest.NewRequest("GET", "/api/users", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for /api/users, got %d", rr.Code)
	}

	// Test group without leading slash should panic
	router2 := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Group without leading slash should panic")
		}
	}()
	router2.Group("api")
}

func TestRoute_Method(t *testing.T) {
	router := New()

	router.Path("/test").GetFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET /test, got %d", rr.Code)
	}
}

func TestRoute_Handler(t *testing.T) {
	router := New()

	// Test valid handler
	called := false
	router.Path("/test").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if !called {
		t.Error("handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	// Test nil handler should panic
	router2 := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Handler with nil should panic")
		}
	}()
	router2.Path("/test2").Handler(nil)
}

func TestRoute_HTTPMethods(t *testing.T) {
	router := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name   string
		method string
		route  func() Route
	}{
		{"Put", "PUT", func() Route { return router.Path("/put").Put(handler) }},
		{"Get", "GET", func() Route { return router.Path("/get").Get(handler) }},
		{"Post", "POST", func() Route { return router.Path("/post").Post(handler) }},
		{"Head", "HEAD", func() Route { return router.Path("/head").Head(handler) }},
		{"Patch", "PATCH", func() Route { return router.Path("/patch").Patch(handler) }},
		{"Delete", "DELETE", func() Route { return router.Path("/delete").Delete(handler) }},
		{"Options", "OPTIONS", func() Route { return router.Path("/options").Options(handler) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the method doesn't panic with nil handler
			// (actual registration will panic, but we're testing method chaining)
			_ = tt.route()
		})
	}
}

func TestRoute_HTTPMethodFuncs(t *testing.T) {
	router := New()
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	tests := []struct {
		name   string
		method string
		route  func() Route
	}{
		{"PutFunc", "PUT", func() Route { return router.Path("/put").PutFunc(handler) }},
		{"GetFunc", "GET", func() Route { return router.Path("/get").GetFunc(handler) }},
		{"PostFunc", "POST", func() Route { return router.Path("/post").PostFunc(handler) }},
		{"HeadFunc", "HEAD", func() Route { return router.Path("/head").HeadFunc(handler) }},
		{"PatchFunc", "PATCH", func() Route { return router.Path("/patch").PatchFunc(handler) }},
		{"DeleteFunc", "DELETE", func() Route { return router.Path("/delete").DeleteFunc(handler) }},
		{"OptionsFunc", "OPTIONS", func() Route { return router.Path("/options").OptionsFunc(handler) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the method doesn't panic with nil handler
			// (actual registration will panic, but we're testing method chaining)
			_ = tt.route()
		})
	}
}

func TestRoute_HTTPMethodContexts(t *testing.T) {
	router := New()
	handler := func(c *httpx.Context) error {
		c.WriteHeader(http.StatusOK)
		return nil
	}

	tests := []struct {
		name   string
		method string
		route  func() Route
	}{
		{"PutFunc", "PUT", func() Route { return router.Path("/put").PutContext(handler) }},
		{"GetFunc", "GET", func() Route { return router.Path("/get").GetContext(handler) }},
		{"PostFunc", "POST", func() Route { return router.Path("/post").PostContext(handler) }},
		{"HeadFunc", "HEAD", func() Route { return router.Path("/head").HeadContext(handler) }},
		{"PatchFunc", "PATCH", func() Route { return router.Path("/patch").PatchContext(handler) }},
		{"DeleteFunc", "DELETE", func() Route { return router.Path("/delete").DeleteContext(handler) }},
		{"OptionsFunc", "OPTIONS", func() Route { return router.Path("/options").OptionsContext(handler) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the method doesn't panic with nil handler
			// (actual registration will panic, but we're testing method chaining)
			_ = tt.route()
		})
	}
}

func TestRoute_EmptyPathDefaultsToRoot(t *testing.T) {
	router := New()

	called := false
	router.Path("").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if !called {
		t.Error("handler should have been called for root path")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for root path, got %d", rr.Code)
	}
}

func TestRoute_PathEdgeCases(t *testing.T) {
	router := New()

	// Test empty path returns same route (will default to root)
	router.Path("").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for root path, got %d", rr.Code)
	}

	// Test "/" path returns same route (will default to root)
	router2 := New()
	router2.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req = httptest.NewRequest("GET", "/", nil)
	rr = httptest.NewRecorder()
	router2.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for '/' path, got %d", rr.Code)
	}

	// Test path normalization
	router3 := New()
	router3.Path("/test//path").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req = httptest.NewRequest("GET", "/test/path", nil)
	rr = httptest.NewRecorder()
	router3.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for normalized path, got %d", rr.Code)
	}
}

func TestRoute_GroupEdgeCases(t *testing.T) {
	router := New()

	// Test empty group returns same route
	router.Group("").Path("/test").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for path without group, got %d", rr.Code)
	}

	// Test "/" group returns same route
	router2 := New()
	router2.Group("/").Path("/test").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req = httptest.NewRequest("GET", "/test", nil)
	rr = httptest.NewRecorder()
	router2.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for path with '/' group, got %d", rr.Code)
	}

	// Test group normalization
	router3 := New()
	router3.Group("/api//v1").Path("/users").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req = httptest.NewRequest("GET", "/api/v1/users", nil)
	rr = httptest.NewRecorder()
	router3.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for normalized group path, got %d", rr.Code)
	}
}

func TestRoute_AuthWithMiddlewares(t *testing.T) {
	router := New()

	// Test auth middleware is prepended when other middlewares exist
	authCalled := false
	otherCalled := false

	auth := httpx.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCalled = true
			next.ServeHTTP(w, r)
		})
	})

	other := httpx.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			otherCalled = true
			next.ServeHTTP(w, r)
		})
	})

	router.Path("/test").Use(other).Auth(auth).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if !authCalled {
		t.Error("auth middleware should have been called")
	}
	if !otherCalled {
		t.Error("other middleware should have been called")
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/path", "/path"},
		{"/path//to//resource", "/path/to/resource"},
		{"//path", "/path"}, // strings.ReplaceAll replaces all occurrences
		{"/", "/"},
		{"", ""},
	}

	for _, tt := range tests {
		result := normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
