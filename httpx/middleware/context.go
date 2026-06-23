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

	"github.com/xgfone/go-toolkit/httpx"
)

var (
	ctxAcquireContext func() *httpx.Context
	ctxReleaseContext func(*httpx.Context)
)

func init() {
	SetAcquireContext(httpx.AcquireContext)
	SetReleaseContext(httpx.ReleaseContext)
}

// SetAcquireContext resets the acquire function for the context.
//
// Default: httpx.AcquireContext
func SetAcquireContext(acquire func() *httpx.Context) {
	if acquire == nil {
		panic("middleware.SetAcquireContext: acquire function is nil")
	}
	ctxAcquireContext = acquire
}

// SetReleaseContext resets the release function for the context.
//
// Default: httpx.ReleaseContext
func SetReleaseContext(release func(*httpx.Context)) {
	if release == nil {
		panic("middleware.SetReleaseContext: release function is nil")
	}
	ctxReleaseContext = release
}

// Context is an http middleware to allocate a context and put it
// into the http request, then release it after handling the http request.
func Context(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c := httpx.GetContext(r.Context()); c == nil {
			c = ctxAcquireContext()
			defer ctxReleaseContext(c)

			c.Reset(w, r.WithContext(httpx.SetContext(r.Context(), c)))
			next.ServeHTTP(c.ResponseWriter, c.Request)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
