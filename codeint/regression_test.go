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

package codeint

import (
	"net/http/httptest"
	"testing"
)

func TestRegressionLowBusinessCode(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewError(42).ServeHTTP panics: %v", r)
		}
	}()

	e := NewError(42)
	if s := e.StatusCode(); s != 500 {
		t.Errorf("expected safe HTTP status 500 for business code 42, got %d", s)
	}
	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}
