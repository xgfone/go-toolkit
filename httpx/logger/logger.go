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
// with the default [slog] logger.
package logger

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/xgfone/go-toolkit/errorx"
	"github.com/xgfone/go-toolkit/httpx"
)

// Config configures a request logger middleware.
type Config struct {
	// Enabled reports whether the request should be logged.
	//
	// If nil, all requests are logged when the configured [slog] level is enabled.
	Enabled func(*http.Request) bool

	// GetRequestId returns the request id written to the "reqid" log attribute.
	//
	// If nil, the "reqid" attribute is omitted.
	GetRequestId func(*http.Request) string

	// GetLogAttrs returns log attributes after the next handler returns and
	// before [Config.PostHandle] runs. It is only called for logged requests.
	// It may read the final response headers and the request's [httpx.Context].
	// Returned attributes are appended after the built-in request attributes.
	//
	// [NewDefaultConfig] supplies the "code", "err", and "sensitive_error"
	// attributes. Replacing this callback replaces those default attributes;
	// wrap the original callback to preserve them when adding other fields.
	// Headers, query parameters, and bodies must be explicitly added here.
	//
	// If nil, only the built-in request attributes are logged.
	// Callbacks must be safe for concurrent use.
	GetLogAttrs func(w http.ResponseWriter, r *http.Request) []slog.Attr

	// PostHandle is called after [Config.PreHandle] and before the handler
	// returned by [Config.Middleware] ends.
	PostHandle func(w http.ResponseWriter, r *http.Request)

	// PreHandle is called immediately before the next HTTP handler.
	PreHandle func(w http.ResponseWriter, r *http.Request)
}

// Middleware returns a new logger middleware with the given priority.
func (c Config) Middleware(priority int) httpx.Middleware {
	logger := &logger{
		enabled:      c.Enabled,
		preHandle:    c.PreHandle,
		postHandle:   c.PostHandle,
		getLogAttrs:  c.GetLogAttrs,
		getRequestId: c.GetRequestId,

		level: slog.LevelInfo,
		prio:  priority,
	}
	return logger
}

// NewDefaultConfig returns a new default [Config].
//
//   - [Config.Enabled]: logs requests by default, except for the root path "/".
//   - [Config.GetRequestId]: reads the request id from the X-Request-Id header.
//   - [Config.GetLogAttrs]: reads the response status and error from the
//     [httpx.Context] or [httpx.ResponseWriter]. It logs the safe error message
//     as "err" and the original sensitive error, when present, as "sensitive_error".
//     Headers, query parameters, and bodies are omitted.
//
// Sensitive error messages may contain confidential details. Use this default
// with access-controlled internal logs, or replace [Config.GetLogAttrs] to filter them.
func NewDefaultConfig() Config {
	return Config{
		Enabled:      enabled,
		GetRequestId: getRequestId,
		GetLogAttrs:  getLogAttrs,
	}
}

func enabled(r *http.Request) bool {
	return r.URL.Path != "/"
}

func getRequestId(r *http.Request) string {
	return r.Header.Get("X-Request-Id")
}

func getLogAttrs(w http.ResponseWriter, r *http.Request) []slog.Attr {
	status, err := getResponse(w, r)
	attrs := make([]slog.Attr, 0, 3)
	attrs = append(attrs, slog.Int("code", status))
	if err != nil {
		attrs = append(attrs, slog.String("err", err.Error()))
		if msg, ok := getSensitiveErrorMessage(err); ok {
			attrs = append(attrs, slog.String("sensitive_error", msg))
		}
	}
	return attrs
}

func getResponse(w http.ResponseWriter, r *http.Request) (status int, err error) {
	if c := httpx.GetContext(r.Context()); c != nil {
		return c.StatusCode(), c.Error
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
	enabled      func(r *http.Request) bool
	getRequestId func(r *http.Request) string
	getLogAttrs  func(w http.ResponseWriter, r *http.Request) []slog.Attr
	postHandle   func(w http.ResponseWriter, r *http.Request)
	preHandle    func(w http.ResponseWriter, r *http.Request)

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
	if !enableLevel(ctx, l.level) || (l.enabled != nil && !l.enabled(r)) {
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
		slog.String("cost", cost.String()),
	)

	if l.getLogAttrs != nil {
		attrs.Append(l.getLogAttrs(w, r)...)
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

	if se, ok := errors.AsType[_SensitiveError](err); ok {
		if e := se.SensitiveError(); e != nil {
			return e.Error(), true
		}
		return "<nil>", true
	}

	return "", false
}

type _SensitiveError interface {
	SensitiveError() error
	error
}

/// ----------------------------------------------------------------------- ///

type attrswrapper struct{ Attrs []slog.Attr }

func (w *attrswrapper) Reset()                { clear(w.Attrs); w.Attrs = w.Attrs[:0] }
func (w *attrswrapper) Append(a ...slog.Attr) { w.Attrs = append(w.Attrs, a...) }

var attrspool = &sync.Pool{New: func() any {
	return &attrswrapper{Attrs: make([]slog.Attr, 0, 36)}
}}

func getattrs() *attrswrapper  { return attrspool.Get().(*attrswrapper) }
func putattrs(w *attrswrapper) { w.Reset(); attrspool.Put(w) }
