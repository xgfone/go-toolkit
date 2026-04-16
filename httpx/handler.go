// Copyright 2025 xgfone
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

package httpx

import (
	"net/http"

	"github.com/xgfone/go-toolkit/internal/render"
)

// Pre-define some http handlers.
var (
	Handler200 = handler(200)
	Handler201 = handler(201)
	Handler204 = handler(204)
	Handler400 = handler(400)
	Handler401 = handler(401)
	Handler403 = handler(403)
	Handler404 = handler(404)
	Handler500 = handler(500)
	Handler501 = handler(501)
	Handler502 = handler(502)
	Handler503 = handler(503)
)

func handler(code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if code == 404 {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(code)
	})
}

// JSON sends the response by the json format to the client.
func JSON(w http.ResponseWriter, code int, v any) (err error) {
	return render.JSON(w, code, v)
}

// XML sends the response by the xml format to the client.
func XML(w http.ResponseWriter, code int, v any) (err error) {
	return render.XML(w, code, v)
}
