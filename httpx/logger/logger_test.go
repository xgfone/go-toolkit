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

package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xgfone/go-toolkit/errorx"
	"github.com/xgfone/go-toolkit/httpx"
	"github.com/xgfone/go-toolkit/httpx/middleware"
)

func TestLoggerLogsRequest(t *testing.T) {
	logs := captureLogs(t)

	preHandlerCalled := false
	postHandlerCalled := false
	config := NewDefaultConfig()
	getDefaultAttrs := config.GetLogAttrs

	attrsCalled := 0
	config.GetLogAttrs = func(w http.ResponseWriter, r *http.Request) []slog.Attr {
		attrsCalled++
		if !preHandlerCalled || w.Header().Get("X-Result") != "created" {
			t.Fatal("GetLogAttrs called before the handler completed")
		}
		if postHandlerCalled {
			t.Fatal("GetLogAttrs called after PostHandle")
		}

		c := httpx.GetContext(r.Context())
		if c == nil || c.StatusCode() != http.StatusCreated {
			t.Fatal("GetLogAttrs cannot read the final HTTP context")
		}

		return append(getDefaultAttrs(w, r),
			slog.String("reqbody", "request-body"),
			slog.Any("resbody", c.ResponseBody),
			slog.String("query", r.URL.RawQuery),
			slog.String("result", w.Header().Get("X-Result")),
		)
	}

	config.PreHandle = func(http.ResponseWriter, *http.Request) { preHandlerCalled = true }
	config.PostHandle = func(w http.ResponseWriter, r *http.Request) { postHandlerCalled = true }

	req := httptest.NewRequest(http.MethodPost, "http://example.com/items?q=1", nil)
	req.Header.Set("X-Request-Id", "rid-1")

	raw := errors.New("secret")
	mdHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := httpx.GetContext(r.Context())
		c.ResponseBody = "response-body"
		c.AppendError(errorx.Sensitive(raw, "safe"))
		w.Header().Set("X-Result", "created")
		w.WriteHeader(http.StatusCreated)
	})
	handler := middleware.Context(config.Middleware(10).HTTPHandler(mdHandler))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !preHandlerCalled {
		t.Fatal("PreHandler was not called")
	}
	if !postHandlerCalled {
		t.Fatal("PostHandler was not called")
	}
	if attrsCalled != 1 {
		t.Fatalf("GetLogAttrs called %d times, want 1", attrsCalled)
	}
	if len(logs.records) != 1 {
		t.Fatalf("got %d log records, want 1", len(logs.records))
	}

	attrs := logs.records[0]
	for name, want := range map[string]any{
		"reqid":   "rid-1",
		"method":  http.MethodPost,
		"host":    "example.com",
		"path":    "/items",
		"query":   "q=1",
		"code":    int64(http.StatusCreated),
		"reqbody": "request-body",
		"resbody": "response-body",
		"result":  "created",
		"err":     "safe",

		"sensitive_error": "secret",
	} {
		if got := attrs[name]; got != want {
			t.Fatalf("attr %q = %#v, want %#v", name, got, want)
		}
	}
}

func TestLoggerDefaultsOmitHTTPDetails(t *testing.T) {
	logs := captureLogs(t)
	config := NewDefaultConfig()
	req := httptest.NewRequest(http.MethodPost, "/items?token=query-secret", nil)
	req.Header.Set("Authorization", "Bearer request-secret")
	req.Header.Set("Cookie", "session=cookie-secret")
	handler := middleware.Context(config.Middleware(1).
		HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := httpx.GetContext(r.Context())
			c.ResponseBody = "response-secret"
			c.AppendError(errorx.Sensitive(errors.New("underlying-secret"), "safe"))
			w.Header().Set("Set-Cookie", "session=response-cookie-secret")
			w.WriteHeader(http.StatusCreated)
		})))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if len(logs.records) != 1 {
		t.Fatalf("got %d log records, want 1", len(logs.records))
	}

	attrs := logs.records[0]
	if attrs["code"] != int64(http.StatusCreated) || attrs["err"] != "safe" {
		t.Fatalf("missing status or safe error: %v", attrs)
	}
	if attrs["sensitive_error"] != "underlying-secret" {
		t.Fatalf("missing original sensitive error: %v", attrs)
	}

	for _, key := range []string{
		"query",
		"reqheader",
		"resheader",
		"reqbody",
		"resbody",
	} {
		if _, ok := attrs[key]; ok {
			t.Errorf("default logger included %q", key)
		}
	}
}

func TestLoggerReplacesDefaultLogAttrs(t *testing.T) {
	for _, useGetter := range []bool{true, false} {
		t.Run(fmt.Sprintf("getter=%t", useGetter), func(t *testing.T) {
			logs := captureLogs(t)
			config := NewDefaultConfig()
			config.GetLogAttrs = nil
			if useGetter {
				config.GetLogAttrs = func(http.ResponseWriter, *http.Request) []slog.Attr {
					return []slog.Attr{slog.String("custom", "value")}
				}
			}

			handler := middleware.Context(config.Middleware(1).
				HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					httpx.GetContext(r.Context()).AppendError(errorx.Sensitive(errors.New("secret"), "safe"))
					w.WriteHeader(http.StatusInternalServerError)
				})))
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/items", nil))

			if len(logs.records) != 1 {
				t.Fatalf("got %d log records, want 1", len(logs.records))
			}
			attrs := logs.records[0]
			if attrs["method"] != http.MethodGet || attrs["path"] != "/items" {
				t.Fatalf("missing built-in request attributes: %v", attrs)
			}
			if useGetter && attrs["custom"] != "value" {
				t.Fatalf("missing custom attribute: %v", attrs)
			}
			for _, key := range []string{"code", "err", "sensitive_error"} {
				if _, ok := attrs[key]; ok {
					t.Errorf("replaced logger included default %q", key)
				}
			}
		})
	}
}

func TestLoggerSkipsWhenDisabled(t *testing.T) {
	logs := captureLogs(t)

	nextCalled := false
	handler := Config{
		Enabled: func(*http.Request) bool { return false },
		GetLogAttrs: func(http.ResponseWriter, *http.Request) []slog.Attr {
			t.Error("GetLogAttrs called for a disabled request")
			return nil
		},
	}.Middleware(3).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !nextCalled {
		t.Fatal("next handler was not called")
	}
	if len(logs.records) != 0 {
		t.Fatalf("got %d log records, want 0", len(logs.records))
	}
}

func TestLoggerEdgeCases(t *testing.T) {
	logger := Config{}.Middleware(3).(*logger)
	if got := logger.Priority(); got != 3 {
		t.Fatalf("Priority() = %d, want 3", got)
	}

	assertPanic(t, func() { logger.HTTPHandler(nil) })

	rec := httptest.NewRecorder()
	logger.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError || rec.Body.String() != "Logger: NO NEXT HANDLER" {
		t.Fatalf("ServeHTTP without next = %d, %q; want %d, %q",
			rec.Code, rec.Body.String(), http.StatusInternalServerError, "Logger: NO NEXT HANDLER")
	}
}

func TestGetResponseUnwrapsResponseWriter(t *testing.T) {
	rw := &statusWriter{code: http.StatusAccepted}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	errBody := errors.New("response error")

	ctx := httpx.AcquireContext()
	defer httpx.ReleaseContext(ctx)
	ctx.ResponseCode = http.StatusPartialContent
	ctx.ResponseBody = "small response"
	ctx.Error = errBody
	reqWithContext := req.WithContext(httpx.SetContext(req.Context(), ctx))

	status, err := getResponse(rw, reqWithContext)
	if status != http.StatusPartialContent || err != errBody {
		t.Fatalf("getResponse(context) = %d, %v; want %d, %v",
			status, err, http.StatusPartialContent, errBody)
	}

	status, err = getResponse(unwrapWriter{unwrapWriter{ResponseWriter: rw}}, req)
	if status != http.StatusAccepted || err != nil {
		t.Fatalf("getResponse() = %d, %v; want %d, nil",
			status, err, http.StatusAccepted)
	}

	status, _ = getResponse(&cycleWriter{}, req)
	if status != 0 {
		t.Fatalf("getResponse(cycleWriter) status = %d, want 0", status)
	}

	status, _ = getResponse(unwrapWriter{}, req)
	if status != 0 {
		t.Fatalf("getResponse(nil unwrap) status = %d, want 0", status)
	}

	status, _ = getResponse(httptest.NewRecorder(), req)
	if status != 0 {
		t.Fatalf("getResponse(default writer) status = %d, want 0", status)
	}
}

func TestGetLogAttrs(t *testing.T) {
	rw := &statusWriter{code: http.StatusAccepted}
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	attrs := getLogAttrs(unwrapWriter{ResponseWriter: rw}, req)
	if len(attrs) != 1 || !attrs[0].Equal(slog.Int("code", http.StatusAccepted)) {
		t.Fatalf("getLogAttrs without context = %v; want code=%d", attrs, http.StatusAccepted)
	}

	ctx := httpx.AcquireContext()
	defer httpx.ReleaseContext(ctx)
	ctx.ResponseCode = http.StatusInternalServerError
	ctx.Error = errors.New("response error")
	attrs = getLogAttrs(rw, req.WithContext(httpx.SetContext(req.Context(), ctx)))
	if len(attrs) != 2 || !attrs[0].Equal(slog.Int("code", http.StatusInternalServerError)) ||
		!attrs[1].Equal(slog.String("err", "response error")) {
		t.Fatalf("getLogAttrs with ordinary error = %v; want code and err only", attrs)
	}
}

func TestGetSensitiveErrorMessage(t *testing.T) {
	var nilSensitive *errorx.SensitiveError
	for _, tt := range []struct {
		name string
		err  error
		msg  string
		ok   bool
	}{
		{name: "nil"},
		{name: "ordinary", err: errors.New("ordinary")},
		{name: "sensitive", err: errorx.Sensitive(errors.New("secret"), "safe"), msg: "secret", ok: true},
		{name: "nil sensitive", err: nilSensitive, msg: "<nil>", ok: true},
		{name: "wrapped", err: fmt.Errorf("service: %w", errorx.Sensitive(errors.New("wrapped-secret"), "safe")), msg: "wrapped-secret", ok: true},
		{name: "joined", err: errors.Join(errors.New("first"), errorx.Sensitive(errors.New("joined-secret"), "safe")), msg: "joined-secret", ok: true},
		{name: "custom", err: customSensitiveError{raw: errors.New("custom-secret")}, msg: "custom-secret", ok: true},
		{name: "wrapped custom", err: fmt.Errorf("service: %w", customSensitiveError{raw: errors.New("custom-secret")}), msg: "custom-secret", ok: true},
		{name: "nil custom cause", err: customSensitiveError{}, msg: "<nil>", ok: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			msg, ok := getSensitiveErrorMessage(tt.err)
			if msg != tt.msg || ok != tt.ok {
				t.Fatalf("getSensitiveErrorMessage() = %q, %t; want %q, %t", msg, ok, tt.msg, tt.ok)
			}
		})
	}
}

type captureHandler struct {
	records []map[string]any
}

func captureLogs(t *testing.T) *captureHandler {
	t.Helper()

	handler := new(captureHandler)
	orig := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return handler
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h *captureHandler) WithGroup(string) slog.Handler            { return h }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := make(map[string]any, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	h.records = append(h.records, attrs)
	return nil
}

type statusWriter struct {
	header http.Header
	code   int
}

func (w *statusWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return len(p), nil
}

func (w *statusWriter) WriteHeader(code int) { w.code = code }
func (w *statusWriter) StatusCode() int      { return w.code }

type unwrapWriter struct {
	http.ResponseWriter
}

func (w unwrapWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type cycleWriter struct {
	header http.Header
}

func (w *cycleWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *cycleWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *cycleWriter) WriteHeader(int)             {}
func (w *cycleWriter) Unwrap() http.ResponseWriter { return w }

type customSensitiveError struct {
	raw error
}

func (e customSensitiveError) Error() string         { return "safe" }
func (e customSensitiveError) SensitiveError() error { return e.raw }

func assertPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatal("got no panic")
		}
	}()
	f()
}
