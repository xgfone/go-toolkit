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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xgfone/go-toolkit/httpx"
)

func TestRegressionRoutesReadRace(t *testing.T) {
	r := New()
	for i := 0; i < 1000; i++ {
		r.Path(fmt.Sprintf("/audit/%d", i)).Get(httpx.Handler200)
	}

	routes := r.Routes()
	done := make(chan struct{})
	go func() {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/audit/0", nil))
		close(done)
	}()

	defer func() { <-done }()
	for {
		select {
		case <-done:
			if !r.Routes()[0].Online {
				t.Error("a new snapshot must reflect successful registration")
			}
			return

		default:
			for i := range routes {
				if routes[i].Online {
					t.Error("an earlier snapshot was mutated during registration")
					return
				}
			}
		}
	}
}
