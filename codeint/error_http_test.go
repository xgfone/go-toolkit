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

package codeint

import (
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestErrorStatusCode(t *testing.T) {
	err := NewError(400).WithStatus(0)
	if code := err.StatusCode(); code != 400 {
		t.Errorf("expect status code %d, but got %d", 400, code)
	}

	err = err.WithCode(400400)
	if code := err.StatusCode(); code != 500 {
		t.Errorf("expect status code %d, but got %d", 500, code)
	}

	err = err.WithStatus(501)
	if code := err.StatusCode(); code != 501 {
		t.Errorf("expect status code %d, but got %d", 501, code)
	}

}

func TestErrorServeHTTP(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	NewError(400).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("expect status code %d, but got %d", 400, rec.Code)
	}

	const body = `{"Code":400,"Message":"Bad Request"}`
	if s := strings.TrimSpace(rec.Body.String()); s != body {
		t.Errorf("expect response body '%s', but got '%s'", body, s)
	}
}

func TestErrorStatusCodeBoundaries(t *testing.T) {
	for _, tc := range []struct{ status, want int }{
		{-1, 500},
		{99, 500},
		{100, 100},
		{599, 599},
		{600, 500},
		{1000, 500},
		{0, 404}, // Only an unspecified status falls back to the business code.
	} {
		t.Run(strconv.Itoa(tc.status), func(t *testing.T) {
			// Status is public; callers can bypass the normalization in WithStatus.
			err := Error{Code: 404, Status: tc.status}
			if got := err.StatusCode(); got != tc.want {
				t.Fatalf("StatusCode = %d, want %d", got, tc.want)
			}

			if tc.want == 500 {
				rec := httptest.NewRecorder()
				err.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
				if rec.Code != 500 {
					t.Errorf("ServeHTTP status = %d, want 500", rec.Code)
				}
			}
		})
	}
}
