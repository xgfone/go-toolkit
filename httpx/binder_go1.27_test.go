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

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type contextBindTarget struct {
	Value string `json:"value" query:"value" header:"X-Value"`
}

func TestContextBind(t *testing.T) {
	t.Run("body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":"body"}`))
		req.Header.Set(HeaderContentType, MIMEApplicationJSON)

		var dst contextBindTarget
		if err := (&Context{Request: req}).BindBody(&dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dst.Value != "body" {
			t.Fatalf("got value %q", dst.Value)
		}
	})

	t.Run("query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?value=query", nil)

		var dst contextBindTarget
		if err := (&Context{Request: req}).BindQuery(&dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dst.Value != "query" {
			t.Fatalf("got value %q", dst.Value)
		}
	})

	t.Run("header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Value", "header")

		var dst contextBindTarget
		if err := (&Context{Request: req}).BindHeader(&dst); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dst.Value != "header" {
			t.Fatalf("got value %q", dst.Value)
		}
	})
}
