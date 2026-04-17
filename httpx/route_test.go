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
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRoutePattern tests Pattern method
func TestRoutePattern(t *testing.T) {
	tests := []struct {
		name     string
		route    Route
		expected string
	}{
		{
			name: "full route",
			route: Route{
				Method:  http.MethodGet,
				Host:    "example.com",
				Path:    "/api",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
			expected: "GET example.com/api",
		},
		{
			name: "method and path only",
			route: Route{
				Method:  http.MethodPost,
				Path:    "/login",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
			expected: "POST /login",
		},
		{
			name: "host and path only",
			route: Route{
				Host:    "api.example.com",
				Path:    "/v1",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
			expected: "api.example.com/v1",
		},
		{
			name: "path only",
			route: Route{
				Path:    "/",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
			expected: "/",
		},
		{
			name: "empty route",
			route: Route{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.route.Pattern()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestRouteWriteTo tests WriteTo method
func TestRouteWriteTo(t *testing.T) {
	tests := []struct {
		name     string
		route    Route
		expected string
	}{
		{
			name: "full route",
			route: Route{
				Method:  http.MethodGet,
				Host:    "example.com",
				Path:    "/api",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
			expected: "GET example.com/api",
		},
		{
			name: "method only",
			route: Route{
				Method:  http.MethodDelete,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
			expected: "DELETE ",
		},
		{
			name: "empty route",
			route: Route{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			n, err := tt.route.WriteTo(&buf)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			result := buf.String()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}

			if n != int64(len(result)) {
				t.Errorf("expected %d bytes, got %d", len(result), n)
			}
		})
	}
}

// TestRouteWriteToWithError tests WriteTo with failing writer
func TestRouteWriteToWithError(t *testing.T) {
	route := Route{
		Method:  http.MethodGet,
		Host:    "example.com",
		Path:    "/test",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}

	// Writer that fails immediately
	fw := &failingWriter{}
	n, err := route.WriteTo(fw)
	if err == nil {
		t.Error("expected error")
	}
	if n != 0 {
		t.Errorf("expected 0 bytes, got %d", n)
	}
}

// TestRouteHandler tests Handler field
func TestRouteHandler(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	route := Route{
		Method:  http.MethodGet,
		Path:    "/test",
		Handler: handler,
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	route.Handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

// TestTryWriteString tests helper function
func TestTryWriteString(t *testing.T) {
	var buf bytes.Buffer
	var n int64

	// Normal write
	err := tryWriteString(&buf, "test", &n, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 bytes, got %d", n)
	}
	if buf.String() != "test" {
		t.Errorf("expected 'test', got '%s'", buf.String())
	}

	// With existing error
	buf.Reset()
	n = 0
	err = tryWriteString(&buf, "test", &n, io.EOF)
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

// TestRouteWriteToLargeData tests with large data
func TestRouteWriteToLargeData(t *testing.T) {
	longPath := "/" + strings.Repeat("a", 1000)
	route := Route{
		Method:  http.MethodGet,
		Host:    "example.com",
		Path:    longPath,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}

	var buf bytes.Buffer
	n, err := route.WriteTo(&buf)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := "GET example.com" + longPath
	result := buf.String()
	if result != expected {
		t.Errorf("expected length %d, got %d", len(expected), len(result))
	}
	if n != int64(len(expected)) {
		t.Errorf("expected %d bytes, got %d", len(expected), n)
	}
}

// failingWriter always fails on Write
type failingWriter struct{}

func (w *failingWriter) Write(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
