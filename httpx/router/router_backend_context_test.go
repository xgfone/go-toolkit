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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xgfone/go-toolkit/httpx"
	"github.com/xgfone/go-toolkit/httpx/middleware"
)

func TestRoutingErrorWriterResponseController(t *testing.T) {
	c := httpx.AcquireContext()
	defer httpx.ReleaseContext(c)

	rec := httptest.NewRecorder()
	c.Reset(rec, httptest.NewRequest("GET", "/", nil))
	w := &routingErrorWriter{ResponseWriter: c.ResponseWriter, header: rec.Header().Clone()}

	// The controller must traverse both wrappers to reach the recorder's Flush.
	if err := http.NewResponseController(w).Flush(); err != nil {
		t.Fatal(err)
	}
	if !rec.Flushed {
		t.Fatal("Flush did not reach the underlying writer")
	}
}

func TestServeMuxBackendContext(t *testing.T) {
	tests := []struct {
		method, path string
		status       int
		body         string
	}{
		{"GET", "/matched", 200, "matched"},
		{"GET", "/own404", 404, "handler's own 404"},
		{"GET", "/missing", 418, "custom not found"},
		{"POST", "/matched", 405, ""},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			respond := func(status int, body string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					c := httpx.GetContext(r.Context())
					if w != c.ResponseWriter {
						t.Fatal("application handler received a different writer")
					}

					if c.StatusCode() != 0 || c.BytesWritten != 0 {
						t.Fatalf("response committed before handler: status=%d bytes=%d", c.StatusCode(), c.BytesWritten)
					}

					c.WriteHeader(status)
					if _, err := io.WriteString(c.ResponseWriter, body); err != nil {
						t.Fatal(err)
					}

					if err := http.NewResponseController(w).Flush(); err != nil {
						t.Fatal(err)
					}
				}
			}

			router := New()
			router.Path("/matched").Get(respond(200, "matched"))
			router.Path("/own404").Get(respond(404, "handler's own 404"))
			router.SetNotFound(respond(418, "custom not found"))

			rec := httptest.NewRecorder()
			h := middleware.Context(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				router.ServeHTTP(w, r)
				c := httpx.GetContext(r.Context())
				if c.StatusCode() != tt.status || c.BytesWritten != rec.Body.Len() {
					t.Errorf("incorrect context accounting: status=%d bytes=%d, response=%d %q",
						c.StatusCode(), c.BytesWritten, rec.Code, rec.Body.String())
				}
			}))

			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if tt.body != "" && (rec.Body.String() != tt.body || !rec.Flushed) {
				t.Fatalf("unexpected body or Flush: %q, flushed=%v", rec.Body.String(), rec.Flushed)
			}
		})
	}
}

func TestRegisterRouteClearsStaleOnline(t *testing.T) {
	r := New()
	for range 2 {
		r.Register(httpx.Route{Path: "/", Online: true, Handler: httpx.Handler200})
	}

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	routes := r.Routes()
	if !routes[0].Online || routes[1].Online {
		t.Fatalf("Online must describe this registration attempt: first=%v conflicting=%v", routes[0].Online, routes[1].Online)
	}
}
