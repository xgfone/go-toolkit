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
	"fmt"
	"net/http"
)

// ResponseWriter is an extended http.ResponseWriter.
type ResponseWriter interface {
	http.ResponseWriter
	WroteHeader() bool
	StatusCode() int
}

// NewResponseWriter returns a ResponseWriter.
//
// Note: If w has implemented ResponseWriter, return it directly.
func NewResponseWriter(w http.ResponseWriter) ResponseWriter {
	if rw, ok := w.(ResponseWriter); ok {
		return rw
	}
	return &_ResponseWriter{ResponseWriter: w}
}

type _ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (r *_ResponseWriter) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *_ResponseWriter) StatusCode() int {
	if r.statusCode == 0 {
		return 200
	}
	return r.statusCode
}

func (r *_ResponseWriter) WroteHeader() bool {
	return r.statusCode > 0
}

func (r *_ResponseWriter) WriteHeader(code int) {
	if code < 100 {
		panic(fmt.Errorf("invalid http response status code %d", code))
	}

	if r.statusCode == 0 {
		r.statusCode = code
		r.ResponseWriter.WriteHeader(code)
	}
}
