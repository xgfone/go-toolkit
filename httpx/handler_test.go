// Copyright 2025 xgfone
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
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	Handler201.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Errorf("expect status code %d, but got %d", 201, rec.Code)
	}

	rec = httptest.NewRecorder()
	Handler404.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Errorf("expect status code %d, but got %d", 404, rec.Code)
	}
}

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := JSON(rec, 200, nil); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s != "" {
		t.Errorf("expect response body '%s', but got '%s'", "", s)
	}

	rec = httptest.NewRecorder()
	expectbody := `{"a":"b"}`
	if err := JSON(rec, 200, map[string]string{"a": "b"}); err != nil {
		t.Fatal(err)
	} else if body := strings.TrimSpace(rec.Body.String()); body != expectbody {
		t.Errorf("expect response body '%s', but got '%s'", expectbody, body)
	}
}

func TestXML(t *testing.T) {
	var req struct {
		XMLName xml.Name `xml:"outer"`
		A       string   `xml:"a"`
	}
	req.A = "b"

	rec := httptest.NewRecorder()
	if err := XML(rec, 400, nil); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s != "" {
		t.Errorf("expect response body '%s', but got '%s'", "", s)
	}

	rec = httptest.NewRecorder()
	if err := XML(rec, 200, req); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s == "" {
		t.Error("unexpected empty response body")
	}

	expectbody := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<outer><a>b</a></outer>`
	if body := rec.Body.String(); body != expectbody {
		t.Errorf("expect response body '%s', but got '%s'", expectbody, body)
	}
}

func TestContextHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	ContextHandler(func(c *Context) error {
		if c != nil {
			t.Error("expect a nil, but got a Context")
		}
		return nil
	}).ServeHTTP(rec, req)

	_c := new(Context)
	req = req.WithContext(SetContext(req.Context(), _c))
	ContextHandler(func(c *Context) error {
		if c != _c {
			t.Error("context is inconsistent")
		}
		return nil
	}).ServeHTTP(rec, req)

}

func TestContextHandlerAndOr(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		if h := ContextHandlerAnd(); h != nil {
			t.Error("ContextHandlerAnd with no handlers should return nil")
		}
		if h := ContextHandlerOr(); h != nil {
			t.Error("ContextHandlerOr with no handlers should return nil")
		}
	})

	t.Run("one", func(t *testing.T) {
		h := ContextHandler(func(c *Context) error { return nil })
		if got := ContextHandlerAnd(h); got == nil || got(nil) != nil {
			t.Error("ContextHandlerAnd with one handler should return and call it")
		}
		if got := ContextHandlerOr(h); got == nil || got(nil) != nil {
			t.Error("ContextHandlerOr with one handler should return and call it")
		}
	})

	t.Run("and", func(t *testing.T) {
		errTarget := fmt.Errorf("stop")
		var calls []int

		// all succeed
		h := ContextHandlerAnd(
			func(c *Context) error { calls = append(calls, 1); return nil },
			func(c *Context) error { calls = append(calls, 2); return nil },
		)
		if err := h(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if len(calls) != 2 || calls[0] != 1 || calls[1] != 2 {
			t.Errorf("expected calls [1 2], got %v", calls)
		}

		// short-circuit on first error
		calls = nil
		h = ContextHandlerAnd(
			func(c *Context) error { calls = append(calls, 1); return nil },
			func(c *Context) error { calls = append(calls, 2); return errTarget },
			func(c *Context) error { calls = append(calls, 3); return nil },
		)
		if err := h(nil); err != errTarget {
			t.Errorf("expected target error, got %v", err)
		}
		if len(calls) != 2 || calls[0] != 1 || calls[1] != 2 {
			t.Errorf("expected calls [1 2], got %v", calls)
		}
	})

	t.Run("or", func(t *testing.T) {
		err1 := fmt.Errorf("err1")
		var calls []int

		// short-circuit on first success
		h := ContextHandlerOr(
			func(c *Context) error { calls = append(calls, 1); return nil },
			func(c *Context) error { calls = append(calls, 2); return fmt.Errorf("unreachable") },
		)
		if err := h(nil); err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if len(calls) != 1 || calls[0] != 1 {
			t.Errorf("expected calls [1], got %v", calls)
		}

		// all fail
		calls = nil
		h = ContextHandlerOr(
			func(c *Context) error { calls = append(calls, 1); return err1 },
			func(c *Context) error { calls = append(calls, 2); return fmt.Errorf("err2") },
		)
		if err := h(nil); err == nil || err.Error() != "err2" {
			t.Errorf("expected last error 'err2', got %v", err)
		}
		if len(calls) != 2 || calls[0] != 1 || calls[1] != 2 {
			t.Errorf("expected calls [1 2], got %v", calls)
		}
	})
}

func TestContextHandler_HTTPHandler(t *testing.T) {
	// Test case 1: No context in request
	t.Run("NoContext", func(t *testing.T) {
		handler := ContextHandler(func(c *Context) error { return nil })

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called when there's no context")
		})

		wrapped := handler.HTTPHandler(next)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		wrapped.ServeHTTP(rec, req)

		if rec.Code != 500 {
			t.Errorf("expected status code 500, got %d", rec.Code)
		}
		if body := rec.Body.String(); body != "missing httpx.Context" {
			t.Errorf("expected body 'missing httpx.Context', got '%s'", body)
		}
	})

	// Test case 2: ContextHandler returns error
	t.Run("HandlerError", func(t *testing.T) {
		const expectedErr = "test error"
		handler := ContextHandler(func(c *Context) error { return fmt.Errorf(expectedErr) })

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("next handler should not be called when ContextHandler returns error")
		})

		wrapped := handler.HTTPHandler(next)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)

		ctx := AcquireContext()
		defer ReleaseContext(ctx)
		req = req.WithContext(SetContext(req.Context(), ctx))

		wrapped.ServeHTTP(rec, req)

		if ctx.Error == nil {
			t.Error("expected error to be appended to context")
		} else if ctx.Error.Error() != expectedErr {
			t.Errorf("expected error '%s', got '%s'", expectedErr, ctx.Error.Error())
		}
	})

	// Test case 3: ContextHandler succeeds, next handler is called
	t.Run("Success", func(t *testing.T) {
		handler := ContextHandler(func(c *Context) error { return nil })

		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(200)
		})

		wrapped := handler.HTTPHandler(next)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)

		ctx := AcquireContext()
		defer ReleaseContext(ctx)
		req = req.WithContext(SetContext(req.Context(), ctx))

		wrapped.ServeHTTP(rec, req)

		if !called {
			t.Error("next handler should be called when ContextHandler succeeds")
		}
		if rec.Code != 200 {
			t.Errorf("expected status code 200, got %d", rec.Code)
		}
	})
}
