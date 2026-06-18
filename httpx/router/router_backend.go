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

	"github.com/xgfone/go-toolkit/httpx"
)

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
	var failed bool
	pattern := route.Pattern()
	defer recoverRoutePanic(pattern, &failed)
	server.Handle(pattern, route.Handler)
	route.Online = !failed
}

func recoverRoutePanic(pattern string, failed *bool) {
	if r := recover(); r != nil {
		slog.Error("fail to register the http route", "pattern", pattern, "err", r)
		*failed = true
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
