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

package router

import (
	"go/token"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/xgfone/go-toolkit/httpx"
)

// DefaultRouter is the global default router.
var DefaultRouter = New()

// Router is an HTTP server router.
type Router struct {
	middlewares httpx.Middlewares
	newBackend  func(routes []httpx.Route, notfound http.Handler) http.Handler
	onroutes    func(httpx.Route, httpx.Middlewares) httpx.Route
	notfound    http.Handler

	routes []httpx.Route
	server http.Handler
	once   sync.Once
}

// New creates a new Router.
func New() *Router {
	r := &Router{}
	r.SetBackend(newServeMuxBackend)
	r.SetNotFound(httpx.Handler404)
	r.OnRegister(r.onRegister)
	return r
}

// SetBackend sets the backend handler factory.
//
// Default: use *http.ServeMux as the backend.
func (r *Router) SetBackend(new func(routes []httpx.Route, notfound http.Handler) http.Handler) {
	if new == nil {
		panic("Router.SetBackend: new function must not be nil")
	}
	r.newBackend = new
}

// SetNotFound sets the not found handler.
//
// Default: use httpx.Handler404.
func (r *Router) SetNotFound(notfound http.Handler) {
	if notfound == nil {
		panic("Router.SetNotFound: the NotFound handler must not be nil")
	}
	r.notfound = notfound
}

// Use adds global middlewares to the router, which are called before routing.
func (r *Router) Use(mdws ...httpx.Middleware) {
	r.middlewares = append(r.middlewares, mdws...)
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.once.Do(r.initServer)
	r.server.ServeHTTP(w, req)
}

func (r *Router) initServer() {
	r.middlewares.Sort()
	r.server = r.newBackend(r.routes, r.notfound)
	r.server = r.middlewares.HTTPHandler(r.server)
}

func newServeMuxBackend(routes []httpx.Route, notfound http.Handler) http.Handler {
	var hasall bool
	server := http.NewServeMux()
	for i := range routes {
		route := &routes[i]
		registerRoute(server, route)
		hasall = hasall || route.Path == "/" || isWildcardRoute(route.Path)
	}

	if !hasall {
		server.Handle("/", notfound)
	}
	return server
}

func registerRoute(server *http.ServeMux, route *httpx.Route) {
	pattern := route.Pattern()
	defer recoverRoutePanic(pattern)
	server.Handle(pattern, route.Handler)
}

func recoverRoutePanic(pattern string) {
	if r := recover(); r != nil {
		slog.Error("fail to register the http route", "pattern", pattern, "err", r)
	}
}

func isWildcardRoute(path string) bool {
	return strings.HasPrefix(path, "/{") &&
		strings.HasSuffix(path, "...}") &&
		isIdentifier(path[len("/{"):len(path)-len("...}")])
}

func isIdentifier(name string) bool {
	return name == "" || token.IsIdentifier(name)
}
