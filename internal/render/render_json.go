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

package render

import (
	"net/http"

	"github.com/xgfone/go-toolkit/internal/pools"
	"github.com/xgfone/go-toolkit/jsonx"
)

// JSON sends the response by the json format to the client.
func JSON(w http.ResponseWriter, code int, v any) (err error) {
	if v == nil {
		w.WriteHeader(code)
		return
	}

	pool, buf := pools.GetBuffer(64 * 1024) // 64KB
	defer pools.PutBuffer(pool, buf)

	if err = jsonx.MarshalWriter(buf, v); err == nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(code)
		err = write(w, buf)
	}

	return
}
