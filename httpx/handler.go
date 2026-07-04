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
	"io"
	"net/http"
	"slices"

	"github.com/xgfone/go-toolkit/internal/render"
)

// Pre-define some http handlers.
var (
	Handler200 = handler(200)
	Handler201 = handler(201)
	Handler204 = handler(204)
	Handler400 = handler(400)
	Handler401 = handler(401)
	Handler403 = handler(403)
	Handler404 = handler(404)
	Handler500 = handler(500)
	Handler501 = handler(501)
	Handler502 = handler(502)
	Handler503 = handler(503)
)

func handler(code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if code == 404 {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(code)
	})
}

// JSON sends the response by the json format to the client.
func JSON(w http.ResponseWriter, code int, v any) (err error) {
	return render.JSON(w, code, v)
}

// XML sends the response by the xml format to the client.
func XML(w http.ResponseWriter, code int, v any) (err error) {
	return render.XML(w, code, v)
}

var (
	_ http.Handler = ContextHandler(nil)
	_ Middleware   = ContextHandler(nil)
)

// ContextHandler is the handler function for the request context.
type ContextHandler func(c *Context) error

// ServeHTTP implements the http.Handler interface.
//
// Note: a Context must be got by GetContext from the request context.
func (h ContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := GetContext(r.Context())
	c.AppendError(h(c))
}

// HTTPHandler implements the Middleware interface.
func (h ContextHandler) HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c := GetContext(r.Context()); c == nil {
			w.WriteHeader(500)
			_, _ = io.WriteString(w, "missing httpx.Context")
		} else if err := h(c); err != nil {
			c.AppendError(err)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// ContextHandlerAnd combines multiple ContextHandlers with AND logic and
// returns a ContextHandler that calls each handler in order.
//
// If any handler returns a non-nil error, the combined handler short-circuits
// and returns that error immediately without calling the remaining handlers.
// If all handlers return nil, the combined handler returns nil.
//
// If no handlers are provided, it returns nil.
// If only one handler is provided, it returns that handler directly.
func ContextHandlerAnd(handlers ...ContextHandler) ContextHandler {
	return contextHandlers(true, handlers...)
}

// ContextHandlerOr combines multiple ContextHandlers with OR logic and
// returns a ContextHandler that calls each handler in order.
//
// If any handler returns nil (success), the combined handler short-circuits
// and returns nil immediately without calling the remaining handlers.
// If all handlers return a non-nil error, the combined handler returns
// the last error encountered.
//
// If no handlers are provided, it returns nil.
// If only one handler is provided, it returns that handler directly.
func ContextHandlerOr(handlers ...ContextHandler) ContextHandler {
	return contextHandlers(false, handlers...)
}

func contextHandlers(fastfail bool, handlers ...ContextHandler) ContextHandler {
	switch len(handlers) {
	case 0:
		return nil

	case 1:
		return handlers[0]
	}

	handlers = slices.Clone(handlers)
	return func(c *Context) (err error) {
		for i := range len(handlers) {
			if err = handlers[i](c); fastfail == (err != nil) {
				return
			}
		}
		return
	}
}
