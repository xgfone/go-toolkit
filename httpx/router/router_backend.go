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
	"log/slog"
	"net/http"
	"strings"

	"github.com/xgfone/go-toolkit/httpx"
)

func newServeMuxBackend(routes []httpx.Route, notfound http.Handler) http.Handler {
	server := http.NewServeMux()
	var hasCatchAll bool
	for i := range routes {
		route := &routes[i]
		registerRoute(server, route)
		if route.Online && route.Host == "" && route.Method == "" {
			hasCatchAll = hasCatchAll || isCatchAllPath(route.Path)
		}
	}

	// Unmatched paths and methods both reach this fallback.
	if !hasCatchAll {
		server.Handle("/", notfound)
	}

	return server
}

// Only called for successfully registered paths: ServeMux has already
// validated wildcard names, so no separate identifier parser is needed.
func isCatchAllPath(path string) bool {
	return path == "/" || (strings.HasPrefix(path, "/{") &&
		strings.HasSuffix(path, "...}") && strings.Count(path, "/") == 1)
}

func registerRoute(server *http.ServeMux, route *httpx.Route) {
	// The caller may supply an already-online route. A failed registration
	// must clear that stale state; recovery does not resume after Handle.
	route.Online = false
	pattern := route.Pattern()
	defer recoverRoutePanic(pattern)
	server.Handle(pattern, route.Handler)
	route.Online = true
}

func recoverRoutePanic(pattern string) {
	if r := recover(); r != nil {
		slog.Error("fail to register the http route", "pattern", pattern, "err", r)
	}
}
