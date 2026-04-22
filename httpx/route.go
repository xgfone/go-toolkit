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

package httpx

import (
	"bytes"
	"io"
	"net/http"
)

// Route is a http request route.
type Route struct {
	Host   string `json:",omitempty"`
	Path   string `json:",omitempty"`
	Method string `json:",omitempty"`

	// Whether the route is registered successfully.
	Online bool `json:",omitempty"`

	http.Handler `json:"-"`
}

// Pattern returns the route pattern for Go 1.22+.
func (r Route) Pattern() string {
	buf := bytes.NewBuffer(nil)
	buf.Grow(len(r.Host) + len(r.Path) + len(r.Method) + 1)
	_, _ = r.WriteTo(buf)
	return buf.String()
}

var _ io.WriterTo = (Route{})

// WriteTo implements the io.WriterTo interface to write the route pattern to w.
func (r Route) WriteTo(w io.Writer) (n int64, err error) {
	err = tryWriteString(w, r.Method, &n, err)

	if r.Method != "" {
		err = tryWriteString(w, " ", &n, err)
	}

	err = tryWriteString(w, r.Host, &n, err)
	err = tryWriteString(w, r.Path, &n, err)
	return
}

func tryWriteString(w io.Writer, s string, m *int64, err error) error {
	if err != nil || s == "" {
		return err
	}

	n, err := io.WriteString(w, s)
	*m += int64(n)
	return err
}
