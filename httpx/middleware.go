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
	"slices"
)

var (
	_ Middleware = Middlewares(nil)
	_ Middleware = MiddlewareFunc(nil)
)

// Middleware is a HTTP handler middleware.
type Middleware interface {
	HTTPHandler(next http.Handler) http.Handler
}

// MiddlewareFunc is the middleware function.
type MiddlewareFunc func(next http.Handler) http.Handler

// HTTPHandler implements the interface Middleware.
func (f MiddlewareFunc) HTTPHandler(next http.Handler) http.Handler { return f(next) }

// Middlewares is a set of middlewares.
type Middlewares []Middleware

// HTTPHandler implements the Middleware interface to returns a new HTTP handler.
func (ms Middlewares) HTTPHandler(next http.Handler) http.Handler {
	for i := len(ms) - 1; i >= 0; i-- {
		next = ms[i].HTTPHandler(next)
	}
	return next
}

// Sort tries to sort the middlewares by priority from big to small.
//
// If the middleware implements the interface{ Priority() int }, use it.
// Otherwise, use 1 instead.
func (ms Middlewares) Sort() {
	slices.SortStableFunc(ms, func(a, b Middleware) int {
		return getPriority(b) - getPriority(a) // From bigger to smaller
	})
}

// PriorityMiddleware returns a middleware with the given priority.
func PriorityMiddleware(priority int, m Middleware) Middleware {
	return _PriorityMiddleware{Middleware: m, priority: priority}
}

// PriorityMiddlewareFunc returns a middleware with the given priority.
func PriorityMiddlewareFunc(priority int, m MiddlewareFunc) Middleware {
	return PriorityMiddleware(priority, m)
}

type _PriorityMiddleware struct {
	Middleware
	priority int
}

func (m _PriorityMiddleware) Priority() int {
	return m.priority
}

func getPriority(m Middleware) int {
	if p, ok := m.(_Priority); ok {
		return p.Priority()
	}
	return 1
}

type _Priority interface {
	Priority() int
}
