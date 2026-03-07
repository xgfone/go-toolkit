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
	"errors"
	"net/http"
	"testing"
)

func TestClientError(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	rsp := &http.Response{StatusCode: http.StatusBadRequest}

	t.Run("Basic", func(t *testing.T) {
		err := newClientError(req, rsp)
		if err.Request() != req {
			t.Error("Request mismatch")
		}
		if err.Response() != rsp {
			t.Error("Response mismatch")
		}
		if err.StatusCode() != http.StatusBadRequest {
			t.Errorf("StatusCode: got %d, want %d", err.StatusCode(), http.StatusBadRequest)
		}
		if err.ResponseBody() != "" {
			t.Errorf("ResponseBody: got %q, want empty", err.ResponseBody())
		}
		if err.Unwrap() != nil {
			t.Error("Unwrap should be nil")
		}
	})

	t.Run("WithBody", func(t *testing.T) {
		err := newClientError(req, rsp).WithBody([]byte("error body"))
		if err.ResponseBody() != "error body" {
			t.Errorf("ResponseBody: got %q, want %q", err.ResponseBody(), "error body")
		}
	})

	t.Run("WithError", func(t *testing.T) {
		wrapped := errors.New("wrapped error")
		err := newClientError(req, rsp).WithError(wrapped)
		if err.Unwrap() != wrapped {
			t.Error("Unwrap mismatch")
		}
	})

	t.Run("ErrorString", func(t *testing.T) {
		tests := []struct {
			name     string
			err      _ClientError
			expected string
		}{
			{
				name:     "only status code",
				err:      newClientError(req, &http.Response{StatusCode: 404}),
				expected: "statuscode=404",
			},
			{
				name:     "with body",
				err:      newClientError(req, &http.Response{StatusCode: 400}).WithBody([]byte("bad request")),
				expected: "statuscode=400, body=bad request",
			},
			{
				name:     "with error",
				err:      newClientError(req, &http.Response{StatusCode: 500}).WithError(errors.New("server error")),
				expected: "statuscode=500, err=server error",
			},
			{
				name: "with body and error",
				err: newClientError(req, &http.Response{StatusCode: 503}).
					WithBody([]byte("service unavailable")).
					WithError(errors.New("timeout")),
				expected: "statuscode=503, body=service unavailable, err=timeout",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.err.Error(); got != tt.expected {
					t.Errorf("Error(): got %q, want %q", got, tt.expected)
				}
			})
		}
	})

	t.Run("Immutability", func(t *testing.T) {
		base := newClientError(req, &http.Response{StatusCode: 400})

		// Test WithBody doesn't mutate original
		withBody := base.WithBody([]byte("body"))
		if base.ResponseBody() != "" {
			t.Error("WithBody mutated base")
		}
		if withBody.ResponseBody() != "body" {
			t.Error("WithBody didn't set body")
		}

		// Test WithError doesn't mutate original
		wrapped := errors.New("error")
		withError := base.WithError(wrapped)
		if base.Unwrap() != nil {
			t.Error("WithError mutated base")
		}
		if withError.Unwrap() != wrapped {
			t.Error("WithError didn't set error")
		}

		// Test chaining
		chained := base.WithBody([]byte("chained")).WithError(wrapped)
		if chained.ResponseBody() != "chained" || chained.Unwrap() != wrapped {
			t.Error("Chaining failed")
		}
		if base.ResponseBody() != "" || base.Unwrap() != nil {
			t.Error("Chaining mutated base")
		}
	})
}
