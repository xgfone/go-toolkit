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
	config.GetRequestBody = func(*http.Request) any { return "request-body" }
	config.PreHandle = func(http.ResponseWriter, *http.Request) { preHandlerCalled = true }
	config.PostHandle = func(w http.ResponseWriter, r *http.Request) { postHandlerCalled = true }

	raw := errors.New("secret")
	handler := middleware.Context(config.Logger(10).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := httpx.GetContext(r.Context())
		c.ResponseBody = "response-body"
		c.AppendError(errorx.Sensitive(raw, "safe"))
		w.WriteHeader(http.StatusCreated)
	})))

	req := httptest.NewRequest(http.MethodPost, "http://example.com/items?q=1", nil)
	req.Header.Set("X-Request-Id", "rid-1")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !preHandlerCalled {
		t.Fatal("PreHandler was not called")
	}
	if !postHandlerCalled {
		t.Fatal("PostHandler was not called")
	}
	if len(logs.records) != 1 {
		t.Fatalf("got %d log records, want 1", len(logs.records))
	}

	attrs := logs.records[0]
	for name, want := range map[string]any{
		"reqid":           "rid-1",
		"method":          http.MethodPost,
		"host":            "example.com",
		"path":            "/items",
		"query":           "q=1",
		"code":            int64(http.StatusCreated),
		"reqbody":         "request-body",
		"resbody":         "response-body",
		"err":             "safe",
		"sensitive_error": "secret",
	} {
		if got := attrs[name]; got != want {
			t.Fatalf("attr %q = %#v, want %#v", name, got, want)
		}
	}
}

func TestLoggerSkipsWhenDisabled(t *testing.T) {
	logs := captureLogs(t)

	nextCalled := false
	handler := Config{
		Enabled: func(*http.Request) bool { return false },
	}.Logger(3).HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	logger := Config{}.Logger(3).(*logger)
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

	status, response, err := getResponse(rw, reqWithContext)
	if status != http.StatusPartialContent || response != "small response" || err != errBody {
		t.Fatalf("getResponse(context) = %d, %#v, %v; want %d, %q, %v",
			status, response, err, http.StatusPartialContent, "small response", errBody)
	}

	ctx.BytesWritten = 2049
	status, response, err = getResponse(rw, reqWithContext)
	if status != http.StatusPartialContent || response != "[BODY TOO LONG: 2049]" || err != errBody {
		t.Fatalf("getResponse(long context) = %d, %#v, %v; want %d, %q, %v",
			status, response, err, http.StatusPartialContent, "[BODY TOO LONG: 2049]", errBody)
	}

	status, response, err = getResponse(unwrapWriter{ResponseWriter: unwrapWriter{ResponseWriter: rw}}, req)
	if status != http.StatusAccepted || response != nil || err != nil {
		t.Fatalf("getResponse() = %d, %#v, %v; want %d, nil, nil", status, response, err, http.StatusAccepted)
	}

	status, _, _ = getResponse(&cycleWriter{}, req)
	if status != 0 {
		t.Fatalf("getResponse(cycleWriter) status = %d, want 0", status)
	}

	status, _, _ = getResponse(unwrapWriter{}, req)
	if status != 0 {
		t.Fatalf("getResponse(nil unwrap) status = %d, want 0", status)
	}

	status, _, _ = getResponse(httptest.NewRecorder(), req)
	if status != 0 {
		t.Fatalf("getResponse(default writer) status = %d, want 0", status)
	}
}

func TestGetSensitiveErrorMessage(t *testing.T) {
	msg, ok := getSensitiveErrorMessage(errorx.Sensitive(errors.New("secret"), "safe"))
	if !ok || msg != "secret" {
		t.Fatalf("getSensitiveErrorMessage(errorx.Sensitive(...)) = %q, %v; want %q, true", msg, ok, "secret")
	}

	var nilSensitive *errorx.SensitiveError
	msg, ok = getSensitiveErrorMessage(nilSensitive)
	if !ok || msg != "<nil>" {
		t.Fatalf("getSensitiveErrorMessage((*errorx.SensitiveError)(nil)) = %q, %v; want %q, true", msg, ok, "<nil>")
	}

	msg, ok = getSensitiveErrorMessage(customSensitiveError{raw: errors.New("custom-secret")})
	if !ok || msg != "custom-secret" {
		t.Fatalf("getSensitiveErrorMessage(customSensitiveError) = %q, %v; want %q, true", msg, ok, "custom-secret")
	}

	msg, ok = getSensitiveErrorMessage(customSensitiveError{})
	if !ok || msg != "<nil>" {
		t.Fatalf("getSensitiveErrorMessage(customSensitiveError{}) = %q, %v; want %q, true", msg, ok, "<nil>")
	}

	msg, ok = getSensitiveErrorMessage(errors.New("plain"))
	if ok || msg != "" {
		t.Fatalf("getSensitiveErrorMessage(plain) = %q, %v; want empty, false", msg, ok)
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
