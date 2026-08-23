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

//go:build go1.27

package httpx

// BindBody binds c.Request's body into dst according to the request
// Content-Type, then sets defaults and validates dst.
//
// It is a convenience wrapper for BindBody(c.Request, dst).
func (c *Context) BindBody[T any](dst *T) error {
	return BindBody(c.Request, dst)
}

// BindQuery binds c.Request's query parameters into dst using the "query"
// struct tag, then sets defaults and validates dst.
//
// It is a convenience wrapper for BindQuery(c.Request, dst).
func (c *Context) BindQuery[T any](dst *T) error {
	return BindQuery(c.Request, dst)
}

// BindHeader binds c.Request's headers into dst using the "header" struct tag,
// then sets defaults and validates dst.
//
// It is a convenience wrapper for BindHeader(c.Request, dst).
func (c *Context) BindHeader[T any](dst *T) error {
	return BindHeader(c.Request, dst)
}
