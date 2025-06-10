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

package option

import (
	"net/http"
	"testing"
)

func TestByteRange(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = ByteRange(0, 0)(req)
	if ranger := req.Header.Get("Range"); ranger != "" {
		t.Errorf("expect no Range, but got '%s'", ranger)
	}

	req = ByteRange(0, 1)(req)
	if ranger := req.Header.Get("Range"); ranger != "bytes=0-0" {
		t.Errorf("expect Range '%s', but got '%s'", "bytes=0-0", ranger)
	}
}

func TestAuthBearer(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expect a panic, but got nil")
			}
		}()
		AuthBearer("  ")
	}()

	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	req = Apply(req, AuthBearer(" apikey "))
	if value := req.Header.Get("Authorization"); value != "Bearer apikey" {
		t.Errorf("expect '%s', but got '%s'", "Bearer apikey", value)
	}
}
