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

func TestResponseWriter(t *testing.T) {
	t.Run("NewResponseWriter", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := NewResponseWriter(rec)

		if _, ok := rw.(*_ResponseWriter); !ok {
			t.Error("should return *_ResponseWriter for regular http.ResponseWriter")
		}

		// Should return same object if already ResponseWriter
		if NewResponseWriter(rw) != rw {
			t.Error("should return same ResponseWriter")
		}

		// Should return custom ResponseWriter as-is
		customRW := &customResponseWriter{recorder: httptest.NewRecorder()}
		if NewResponseWriter(customRW) != customRW {
			t.Error("should return custom ResponseWriter as-is")
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := NewResponseWriter(rec).(*_ResponseWriter)

		if rw.Unwrap() != rec {
			t.Error("Unwrap should return original ResponseWriter")
		}
	})

	t.Run("StatusCode", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := NewResponseWriter(rec).(*_ResponseWriter)

		if code := rw.StatusCode(); code != 200 {
			t.Errorf("default status code should be 200, got %d", code)
		}

		rw.WriteHeader(404)
		if code := rw.StatusCode(); code != 404 {
			t.Errorf("status code should be 404, got %d", code)
		}
	})

	t.Run("WroteHeader", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := NewResponseWriter(rec).(*_ResponseWriter)

		if rw.WroteHeader() {
			t.Error("WroteHeader should be false initially")
		}

		rw.WriteHeader(200)
		if !rw.WroteHeader() {
			t.Error("WroteHeader should be true after WriteHeader")
		}
	})

	t.Run("WriteHeader", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := NewResponseWriter(rec).(*_ResponseWriter)

		rw.WriteHeader(201)
		if rec.Code != 201 {
			t.Errorf("status code should be 201, got %d", rec.Code)
		}

		// Should ignore second WriteHeader
		rec2 := httptest.NewRecorder()
		rw2 := NewResponseWriter(rec2).(*_ResponseWriter)
		rw2.WriteHeader(200)
		rw2.WriteHeader(404)
		if rec2.Code != 200 {
			t.Errorf("second WriteHeader should be ignored, got %d", rec2.Code)
		}
	})

	t.Run("WriteHeaderPanic", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("should panic with invalid status code")
			}
		}()

		rec := httptest.NewRecorder()
		rw := NewResponseWriter(rec).(*_ResponseWriter)
		rw.WriteHeader(99)
	})

	t.Run("Write", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := NewResponseWriter(rec)

		data := []byte("test")
		n, err := rw.Write(data)
		if err != nil {
			t.Errorf("Write error: %v", err)
		}
		if n != len(data) {
			t.Errorf("wrote %d bytes, expected %d", n, len(data))
		}
		if rec.Body.String() != string(data) {
			t.Errorf("body %q, expected %q", rec.Body.String(), string(data))
		}
		if rec.Code != 200 {
			t.Errorf("Write should set status 200, got %d", rec.Code)
		}
	})

	t.Run("Header", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := NewResponseWriter(rec)

		headers := rw.Header()
		headers.Set("Content-Type", "application/json")
		headers.Set("X-Custom", "value")

		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type %q, expected %q", ct, "application/json")
		}
		if custom := rec.Header().Get("X-Custom"); custom != "value" {
			t.Errorf("X-Custom %q, expected %q", custom, "value")
		}
	})
}

type customResponseWriter struct {
	recorder *httptest.ResponseRecorder
}

func (c *customResponseWriter) Header() http.Header {
	return c.recorder.Header()
}

func (c *customResponseWriter) Write(data []byte) (int, error) {
	return c.recorder.Write(data)
}

func (c *customResponseWriter) WriteHeader(statusCode int) {
	c.recorder.WriteHeader(statusCode)
}

func (c *customResponseWriter) WroteHeader() bool {
	return c.recorder.Code > 0
}

func (c *customResponseWriter) StatusCode() int {
	if c.recorder.Code == 0 {
		return 200
	}
	return c.recorder.Code
}
