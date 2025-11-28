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
