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
	"encoding/xml"
	"net/http"

	"github.com/xgfone/go-toolkit/internal/pools"
)

// XML sends the response by the xml format to the client.
func XML(w http.ResponseWriter, code int, v any) (err error) {
	if v == nil {
		w.WriteHeader(code)
		return
	}

	pool, buf := pools.GetBuffer(64 * 1024) // 64KB
	defer pools.PutBuffer(pool, buf)

	_, _ = buf.WriteString(xml.Header)
	if err = xml.NewEncoder(buf).Encode(v); err == nil {
		w.Header().Set("Content-Type", "application/xml; charset=UTF-8")
		w.WriteHeader(code)
		err = write(w, buf)
	}

	return
}
