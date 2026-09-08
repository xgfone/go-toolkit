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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestConcurrentRoutingResponses(t *testing.T) {
	r := New()
	r.Path("/items/{id}").GetFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Flusher); !ok {
			t.Error("handler lost original writer")
		}

		w.WriteHeader(http.StatusNotFound) // An application 404 must bypass fallback.
		_, _ = io.WriteString(w, r.PathValue("id"))
	})

	r.SetNotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("X-Content-Type-Options") != "" {
			t.Error("discarded mux headers reached fallback")
		}

		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, "fallback")
	}))

	var wg sync.WaitGroup
	for worker := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 32 {
				id := fmt.Sprintf("%d-%d", worker, i)
				for _, tc := range []struct {
					method, path, body string
					status             int
				}{
					{"GET", "/items/" + id, id, http.StatusNotFound},
					{"GET", "/missing", "fallback", http.StatusTeapot},
					{"POST", "/items/" + id, "fallback", http.StatusTeapot},
				} {
					rec := httptest.NewRecorder()
					rec.Header().Set("X-Request", id)
					r.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
					if rec.Code != tc.status || rec.Body.String() != tc.body ||
						rec.Header().Get("X-Request") != id {
						t.Errorf("request %s: %d %s %v", id, rec.Code, rec.Body, rec.Header())
					}
				}
			}
		}()
	}
	wg.Wait()
}
