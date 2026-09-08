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
	"maps"
	"net/http"

	"github.com/xgfone/go-toolkit/httpx"
)

func newServeMuxBackend(routes []httpx.Route, notfound http.Handler) http.Handler {
	server := http.NewServeMux()
	for i := range routes {
		registerRoute(server, &routes[i])
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, pattern := server.Handler(r)
		if pattern != "" {
			// ServeHTTP also populates Pattern and PathValue for matched routes.
			server.ServeHTTP(w, r)
			return
		}

		// An empty pattern can mean either 404 or 405. Only replace 404;
		// a catch-all route would hide the mux's 405 and Allow header.
		r.Pattern = ""
		rw := &routingErrorWriter{ResponseWriter: w, header: w.Header().Clone()}
		handler.ServeHTTP(rw, r)
		if rw.status == http.StatusNotFound {
			notfound.ServeHTTP(w, r)
		}
	})
}

// Only ServeMux-generated errors use this writer. Matched handlers receive
// their original writer, including its optional interfaces.
type routingErrorWriter struct {
	http.ResponseWriter
	header http.Header
	status int
}

func (w *routingErrorWriter) Header() http.Header         { return w.header }
func (w *routingErrorWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *routingErrorWriter) WriteHeader(code int) {
	w.status = code
	if code != http.StatusNotFound {
		header := w.ResponseWriter.Header()
		clear(header)
		maps.Copy(header, w.header)
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *routingErrorWriter) Write(data []byte) (int, error) {
	if w.status == http.StatusNotFound {
		return len(data), nil
	}
	return w.ResponseWriter.Write(data)
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
