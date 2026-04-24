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
	"strings"
	"testing"
)

func newResponseWriter(c *Context, w http.ResponseWriter) ResponseWriter {
	c.w = w
	return newContextResponseWriter(c)
}

func TestContextResponseWriter(t *testing.T) {
	t.Run("Unwrap", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx := &Context{}
		rw := newResponseWriter(ctx, rec)

		if w, ok := rw.(interface{ Unwrap() http.ResponseWriter }); ok {
			if w.Unwrap() != rec {
				t.Error("Unwrap should return original ResponseWriter")
			}
		} else {
			t.Error("newContextResponseWriter should return _ContextResponseWriter")
		}
	})

	t.Run("Header", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx := &Context{}
		rw := newResponseWriter(ctx, rec)

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

	t.Run("Write", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx := &Context{}
		rw := newResponseWriter(ctx, rec)

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
		if ctx.Code != 200 {
			t.Errorf("ctx.Code should be 200, got %d", ctx.Code)
		}
	})

	t.Run("WriteHeader", func(t *testing.T) {
		t.Run("ValidStatusCode", func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx := &Context{}
			rw := newResponseWriter(ctx, rec)

			rw.WriteHeader(201)
			if rec.Code != 201 {
				t.Errorf("status code should be 201, got %d", rec.Code)
			}
			if ctx.Code != 201 {
				t.Errorf("ctx.Code should be 201, got %d", ctx.Code)
			}
		})

		t.Run("IgnoreSecondWriteHeader", func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx := &Context{}
			rw := newResponseWriter(ctx, rec)

			rw.WriteHeader(200)
			rw.WriteHeader(404)
			if rec.Code != 200 {
				t.Errorf("second WriteHeader should be ignored, got %d", rec.Code)
			}
			if ctx.Code != 200 {
				t.Errorf("ctx.Code should remain 200, got %d", ctx.Code)
			}
		})

		t.Run("PanicOnInvalidStatusCode", func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("should panic with invalid status code")
				}
			}()

			rec := httptest.NewRecorder()
			ctx := &Context{}
			rw := newResponseWriter(ctx, rec)
			rw.WriteHeader(99)
		})
	})

	t.Run("StatusCode", func(t *testing.T) {
		t.Run("DefaultStatusCode", func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx := &Context{}
			rw := newResponseWriter(ctx, rec)

			if code := rw.StatusCode(); code != 0 {
				t.Errorf("default status code should be 0, got %d", code)
			}
		})

		t.Run("AfterWriteHeader", func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx := &Context{}
			rw := newResponseWriter(ctx, rec)

			rw.WriteHeader(404)
			if code := rw.StatusCode(); code != 404 {
				t.Errorf("status code should be 404, got %d", code)
			}
		})

		t.Run("AfterWrite", func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx := &Context{}
			rw := newResponseWriter(ctx, rec)

			rw.Write([]byte("test"))
			if code := rw.StatusCode(); code != 200 {
				t.Errorf("status code should be 200 after Write, got %d", code)
			}
		})
	})

	t.Run("JSON", func(t *testing.T) {
		ctx := &Context{}
		rec := httptest.NewRecorder()
		rw := newResponseWriter(ctx, rec)
		ctx.ResponseWriter = rw

		if v, ok := rw.(interface{ JSON(int, any) }); !ok {
			t.Errorf("rw should implement JSON method")
		} else {
			v.JSON(201, map[string]string{"a": "b"})
			if rec.Code != 201 {
				t.Errorf("status code should be 200, got %d", rec.Code)
			} else if s := strings.TrimSpace(rec.Body.String()); s != `{"a":"b"}` {
				t.Errorf("response body should be '%s', got '%s'", `{"a":"b"}`, s)
			}
		}
	})

	t.Run("GetContext", func(t *testing.T) {
		ctx := &Context{}
		rec := httptest.NewRecorder()
		rw := newResponseWriter(ctx, rec)
		if v, ok := rw.(interface{ GetContext() *Context }); !ok {
			t.Errorf("rw should implement GetContext method")
		} else if ctx2 := v.GetContext(); ctx2 != ctx {
			t.Errorf("GetContext should return the original context")
		}
	})
}
