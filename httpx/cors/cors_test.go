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

package cors

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCORSActualRequest(t *testing.T) {
	handler := Config{
		AllowOrigins:     []string{"https://example.com"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"X-Request-Id"},
	}.CORS(10).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("unexpected Access-Control-Allow-Credentials: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Request-Id" {
		t.Fatalf("unexpected Access-Control-Expose-Headers: %q", got)
	}
	if !varyContains(rec.Header(), "Origin") {
		t.Fatal("Vary does not contain Origin")
	}
}

func TestCORSNonPreflightOptionsRequest(t *testing.T) {
	called := false
	handler := Config{AllowOrigins: []string{"https://example.com"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Methods: %q", got)
	}
}

func TestCORSPreflightRequest(t *testing.T) {
	maxAge := 3600
	called := false
	config := NewDefaultConfig()
	config.AllowOrigins = []string{"https://example.com"}
	config.MaxAge = &maxAge
	handler := config.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	req.Header.Set("Access-Control-Request-Headers", "X-Token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("next handler was called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "X-Token" {
		t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Fatalf("unexpected Access-Control-Max-Age: %q", got)
	}
	for _, field := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
		if !varyContains(rec.Header(), field) {
			t.Fatalf("Vary does not contain %s", field)
		}
	}
}

func TestCORSDefaultConfigPreflight(t *testing.T) {
	handler := NewDefaultConfig().CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler was called")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "HEAD, GET, POST, PUT, PATCH, DELETE" {
		t.Fatalf("unexpected Access-Control-Allow-Methods: %q", got)
	}
	if varyContains(rec.Header(), "Origin") {
		t.Fatal("Vary contains Origin for static wildcard origin")
	}
	for _, field := range []string{"Access-Control-Request-Method", "Access-Control-Request-Headers"} {
		if !varyContains(rec.Header(), field) {
			t.Fatalf("Vary does not contain %s", field)
		}
	}
}

func TestCORSPreflightForbidden(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		origin         string
		requestMethod  string
		requestHeaders string
	}{
		{
			name:          "disallowed method",
			config:        Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: []string{http.MethodGet}},
			origin:        "https://example.com",
			requestMethod: http.MethodPut,
		},
		{
			name:          "invalid request method token",
			config:        Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: []string{http.MethodPut}},
			origin:        "https://example.com",
			requestMethod: "BAD METHOD",
		},
		{
			name:          "disallowed origin",
			config:        Config{AllowOrigins: []string{"https://allowed.example"}, AllowMethods: []string{http.MethodPut}},
			origin:        "https://denied.example",
			requestMethod: http.MethodPut,
		},
		{
			name:           "disallowed header",
			config:         Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: []string{http.MethodPut}, AllowHeaders: []string{"X-Allowed"}},
			origin:         "https://example.com",
			requestMethod:  http.MethodPut,
			requestHeaders: "X-Denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.config.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("next handler was called")
			}))

			req := httptest.NewRequest(http.MethodOptions, "/", nil)
			req.Header.Set("Origin", tt.origin)
			req.Header.Set("Access-Control-Request-Method", tt.requestMethod)
			if tt.requestHeaders != "" {
				req.Header.Set("Access-Control-Request-Headers", tt.requestHeaders)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusForbidden)
			}
			for _, header := range []string{
				"Access-Control-Allow-Origin",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
			} {
				if got := rec.Header().Get(header); got != "" {
					t.Fatalf("unexpected %s: %q", header, got)
				}
			}
		})
	}
}

func TestCORSPreflightAllowHeaders(t *testing.T) {
	allowPut := []string{http.MethodPut}
	tests := []struct {
		name           string
		config         Config
		requestHeaders []string
		want           string
	}{
		{
			name:           "wildcard reflects authorization",
			config:         Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: allowPut, AllowHeaders: []string{"*"}},
			requestHeaders: []string{"Authorization, X-Token"},
			want:           "Authorization, X-Token",
		},
		{
			name:           "explicit headers match case insensitively",
			config:         Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: allowPut, AllowHeaders: []string{"X-Token"}},
			requestHeaders: []string{"x-token"},
			want:           "X-Token",
		},
		{
			name:           "multiple request header lines",
			config:         Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: allowPut, AllowHeaders: []string{"X-Token", "X-Trace"}},
			requestHeaders: []string{"X-Token,", " X-Trace"},
			want:           "X-Token, X-Trace",
		},
		{
			name:           "default reflects comma separated request headers",
			config:         Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: allowPut},
			requestHeaders: []string{"X-One, X-Two, X-Three, X-Four"},
			want:           "X-One, X-Two, X-Three, X-Four",
		},
		{
			name:           "default reflects multiple request header lines",
			config:         Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: allowPut},
			requestHeaders: []string{"X-One", "X-Two", "X-Three", "X-Four"},
			want:           "X-One, X-Two, X-Three, X-Four",
		},
		{
			name:           "default ignores empty request header lines",
			config:         Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: allowPut},
			requestHeaders: []string{"", "X-Token"},
			want:           "X-Token",
		},
		{
			name:   "credentialed wildcard allows no request header",
			config: Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: allowPut, AllowCredentials: true, AllowHeaders: []string{"*"}},
		},
		{
			name:           "credentialed wildcard allows empty request header",
			config:         Config{AllowOrigins: []string{"https://example.com"}, AllowMethods: allowPut, AllowCredentials: true, AllowHeaders: []string{"*"}},
			requestHeaders: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.config.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("next handler was called")
			}))

			req := httptest.NewRequest(http.MethodOptions, "/", nil)
			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Access-Control-Request-Method", http.MethodPut)
			for _, requestHeader := range tt.requestHeaders {
				req.Header.Add("Access-Control-Request-Headers", requestHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNoContent {
				t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusNoContent)
			}
			if got := rec.Header().Get("Access-Control-Allow-Headers"); got != tt.want {
				t.Fatalf("unexpected Access-Control-Allow-Headers: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCORSCredentialedWildcardPreflightReflectsRequest(t *testing.T) {
	handler := Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler was called")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	req.Header.Set("Access-Control-Request-Headers", "X-Token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != http.MethodPatch {
		t.Fatalf("unexpected Access-Control-Allow-Methods: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "X-Token" {
		t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("unexpected Access-Control-Allow-Credentials: %q", got)
	}
}

func TestCORSActualRequestWithoutOrigin(t *testing.T) {
	called := false
	handler := Config{AllowOrigins: []string{"https://example.com"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("next handler was not called")
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if !varyContains(rec.Header(), "Origin") {
		t.Fatal("Vary does not contain Origin")
	}
}

func TestCORSNullOrigin(t *testing.T) {
	handler := Config{AllowOrigins: []string{"null"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "null")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "null" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
}

func TestCORSExposeWildcardWithCredentials(t *testing.T) {
	handler := Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
		ExposeHeaders:    []string{"*", "X-Request-Id"},
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Request-Id" {
		t.Fatalf("unexpected Access-Control-Expose-Headers: %q", got)
	}
}

func TestCORSDisallowedOriginPassesThroughWithVary(t *testing.T) {
	handler := Config{AllowOrigins: []string{"https://allowed.example", "https://other.example"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://denied.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if !varyContains(rec.Header(), "Origin") {
		t.Fatal("Vary does not contain Origin")
	}
}

func TestCORSZeroValueDisablesCORSHeaders(t *testing.T) {
	handler := Config{}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if varyContains(rec.Header(), "Origin") {
		t.Fatal("Vary contains Origin")
	}
}

func TestCORSWildcardStaticOrigin(t *testing.T) {
	handler := Config{AllowOrigins: []string{"*"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		origin string
	}{
		{name: "without request origin"},
		{name: "skips invalid request origin", origin: "https://example.com/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
				t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
			}
			if varyContains(rec.Header(), "Origin") {
				t.Fatal("Vary contains Origin for wildcard origin")
			}
		})
	}
}

func TestCORSWildcardStaticOriginWithExposeHeaders(t *testing.T) {
	handler := Config{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{"X-Request-Id"},
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Request-Id" {
		t.Fatalf("unexpected Access-Control-Expose-Headers: %q", got)
	}
}

func TestCORSStaticOriginRequiresMatchingRequestOrigin(t *testing.T) {
	handler := Config{AllowOrigins: []string{"https://allowed.example"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://denied.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if !varyContains(rec.Header(), "Origin") {
		t.Fatal("Vary does not contain Origin")
	}
}

func TestCORSMultipleExactOrigins(t *testing.T) {
	handler := Config{AllowOrigins: []string{"https://one.example", "https://two.example"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://two.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://two.example" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
}

func TestCORSVaryOriginMergesExistingHeader(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		next      func(http.ResponseWriter, *http.Request)
		setupResp func(http.Header)
	}{
		{
			name:   "next handler adds vary",
			config: Config{AllowOrigins: []string{"*"}, AllowCredentials: true},
			next: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Vary", "Accept-Encoding")
				w.WriteHeader(http.StatusOK)
			},
		},
		{
			name:   "response already has vary",
			config: Config{AllowOrigins: []string{"https://example.com"}},
			next: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			setupResp: func(h http.Header) {
				h.Set("Vary", "Accept-Encoding")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.config.CORS(0).HTTPHandler(http.HandlerFunc(tt.next))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", "https://example.com")
			rec := httptest.NewRecorder()
			if tt.setupResp != nil {
				tt.setupResp(rec.Header())
			}
			handler.ServeHTTP(rec, req)

			for _, field := range []string{"Accept-Encoding", "Origin"} {
				if !varyContains(rec.Header(), field) {
					t.Fatalf("Vary does not contain %s", field)
				}
			}
		})
	}
}

func TestCORSSubdomainPattern(t *testing.T) {
	tests := []struct {
		name        string
		allowOrigin string
		origin      string
		want        string
	}{
		{
			name:        "matches subdomain",
			allowOrigin: "https://*.example.com",
			origin:      "https://api.example.com",
			want:        "https://api.example.com",
		},
		{
			name:        "does not match root domain",
			allowOrigin: "https://*.example.com",
			origin:      "https://example.com",
		},
		{
			name:        "matches subdomain with port",
			allowOrigin: "https://*.example.com:8443",
			origin:      "https://api.example.com:8443",
			want:        "https://api.example.com:8443",
		},
		{
			name:        "does not match when port is missing",
			allowOrigin: "https://*.example.com:8443",
			origin:      "https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Config{AllowOrigins: []string{tt.allowOrigin}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Fatalf("unexpected Access-Control-Allow-Origin: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCORSNormalizesOrigin(t *testing.T) {
	tests := []struct {
		name   string
		config string
		origin string
		want   string
	}{
		{
			name:   "https default port",
			config: "https://example.com:443",
			origin: "https://example.com",
			want:   "https://example.com",
		},
		{
			name:   "http default port",
			config: "http://example.com:80",
			origin: "http://example.com",
			want:   "http://example.com",
		},
		{
			name:   "ipv6",
			config: "https://[0:0:0:0:0:0:0:1]",
			origin: "https://[::1]",
			want:   "https://[::1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Config{AllowOrigins: []string{tt.config}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Fatalf("unexpected Access-Control-Allow-Origin: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCORSConfigTrimsEmptyValues(t *testing.T) {
	handler := Config{
		AllowOrigins:  []string{"", " https://example.com "},
		AllowMethods:  []string{"", http.MethodPut, " "},
		AllowHeaders:  []string{"", "X-Token", " "},
		ExposeHeaders: []string{"", "X-Request-Id", " "},
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	preflight := httptest.NewRequest(http.MethodOptions, "/", nil)
	preflight.Header.Set("Origin", "https://example.com")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPut)
	preflight.Header.Set("Access-Control-Request-Headers", "X-Token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, preflight)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != http.MethodPut {
		t.Fatalf("unexpected Access-Control-Allow-Methods: %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "X-Token" {
		t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
	}

	actual := httptest.NewRequest(http.MethodGet, "/", nil)
	actual.Header.Set("Origin", "https://example.com")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, actual)

	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Request-Id" {
		t.Fatalf("unexpected Access-Control-Expose-Headers: %q", got)
	}
}

func TestCORSNormalizeHost(t *testing.T) {
	normalizeHost := func(host string) (string, bool) {
		switch host {
		case "例子.测试":
			return "xn--fsqu00a.xn--0zwm56d", true
		default:
			return host, true
		}
	}

	handler := Config{
		AllowOrigins:  []string{"https://例子.测试"},
		NormalizeHost: normalizeHost,
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://xn--fsqu00a.xn--0zwm56d")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://xn--fsqu00a.xn--0zwm56d" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
}

func TestCORSRequestOriginIsNotNormalized(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		origin string
	}{
		{
			name:   "uppercase scheme",
			config: Config{AllowOrigins: []string{"https://example.com"}},
			origin: "HTTPS://example.com",
		},
		{
			name:   "uppercase host",
			config: Config{AllowOrigins: []string{"https://example.com"}},
			origin: "https://EXAMPLE.com",
		},
		{
			name:   "default port",
			config: Config{AllowOrigins: []string{"https://example.com"}},
			origin: "https://example.com:443",
		},
		{
			name:   "leading zero port",
			config: Config{AllowOrigins: []string{"https://example.com:8443"}},
			origin: "https://example.com:08443",
		},
		{
			name:   "unicode host",
			config: Config{AllowOrigins: []string{"https://xn--fsqu00a.xn--0zwm56d"}},
			origin: "https://例子.测试",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.config.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
			}
		})
	}
}

func TestCORSSubdomainPatternRejectsOverlongHost(t *testing.T) {
	handler := Config{AllowOrigins: []string{"https://*.example.com"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://"+strings.Repeat("a", 250)+".example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
}

func TestCORSInvalidOriginIsNotReflected(t *testing.T) {
	handler := Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com/")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if !varyContains(rec.Header(), "Origin") {
		t.Fatal("Vary does not contain Origin")
	}
}

func TestCORSReflectsOnlyValidPreflightHeaders(t *testing.T) {
	handler := Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
		AllowMethods:     []string{http.MethodPut},
		AllowHeaders:     []string{"*"},
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler was called")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	req.Header.Set("Access-Control-Request-Headers", "X-Token, Bad Header")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
	}
}

func TestCORSMaxAgeZero(t *testing.T) {
	maxAge := 0
	handler := Config{
		AllowOrigins: []string{"https://example.com"},
		AllowMethods: []string{http.MethodPut},
		MaxAge:       &maxAge,
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler was called")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPut)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Max-Age"); got != "0" {
		t.Fatalf("unexpected Access-Control-Max-Age: %q", got)
	}
}

func TestCORSInvalidConfigPanics(t *testing.T) {
	negativeMaxAge := -1
	tests := []Config{
		{AllowOrigins: []string{"https://example.com/"}},
		{AllowOrigins: []string{"https://example.com:bad"}},
		{AllowOrigins: []string{"https://example.com:65536"}},
		{AllowOrigins: []string{"ftp://example.com"}},
		{AllowOrigins: []string{"https://api.*.example.com"}},
		{AllowOrigins: []string{"https://example.com"}, AllowMethods: []string{"BAD METHOD"}},
		{AllowOrigins: []string{"https://example.com"}, AllowHeaders: []string{"Bad Header"}},
		{AllowOrigins: []string{"https://example.com"}, MaxAge: &negativeMaxAge},
		{
			AllowOrigins: []string{"https://example.com"},
			NormalizeHost: func(host string) (string, bool) {
				return host + ":443", true
			},
		},
	}

	for _, config := range tests {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("Config.CORS did not panic for %+v", config)
				}
			}()
			_ = config.CORS(0)
		}()
	}
}

func TestCORSInvalidSubdomainPatternPanicsWithConfigError(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal("Config.CORS did not panic")
		}
		if got := err.(error).Error(); !strings.Contains(got, "cors.Config.AllowOrigins") {
			t.Fatalf("unexpected panic: %v", err)
		}
	}()

	_ = Config{AllowOrigins: []string{"https://*.example.com/path"}}.CORS(0)
}

func TestCORSPriority(t *testing.T) {
	c := Config{}.CORS(123).(*cors)
	if got := c.Priority(); got != 123 {
		t.Fatalf("unexpected priority: got %d, want %d", got, 123)
	}
}

func TestCORSHTTPHandlerNilPanics(t *testing.T) {
	assertPanics(t, func() {
		_ = Config{}.CORS(0).HTTPHandler(nil)
	})
}

func TestCORSServeHTTPWithoutNext(t *testing.T) {
	rec := httptest.NewRecorder()
	Config{}.CORS(0).(*cors).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Body.String(); got != "CORS: NO NEXT HANDLER" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestCORSOriginHelperEdgeCases(t *testing.T) {
	c := Config{AllowOrigins: []string{"https://*.example.com"}}.CORS(0).(*cors)
	if origin, ok := parseRequestOrigin("not an origin"); ok || c.matchSubdomainOrigin(origin) {
		t.Fatal("invalid origin parsed or matched subdomain pattern")
	}

	if _, ok := parseSubdomainOriginPattern("not an origin"); ok {
		t.Fatal("invalid subdomain origin pattern parsed")
	}
	if _, ok := parseSubdomainOriginPattern("https://*.*.example.com"); ok {
		t.Fatal("invalid wildcard suffix parsed")
	}

	if got, ok := normalizeSubdomainPattern("https://example.com", nil); ok || got != "" {
		t.Fatalf("unexpected normalized subdomain pattern: %q, %v", got, ok)
	}
	if got, ok := normalizeSubdomainPattern("https://*.example.com:bad", nil); ok || got != "" {
		t.Fatalf("unexpected normalized subdomain pattern: %q, %v", got, ok)
	}
	if got, ok := normalizeSubdomainPattern("https://*.*.example.com", nil); ok || got != "" {
		t.Fatalf("unexpected normalized subdomain pattern: %q, %v", got, ok)
	}
}

func TestCORSRequestOriginHelperEdgeCases(t *testing.T) {
	valids := []struct {
		origin string
		scheme string
		host   string
		port   string
	}{
		{
			origin: "null",
		},
		{
			origin: "http://example.com",
			scheme: "http",
			host:   "example.com",
		},
		{
			origin: "https://api_example.com:8443",
			scheme: "https",
			host:   "api_example.com",
			port:   "8443",
		},
		{
			origin: "ws://example.com:0",
			scheme: "ws",
			host:   "example.com",
			port:   "0",
		},
		{
			origin: "wss://[::1]:8443",
			scheme: "wss",
			host:   "::1",
			port:   "8443",
		},
	}

	for _, tt := range valids {
		t.Run("valid "+tt.origin, func(t *testing.T) {
			origin, ok := parseRequestOrigin(tt.origin)
			if !ok {
				t.Fatal("origin was rejected")
			}
			if origin.value != tt.origin || origin.scheme != tt.scheme ||
				origin.host != tt.host || origin.port != tt.port {
				t.Fatalf("unexpected parsed origin: %#v", origin)
			}
		})
	}

	invalids := []string{
		"",
		"https://example.com\n",
		"ftp://example.com",
		"https://example.com/path",
		"https://example.com?x=1",
		"https://example.com#fragment",
		"https://user@example.com",
		"https://[]",
		"https://[bad]",
		"https://[ABCD::1]",
		"https://[::1]x",
		"https://[::1]:443",
		"https://example.com]",
		"https://a:b:c",
		"https://:8443",
		"https://exa$mple.com",
		"https://example.com:",
		"https://example.com:08443",
	}

	for _, origin := range invalids {
		t.Run("invalid "+origin, func(t *testing.T) {
			if got, ok := parseRequestOrigin(origin); ok || got.value != "" {
				t.Fatalf("unexpected parsed origin: %#v, %v", got, ok)
			}
		})
	}
}

func TestCORSURLHelperEdgeCases(t *testing.T) {
	if _, ok := parseOriginURL("https://example.com bad"); ok {
		t.Fatal("origin with space parsed")
	}
	if _, ok := parseOriginURL("https://:80"); ok {
		t.Fatal("origin with empty host parsed")
	}

	if got := serializeOriginURL(&url.URL{Scheme: "https", Host: "bad host"}, nil); got != "" {
		t.Fatalf("unexpected serialized origin: %q", got)
	}
	if got := serializeOriginURL(&url.URL{Scheme: "https", Host: "example.com:bad"}, nil); got != "" {
		t.Fatalf("unexpected serialized origin: %q", got)
	}
	if got := serializeOriginURL(&url.URL{Scheme: "https", Host: "example.com:65536"}, nil); got != "" {
		t.Fatalf("unexpected serialized origin: %q", got)
	}

	if host, ok := serializeOriginHost("", nil); ok || host != "" {
		t.Fatalf("unexpected serialized host: %q, %v", host, ok)
	}
	if host, ok := serializeOriginHost("fe80::1%en0", nil); ok || host != "" {
		t.Fatalf("unexpected serialized host: %q, %v", host, ok)
	}
	if host, ok := serializeOriginHost("example.com", func(string) (string, bool) {
		return "", false
	}); ok || host != "" {
		t.Fatalf("unexpected serialized host: %q, %v", host, ok)
	}

	if host, ok := serializeSubdomainPatternHost("", nil); ok || host != "" {
		t.Fatalf("unexpected serialized pattern host: %q, %v", host, ok)
	}
	if host, ok := serializeSubdomainPatternHost("example.com", nil); ok || host != "" {
		t.Fatalf("unexpected serialized pattern host: %q, %v", host, ok)
	}
	if host, ok := serializeSubdomainPatternHost("*.bad:host", nil); ok || host != "" {
		t.Fatalf("unexpected serialized pattern host: %q, %v", host, ok)
	}

	if port, ok := normalizeURLPort("https", "bad"); ok || port != "" {
		t.Fatalf("unexpected normalized port: %q, %v", port, ok)
	}
	if isDefaultOriginPort("ftp", 21) {
		t.Fatal("ftp port treated as default origin port")
	}
	if port, ok := parsePort(""); ok || port != 0 {
		t.Fatalf("unexpected parsed port: %d, %v", port, ok)
	}
	if port, ok := parsePort("bad"); ok || port != 0 {
		t.Fatalf("unexpected parsed port: %d, %v", port, ok)
	}
	if validToken("") {
		t.Fatal("empty token accepted")
	}
}

func TestCORSAddVaryHeaderEdgeCases(t *testing.T) {
	h := http.Header{}
	h.Add("Vary", "Accept-Encoding, , Origin")
	addVaryHeader(h, " , Access-Control-Request-Method")
	for _, field := range []string{"Accept-Encoding", "Origin", "Access-Control-Request-Method"} {
		if !varyContains(h, field) {
			t.Fatalf("Vary does not contain %s", field)
		}
	}

	h = http.Header{}
	h.Set("Vary", "*")
	addVaryHeader(h, "Origin")
	if got := h.Get("Vary"); got != "*" {
		t.Fatalf("unexpected Vary: %q", got)
	}
}

func assertPanics(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("function did not panic")
		}
	}()
	f()
}

func varyContains(h http.Header, value string) bool {
	for _, vary := range h.Values("Vary") {
		for part := range strings.SplitSeq(vary, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return true
			}
		}
	}
	return false
}
