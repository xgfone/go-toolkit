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
	"bytes"
	"io"
	"net/http"

	"github.com/xgfone/go-toolkit/unsafex"
)

func write(w http.ResponseWriter, b *bytes.Buffer) (err error) {
	n, err := w.Write(unsafex.Bytes(b.String()))
	if err == nil && n != b.Len() {
		err = io.ErrShortWrite
	}
	return
}
