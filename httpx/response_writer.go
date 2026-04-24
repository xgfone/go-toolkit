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

	// StatusCode returns the written status code.
	//
	// Return 0 if the response header has not been written yet.
	StatusCode() int
}

type _ContextResponseWriter Context

func newContextResponseWriter(c *Context) ResponseWriter {
	return (*_ContextResponseWriter)(c)
}

func (w *_ContextResponseWriter) Unwrap() http.ResponseWriter {
	return w.w
}

func (w *_ContextResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *_ContextResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(200)
	return w.w.Write(p)
}

func (w *_ContextResponseWriter) WriteHeader(code int) {
	if code < 100 {
		panic(fmt.Errorf("invalid http response status code %d", code))
	}

	if w.Code == 0 {
		w.Code = code
		w.w.WriteHeader(code)
	}
}

func (w *_ContextResponseWriter) StatusCode() int {
	return w.Code
}

func (w *_ContextResponseWriter) JSON(code int, v any) {
	(*Context)(w).JSON(code, v)
}

func (w *_ContextResponseWriter) HTTPContext() *Context {
	return (*Context)(w)
}
