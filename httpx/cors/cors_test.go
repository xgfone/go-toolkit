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
	handler := Config{
		AllowOrigins: []string{"https://example.com"},
		MaxAge:       &maxAge,
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}
	if varyContains(rec.Header(), "Origin") {
		t.Fatal("Vary contains Origin for wildcard origin")
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

func TestCORSVaryOriginSurvivesNextHandlerSet(t *testing.T) {
	handler := Config{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	for _, field := range []string{"Accept-Encoding", "Origin"} {
		if !varyContains(rec.Header(), field) {
			t.Fatalf("Vary does not contain %s", field)
		}
	}
}

func TestCORSSubdomainPattern(t *testing.T) {
	handler := Config{AllowOrigins: []string{"https://*.example.com"}}.CORS(0).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://api.example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://api.example.com" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec = httptest.NewRecorder()
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

	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
	}
}

func TestCORSMaxAgeZero(t *testing.T) {
	maxAge := 0
	handler := Config{
		AllowOrigins: []string{"https://example.com"},
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
		{AllowOrigins: []string{"https://api.*.example.com"}},
		{AllowOrigins: []string{"https://*.example.com/path"}},
		{AllowOrigins: []string{"https://example.com"}, AllowMethods: []string{"BAD METHOD"}},
		{AllowOrigins: []string{"https://example.com"}, AllowHeaders: []string{"Bad Header"}},
		{AllowOrigins: []string{"https://example.com"}, MaxAge: &negativeMaxAge},
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

func TestCORSPriority(t *testing.T) {
	c := Config{}.CORS(123)
	if got := c.Priority(); got != 123 {
		t.Fatalf("unexpected priority: got %d, want %d", got, 123)
	}
}

func varyContains(h http.Header, value string) bool {
	for _, vary := range h.Values("Vary") {
		for _, part := range strings.Split(vary, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return true
			}
		}
	}
	return false
}
