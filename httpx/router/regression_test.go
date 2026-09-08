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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xgfone/go-toolkit/httpx"
)

func TestRegressionHostRootNotFound(t *testing.T) {
	r := New()
	r.SetNotFound(httpx.Handler403)
	r.Host("a.example").Path("/").Handler(httpx.Handler200)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "http://b.example/missing", nil))
	if rec.Code != 403 {
		t.Fatalf("custom notfound bypassed for other host: %d", rec.Code)
	}
}

func TestRegressionRoutesReadRace(t *testing.T) {
	r := New()
	for i := 0; i < 1000; i++ {
		r.Path(fmt.Sprintf("/audit/%d", i)).Get(httpx.Handler200)
	}

	routes := r.Routes()
	done := make(chan struct{})
	go func() {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/audit/0", nil))
		close(done)
	}()

	defer func() { <-done }()
	for {
		select {
		case <-done:
			if !r.Routes()[0].Online {
				t.Error("a new snapshot must reflect successful registration")
			}
			return

		default:
			for i := range routes {
				if routes[i].Online {
					t.Error("an earlier snapshot was mutated during registration")
					return
				}
			}
		}
	}
}

func TestServeMuxRoutingSemantics(t *testing.T) {
	r := New()
	r.SetNotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Fallback", "true")
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, `{"missing":true}`)
	}))

	r.Path("/items/{id}").GetFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Error("matched handler lost the original writer's Flusher")
		}
		w.Header().Set("X-Pattern", r.Pattern)
		_, _ = io.WriteString(w, r.PathValue("id"))
	})

	r.Path("/files/").Get(httpx.Handler200)
	r.Path("/known404").GetFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "handler's own 404")
	})

	// The mux's redirect status depends on the Go version. Compare against
	// the active standard library instead of imposing a separate policy.
	standard := http.NewServeMux()
	standard.Handle("GET /items/{id}", httpx.Handler200)
	standard.Handle("GET /files/", httpx.Handler200)

	tests := []struct {
		method, path                   string
		status                         int
		body, location, allow, pattern string
	}{
		{"GET", "/items/42", 200, "42", "", "", "GET /items/{id}"},
		{"GET", "/items/a%2Fb", 200, "a/b", "", "", "GET /items/{id}"},
		{"HEAD", "/items/42", 200, "", "", "", "GET /items/{id}"},
		{"POST", "/items/42", 405, "", "", "GET, HEAD", ""},
		{"GET", "/files?x=1", 0, "", "/files/?x=1", "", ""},
		{"GET", "/items//42", 0, "", "/items/42", "", ""},
		{"GET", "/known404", 404, "handler's own 404", "", "", ""},
		{"GET", "/missing", 418, `{"missing":true}`, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			wantStatus := tt.status
			if tt.location != "" {
				baseline := httptest.NewRecorder()
				standard.ServeHTTP(baseline, httptest.NewRequest(tt.method, tt.path, nil))
				wantStatus = baseline.Code
			}

			rec := httptest.NewRecorder()
			rec.Header().Set("X-Upstream", "retained")
			r.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != wantStatus || rec.Header().Get("Location") != tt.location ||
				rec.Header().Get("Allow") != tt.allow || rec.Header().Get("X-Pattern") != tt.pattern {
				t.Fatalf("unexpected response: %d %v %s", rec.Code, rec.Header(), rec.Body)
			}
			if tt.body != "" && rec.Body.String() != tt.body {
				t.Fatalf("body = %q, want %q", rec.Body.String(), tt.body)
			}
			if rec.Header().Get("X-Upstream") != "retained" {
				t.Error("upstream headers were lost")
			}
			if tt.status == 418 {
				if rec.Header().Get("Content-Type") != "application/json" ||
					rec.Header().Get("X-Content-Type-Options") != "" {
					t.Error("ServeMux's discarded 404 headers leaked into the custom response")
				}
			} else if rec.Header().Get("X-Fallback") != "" {
				t.Error("custom NotFound ran for a matched route or routing response")
			}
		})
	}
}

func TestRegressionMethodNotAllowed(t *testing.T) {
	r := New()
	r.Path("/resource").Get(httpx.Handler200)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("POST", "/resource", nil))
	if rec.Code != 405 || rec.Header().Get("Allow") == "" {
		t.Fatalf("expected 405 with Allow, got %d with %v", rec.Code, rec.Header())
	}
}
