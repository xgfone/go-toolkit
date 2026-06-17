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

package middleware

import (
	"net/http"
	"strings"
)

// BlockPath returns an http middleware to stop handling the request and write
// the status code if the request path matches path.
func BlockPath(path string, statusCode int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == path {
				w.WriteHeader(statusCode)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// BlockPathPrefix returns an http middleware to stop handling the request and
// write the status code if the request path has prefix.
func BlockPathPrefix(prefix string, statusCode int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, prefix) {
				w.WriteHeader(statusCode)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
