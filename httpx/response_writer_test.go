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
	"errors"
	"io"
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
		if ctx.ResponseCode != 200 {
			t.Errorf("ctx.Code should be 200, got %d", ctx.ResponseCode)
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
			if ctx.ResponseCode != 201 {
				t.Errorf("ctx.Code should be 201, got %d", ctx.ResponseCode)
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
			if ctx.ResponseCode != 200 {
				t.Errorf("ctx.Code should remain 200, got %d", ctx.ResponseCode)
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

			_, _ = rw.Write([]byte("test"))
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

	t.Run("HTTPContext", func(t *testing.T) {
		ctx := &Context{}
		rec := httptest.NewRecorder()
		rw := newResponseWriter(ctx, rec)
		if v, ok := rw.(interface{ HTTPContext() *Context }); !ok {
			t.Errorf("rw should implement GetContext method")
		} else if ctx2 := v.HTTPContext(); ctx2 != ctx {
			t.Errorf("GetContext should return the original context")
		}
	})
}

func TestContextResponseWriterControl(t *testing.T) {
	t.Run("flush preserves explicit status through an unwrapping writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx := new(Context)
		ctx.Reset(controlUnwrapper{rec}, nil)
		ctx.WriteHeader(http.StatusCreated)

		err := http.NewResponseController(ctx.ResponseWriter).Flush()
		if err != nil {
			t.Fatal(err)
		}
		if !rec.Flushed || rec.Code != http.StatusCreated || ctx.StatusCode() != http.StatusCreated {
			t.Fatalf("flush = %t, status = %d/%d, want 201",
				rec.Flushed, rec.Code, ctx.StatusCode())
		}
	})

	t.Run("unsupported controls do not commit state", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx := new(Context)
		ctx.Reset(struct{ http.ResponseWriter }{rec}, nil)
		controller := http.NewResponseController(ctx.ResponseWriter)
		if err := controller.Flush(); !errors.Is(err, http.ErrNotSupported) {
			t.Fatalf("Flush = %v, want ErrNotSupported", err)
		}

		if _, _, err := controller.Hijack(); !errors.Is(err, http.ErrNotSupported) {
			t.Fatalf("Hijack = %v, want ErrNotSupported", err)
		}
		if ctx.Hijacked() || ctx.StatusCode() != 0 || rec.Flushed {
			t.Fatal("unsupported controls changed response state")
		}

		ctx.NoContent(http.StatusAccepted)
		if ctx.StatusCode() != http.StatusAccepted {
			t.Fatal("unsupported controls prevented a later response")
		}
	})

	t.Run("flush failures preserve recorded status", func(t *testing.T) {
		flushErr := errors.New("flush failed")
		for _, status := range []int{0, http.StatusCreated} {
			rec := httptest.NewRecorder()
			rec.Code = 0
			ctx := new(Context)
			ctx.Reset(controlFlushError{rec, flushErr}, nil)
			if status != 0 {
				ctx.WriteHeader(status)
			}

			err := http.NewResponseController(ctx.ResponseWriter).Flush()
			if !errors.Is(err, flushErr) {
				t.Fatalf("Flush = %v, want %v", err, flushErr)
			}
			if rec.Code != status || ctx.StatusCode() != status {
				t.Fatalf("status after failure = %d/%d, want %d", rec.Code, ctx.StatusCode(), status)
			}
		}
	})

	t.Run("hijacked connections do not commit or flush responses", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rec.Code = 0
		ctx := new(Context)
		ctx.Reset(rec, nil)
		ctx.hijacked = true
		controller := http.NewResponseController(ctx.ResponseWriter)
		if err := controller.Flush(); !errors.Is(err, http.ErrHijacked) {
			t.Fatalf("Flush = %v, want ErrHijacked", err)
		}
		if _, _, err := controller.Hijack(); !errors.Is(err, http.ErrHijacked) {
			t.Fatalf("Hijack = %v, want ErrHijacked", err)
		}

		ctx.WriteHeader(http.StatusOK)
		if ctx.StatusCode() != 0 || rec.Code != 0 || rec.Body.Len() != 0 || rec.Flushed {
			t.Fatal("response operations touched a hijacked connection")
		}
	})

	t.Run("hijack state delegates and resets", func(t *testing.T) {
		ctx := new(Context)
		ctx.Reset(httptest.NewRecorder(), nil)
		ctx.hijacked = true
		nested := new(Context)
		nested.Reset(ctx.ResponseWriter, nil)
		if !nested.Hijacked() {
			t.Fatal("nested context lost the writer's hijack state")
		}

		ctx.Reset(httptest.NewRecorder(), nil)
		if ctx.Hijacked() {
			t.Fatal("reset retained the previous request's hijack state")
		}
	})
}

func TestContextResponseWriterRealHTTP(t *testing.T) {
	t.Run("flush commits implicit status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := newContext(w, r)
			ctx.ResponseWriter.(http.Flusher).Flush()
			if ctx.StatusCode() != http.StatusOK {
				t.Errorf("status after Flush = %d, want 200", ctx.StatusCode())
			}
		}))
		defer server.Close()
		response, err := server.Client().Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		defer response.Body.Close() //nolint:errcheck
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || len(body) != 0 {
			t.Fatalf("response = %d %q, want empty 200", response.StatusCode, body)
		}
	})

	t.Run("hijack tracks takeover without inventing status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := newContext(w, r)
			conn, rw, err := http.NewResponseController(ctx.ResponseWriter).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close() //nolint:errcheck

			if !ctx.Hijacked() || ctx.StatusCode() != 0 {
				t.Errorf("takeover = %t, status = %d, want true, 0", ctx.Hijacked(), ctx.StatusCode())
			}

			const upgrade = "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: test\r\n\r\n"
			if _, err := io.WriteString(rw, upgrade); err != nil {
				t.Error(err)
				return
			}
			if err := rw.Flush(); err != nil {
				t.Error(err)
			}
		}))
		defer server.Close()

		response, err := server.Client().Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close() //nolint:errcheck

		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusSwitchingProtocols || len(body) != 0 {
			t.Fatalf("response = %d %q, want empty 101", response.StatusCode, body)
		}
	})
}

type controlUnwrapper struct{ http.ResponseWriter }

func (w controlUnwrapper) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type controlFlushError struct {
	http.ResponseWriter
	err error
}

func (w controlFlushError) FlushError() error {
	return w.err
}
