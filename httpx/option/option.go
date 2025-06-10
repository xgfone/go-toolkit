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

// Package option provides some request configuration options.
package option

import (
	"fmt"
	"net/http"
	"strings"
)

func noop(r *http.Request) *http.Request { return r }

// Option is used to configure the http request.
type Option func(*http.Request) *http.Request

// Apply applies the options to r and returns the new one.
func Apply(r *http.Request, options ...Option) *http.Request {
	for i := range options {
		r = options[i](r)
	}
	return r
}

// ByteRange returns a request option to add the http request header "Range".
//
// If length is equal to 0, return a noop option that does nothing.
func ByteRange(start, length uint64) Option {
	if length == 0 {
		return noop
	}

	end := start + length - 1
	return func(r *http.Request) *http.Request {
		r.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
		return r
	}
}

// AuthBearer returns a request option to add the Bearer Authorization header.
func AuthBearer(token string) Option {
	if token = strings.TrimSpace(token); token == "" {
		panic("option.AuthBearer: token must not be empty")
	}

	value := "Bearer " + token
	return func(r *http.Request) *http.Request {
		r.Header.Set("Authorization", value)
		return r
	}
}
