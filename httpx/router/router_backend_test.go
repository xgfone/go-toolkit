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
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/httpx"
)

func TestNewServeMuxBackend(t *testing.T) {
	// Test with root route
	routes := []httpx.Route{
		{
			Path: "/",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
	}

	notfound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := newServeMuxBackend(routes, notfound)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for root route, got %d", rr.Code)
	}

	// Test without root route
	routes = []httpx.Route{
		{
			Path: "/api",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
	}

	handler = newServeMuxBackend(routes, notfound)

	req = httptest.NewRequest("GET", "/", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for root without route, got %d", rr.Code)
	}
}

func TestNewServeMuxBackend_WildcardRoutes(t *testing.T) {
	// Test with catch-all wildcard route (should not add catch-all notfound handler)
	routes := []httpx.Route{
		{
			Path: "/{rest...}",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("catch-all route"))
			}),
		},
	}

	notfound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := newServeMuxBackend(routes, notfound)

	// Test that catch-all route matches any path
	req := httptest.NewRequest("GET", "/any/path", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for catch-all route, got %d", rr.Code)
	}

	// Test with parameter wildcard (should still add catch-all notfound handler)
	routes2 := []httpx.Route{
		{
			Path: "/api/{id}",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("parameter route"))
			}),
		},
	}

	handler2 := newServeMuxBackend(routes2, notfound)

	// Test root path should return 404 because parameter route is not a catch-all
	req2 := httptest.NewRequest("GET", "/", nil)
	rr2 := httptest.NewRecorder()
	handler2.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for root without catch-all route, got %d", rr2.Code)
	}
}

func TestNewServeMuxBackend_RegisterRoute(t *testing.T) {
	origlogger := slog.Default()
	defer func() { slog.SetDefault(origlogger) }()

	buf := bytes.NewBuffer(nil)
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))

	route1 := &httpx.Route{Path: "/", Method: "GET", Handler: httpx.Handler201}
	route2 := &httpx.Route{Path: "/", Method: "GET", Handler: httpx.Handler204}

	server := http.NewServeMux()
	registerRoute(server, route1)
	registerRoute(server, route2)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	server.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Errorf("expect status code %d, but got %d", 201, rec.Code)
	}

	if !route1.Online {
		t.Error("expect that route1 is registered, but got not")
	}
	if route2.Online {
		t.Error("expect that route1 is not registered, but registered")
	}

	const expected = "fail to register the http route"
	if s := strings.TrimSpace(buf.String()); !strings.Contains(s, expected) {
		t.Errorf("expect log message to contain '%s', but got '%s'", expected, s)
	}
}
