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
	"net/http"
	"sync"

	"github.com/xgfone/go-toolkit/httpx"
)

// DefaultRouter is the global default router.
var DefaultRouter = New()

// Router is an HTTP server router.
type Router struct {
	middlewares httpx.Middlewares
	newBackend  func(routes []httpx.Route, notfound http.Handler) http.Handler
	notfound    http.Handler

	rmutex sync.RWMutex
	routes []httpx.Route

	server http.Handler
	once   sync.Once
}

// New creates a new Router.
func New() *Router {
	r := &Router{}
	r.SetBackend(newServeMuxBackend)
	r.SetNotFound(httpx.Handler404)
	return r
}

// SetBackend sets the backend handler factory.
//
// Default: use *http.ServeMux as the backend.
func (r *Router) SetBackend(new func(routes []httpx.Route, notfound http.Handler) http.Handler) {
	if new == nil {
		panic("Router.SetBackend: new function must not be nil")
	}
	r.rmutex.Lock()
	r.newBackend = new
	r.rmutex.Unlock()
}

// SetNotFound sets the not found handler.
//
// Default: use httpx.Handler404.
func (r *Router) SetNotFound(notfound http.Handler) {
	if notfound == nil {
		panic("Router.SetNotFound: the NotFound handler must not be nil")
	}
	r.rmutex.Lock()
	r.notfound = notfound
	r.rmutex.Unlock()
}

// Use adds global middlewares to the router, which are called before routing.
func (r *Router) Use(mdws ...httpx.Middleware) {
	r.rmutex.Lock()
	r.middlewares = append(r.middlewares, mdws...)
	r.rmutex.Unlock()
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.once.Do(r.initServer)
	r.server.ServeHTTP(w, req)
}

func (r *Router) initServer() {
	r.rmutex.Lock()
	defer r.rmutex.Unlock()

	r.middlewares.Sort()
	r.server = r.newBackend(r.routes, r.notfound)
	r.server = r.middlewares.HTTPHandler(r.server)
}
