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
	"slices"
	"strings"

	"github.com/xgfone/go-toolkit/httpx"
	"github.com/xgfone/go-toolkit/slicex"
)

func normalizePath(path string) string {
	return strings.ReplaceAll(path, "//", "/")
}

func (r *Router) register(mdws httpx.Middlewares, route httpx.Route) {
	route.Handler = mdws.HTTPHandler(route.Handler)
	r.routes = append(r.routes, route)
}

// Register adds a route to the router.
func (r *Router) Register(route httpx.Route) {
	if route.Path == "" {
		panic("Router.Register: the route path must not be empty")
	}
	if route.Handler == nil {
		panic("Router.Register: the route handler must not be nil")
	}
	r.register(nil, route)
}

// Routes returns all registered routes.
//
// Note: The returned routes should not be modified.
func (r *Router) Routes() []httpx.Route {
	return r.routes
}

// Group returns a new route with the group path prefix.
func (r *Router) Group(group string) Route {
	return (Route{router: r}).Group(group)
}

// Path returns a new route with the path.
func (r *Router) Path(path string) Route {
	return (Route{router: r}).Path(path)
}

// Host returns a new route with the host.
func (r *Router) Host(host string) Route {
	return (Route{router: r}).Host(host)
}

// Router returns the parent router.
func (r Route) Router() *Router {
	return r.router
}

// Route is a http request route builder.
type Route struct {
	auth httpx.Middleware
	mdws httpx.Middlewares

	host   string
	path   string
	group  string
	method string

	router *Router
}

// Use adds middlewares to the route, which will be applied to the route handler,
// so they are called after routing.
func (r Route) Use(mdws ...httpx.Middleware) Route {
	r.mdws = slicex.Merge(r.mdws, mdws)
	return r
}

// Auth resets the authentication middleware for the route.
func (r Route) Auth(auth httpx.Middleware) Route {
	r.auth = auth
	return r
}

// Path sets the route path.
//
// Note: The path must be empty or start with /. If the path is empty or "/",
// use the group prefix without the suffix "/" as the path instead.
func (r Route) Path(path string) Route {
	path = normalizePath(path)
	if path == "" || path == "/" {
		r.path = r.group
		return r
	}

	if path[0] != '/' {
		panic("Route.Path: path must starts with /")
	}

	r.path = r.group + path
	return r
}

// Host sets the route host constraint.
func (r Route) Host(host string) Route {
	r.host = host
	return r
}

// Group sets the route group prefix.
//
// Note: The group must be empty or start with /.
func (r Route) Group(group string) Route {
	group = strings.TrimRight(group, "/")
	group = normalizePath(group)
	if group == "" || group == "/" {
		return r
	}

	if group[0] != '/' {
		panic("Route.Group: group must starts with /")
	}

	r.group += group
	return r
}

// Prefix returns the route path prefix.
func (r Route) Prefix() string {
	return r.group
}

// Method sets the HTTP method for the route.
func (r Route) Method(method string) Route {
	r.method = method
	return r
}

// Handler registers the route with the givien handler.
func (r Route) Handler(handler http.Handler) Route {
	if handler == nil {
		panic("Route.Handler: handler must not be nil")
	}

	route := httpx.Route{
		Host:    r.host,
		Path:    r.path,
		Method:  r.method,
		Handler: handler,
	}

	if route.Path == "" {
		route.Path = "/"
	}

	var mdws httpx.Middlewares
	if r.auth == nil {
		mdws = slices.Clone(r.mdws)
	} else if len(r.mdws) == 0 {
		mdws = httpx.Middlewares{r.auth}
	} else {
		mdws = slicex.Merge([]httpx.Middleware{r.auth}, r.mdws)
	}

	mdws.Sort()
	r.router.register(mdws, route)
	return r
}

// HandlerFunc registers the route with the given handler function.
func (r Route) HandlerFunc(handler http.HandlerFunc) Route {
	return r.Handler(handler)
}

// Put registers a PUT route with the handler.
func (r Route) Put(handler http.Handler) Route {
	return r.Method(http.MethodPut).Handler(handler)
}

// Get registers a GET route with the handler.
func (r Route) Get(handler http.Handler) Route {
	return r.Method(http.MethodGet).Handler(handler)
}

// Post registers a POST route with the handler.
func (r Route) Post(handler http.Handler) Route {
	return r.Method(http.MethodPost).Handler(handler)
}

// Head registers a HEAD route with the handler.
func (r Route) Head(handler http.Handler) Route {
	return r.Method(http.MethodHead).Handler(handler)
}

// Patch registers a PATCH route with the handler.
func (r Route) Patch(handler http.Handler) Route {
	return r.Method(http.MethodPatch).Handler(handler)
}

// Delete registers a DELETE route with the handler.
func (r Route) Delete(handler http.Handler) Route {
	return r.Method(http.MethodDelete).Handler(handler)
}

// Options registers an OPTIONS route with the handler.
func (r Route) Options(handler http.Handler) Route {
	return r.Method(http.MethodOptions).Handler(handler)
}

// PutFunc registers a PUT route with the handler function.
func (r Route) PutFunc(handler http.HandlerFunc) Route {
	return r.Method(http.MethodPut).Handler(handler)
}

// GetFunc registers a GET route with the handler function.
func (r Route) GetFunc(handler http.HandlerFunc) Route {
	return r.Method(http.MethodGet).Handler(handler)
}

// PostFunc registers a POST route with the handler function.
func (r Route) PostFunc(handler http.HandlerFunc) Route {
	return r.Method(http.MethodPost).Handler(handler)
}

// HeadFunc registers a HEAD route with the handle functionr.
func (r Route) HeadFunc(handler http.HandlerFunc) Route {
	return r.Method(http.MethodHead).Handler(handler)
}

// PatchFunc registers a PATCH route with the handler function.
func (r Route) PatchFunc(handler http.HandlerFunc) Route {
	return r.Method(http.MethodPatch).Handler(handler)
}

// DeleteFunc registers a DELETE route with the handler function.
func (r Route) DeleteFunc(handler http.HandlerFunc) Route {
	return r.Method(http.MethodDelete).Handler(handler)
}

// OptionsFunc registers an OPTIONS route with the handler function.
func (r Route) OptionsFunc(handler http.HandlerFunc) Route {
	return r.Method(http.MethodOptions).Handler(handler)
}

// PutContext registers a PUT route with the context handler.
func (r Route) PutContext(handler httpx.ContextHandler) Route {
	return r.Method(http.MethodPut).Handler(handler)
}

// GetContext registers a GET route with the context handler.
func (r Route) GetContext(handler httpx.ContextHandler) Route {
	return r.Method(http.MethodGet).Handler(handler)
}

// PostContext registers a POST route with the context handler.
func (r Route) PostContext(handler httpx.ContextHandler) Route {
	return r.Method(http.MethodPost).Handler(handler)
}

// HeadContext registers a HEAD route with the context handler.
func (r Route) HeadContext(handler httpx.ContextHandler) Route {
	return r.Method(http.MethodHead).Handler(handler)
}

// PatchContext registers a PATCH route with the context handler.
func (r Route) PatchContext(handler httpx.ContextHandler) Route {
	return r.Method(http.MethodPatch).Handler(handler)
}

// DeleteContext registers a DELETE route with the context handler.
func (r Route) DeleteContext(handler httpx.ContextHandler) Route {
	return r.Method(http.MethodDelete).Handler(handler)
}

// OptionsContext registers an OPTIONS route with the context handler.
func (r Route) OptionsContext(handler httpx.ContextHandler) Route {
	return r.Method(http.MethodOptions).Handler(handler)
}
