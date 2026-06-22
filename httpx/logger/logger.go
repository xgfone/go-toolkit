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

// Package logger provides an HTTP middleware that logs requests and responses
// with the default slog logger.
package logger

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/xgfone/go-toolkit/errorx"
	"github.com/xgfone/go-toolkit/httpx"
	"github.com/xgfone/go-toolkit/internal/errors"
)

// Config configures a request logger middleware.
type Config struct {
	// Enabled reports whether the request should be logged.
	//
	// If nil, all requests are logged when the configured slog level is enabled.
	Enabled func(*http.Request) bool

	// GetRequestId returns the request id written to the "reqid" log attribute.
	//
	// If nil, the "reqid" attribute is omitted.
	GetRequestId func(*http.Request) string

	// GetRequestBody returns the request body written to the "reqbody"
	// log attribute.
	//
	// If nil, the request body is omitted.
	GetRequestBody func(*http.Request) any

	// GetResponse returns the response status, response body, and error written
	// to the "code", "resbody", "err", and "sensitive_error" log attributes.
	//
	// If nil, those response-derived attributes are omitted.
	GetResponse func(w http.ResponseWriter, r *http.Request) (status int, response any, err error)

	// PostHandle is called after PreHandle is called and before the Logger handler ends.
	PostHandle func(w http.ResponseWriter, r *http.Request)

	// PreHandle is called immediately before the next HTTP handler.
	PreHandle func(w http.ResponseWriter, r *http.Request)
}

// Logger is the alias of Middleware.
func (c Config) Logger(priority int) httpx.Middleware {
	return c.Middleware(priority)
}

// Middleware returns a new logger middleware with the given priority.
func (c Config) Middleware(priority int) httpx.Middleware {
	logger := &logger{
		enabled:        c.Enabled,
		preHandle:      c.PreHandle,
		postHandle:     c.PostHandle,
		getResponse:    c.GetResponse,
		getRequestId:   c.GetRequestId,
		getRequestBody: c.GetRequestBody,

		level: slog.LevelInfo,
		prio:  priority,
	}
	return logger
}

// NewDefaultConfig returns a new default Config.
//
//	GetRequestId: reads the request id from the X-Request-Id header.
//	GetResponse: reads the response status, body, and error	from the httpx.Context or httpx.ResponseWriter.
func NewDefaultConfig() Config {
	return Config{
		GetResponse:  getResponse,
		GetRequestId: getRequestId,
	}
}

func getRequestId(r *http.Request) string {
	return r.Header.Get("X-Request-Id")
}

func getResponse(w http.ResponseWriter, r *http.Request) (status int, response any, err error) {
	if c := httpx.GetContext(r.Context()); c != nil {
		return c.StatusCode(), c.Response, c.Error
	}

	for range 10 {
		switch rw := w.(type) {
		case httpx.ResponseWriter:
			status = rw.StatusCode()
			return

		case interface{ Unwrap() http.ResponseWriter }:
			w = rw.Unwrap()
			if w == nil {
				return
			}

		default:
			return
		}
	}

	return
}

type logger struct {
	enabled        func(r *http.Request) bool
	getRequestId   func(r *http.Request) string
	getRequestBody func(r *http.Request) any
	getResponse    func(w http.ResponseWriter, r *http.Request) (int, any, error)
	postHandle     func(w http.ResponseWriter, r *http.Request)
	preHandle      func(w http.ResponseWriter, r *http.Request)

	level slog.Level
	next  http.Handler
	prio  int
}

func (l *logger) Priority() int {
	return l.prio
}

func (l *logger) HTTPHandler(next http.Handler) http.Handler {
	if next == nil {
		panic("Logger.HTTPHandler: next http.Handler is nil")
	}

	_l := *l
	_l.next = next
	return &_l
}

func enableLevel(ctx context.Context, level slog.Level) bool {
	return slog.Default().Enabled(ctx, level)
}

func (l *logger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if l.next == nil {
		w.WriteHeader(500)
		_, _ = io.WriteString(w, "Logger: NO NEXT HANDLER")
		return
	}

	ctx := r.Context()
	if !(enableLevel(ctx, l.level) && (l.enabled == nil || l.enabled(r))) {
		l.next.ServeHTTP(w, r)
		return
	}

	if l.preHandle != nil {
		l.preHandle(w, r)
	}
	if l.postHandle != nil {
		defer l.postHandle(w, r)
	}

	start := time.Now()
	l.next.ServeHTTP(w, r)
	cost := time.Since(start)

	attrs := getattrs()
	defer putattrs(attrs)

	if l.getRequestId != nil {
		attrs.Append(slog.String("reqid", l.getRequestId(r)))
	}

	attrs.Append(
		slog.String("raddr", r.RemoteAddr),
		slog.String("method", r.Method),
		slog.String("host", r.Host),
		slog.String("path", r.URL.Path),
		slog.String("query", r.URL.RawQuery),
		slog.String("cost", cost.String()),
	)

	var code int
	var response any
	var err error

	if l.getResponse != nil {
		code, response, err = l.getResponse(w, r)
		attrs.Append(slog.Int("code", code))
	}

	attrs.Append(
		slog.Any("reqheader", r.Header),
		slog.Any("resheader", w.Header()),
	)

	if l.getRequestBody != nil {
		attrs.Append(slog.Any("reqbody", l.getRequestBody(r)))
	}
	if response != nil {
		attrs.Append(slog.Any("resbody", response))
	}
	if err != nil {
		attrs.Append(slog.String("err", err.Error()))
		if errmsg, ok := getSensitiveErrorMessage(err); ok {
			attrs.Append(slog.String("sensitive_error", errmsg))
		}
	}

	slog.LogAttrs(ctx, l.level, "log http request", attrs.Attrs...)
}

func getSensitiveErrorMessage(err error) (msg string, ok bool) {
	if se, ok := errors.AsType[*errorx.SensitiveError](err); ok {
		if se != nil {
			if e := se.SensitiveError(); e != nil {
				return e.Error(), true
			}
		}
		return "<nil>", true
	}

	if se, ok := errors.AsType[errors.SensitiveError](err); ok {
		if e := se.SensitiveError(); e != nil {
			return e.Error(), true
		}
		return "<nil>", true
	}

	return "", false
}

/// ----------------------------------------------------------------------- ///

type attrswrapper struct{ Attrs []slog.Attr }

func (w *attrswrapper) Reset()                { ; clear(w.Attrs); w.Attrs = w.Attrs[:0] }
func (w *attrswrapper) Append(a ...slog.Attr) { w.Attrs = append(w.Attrs, a...) }

var attrspool = &sync.Pool{New: func() any {
	return &attrswrapper{Attrs: make([]slog.Attr, 0, 36)}
}}

func getattrs() *attrswrapper  { return attrspool.Get().(*attrswrapper) }
func putattrs(w *attrswrapper) { w.Reset(); attrspool.Put(w) }
