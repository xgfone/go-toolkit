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

package httpx

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/xgfone/go-toolkit/httpx/option"
)

func TestClient(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expect a panic, but got nil")
			}
		}()
		SetClient(nil)
	}()

	do := func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 201}, nil
	}
	SetClient(WrapClientWithOptions(DoFunc(do), option.ByteRange(0, 1)))

	req, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := GetClient().Do(req)
	if err != nil {
		t.Error(err)
	} else if resp.StatusCode != 201 {
		t.Errorf("expected status code 201, got %d", resp.StatusCode)
	}
}

func TestRequest(t *testing.T) {
	SetClient(DoFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/error" {
			return nil, errors.New("error")
		}
		return &http.Response{StatusCode: 201}, nil
	}))

	err := Get(context.Background(), ":", nil)
	if err == nil {
		t.Error("expect an error, but got nil")
	}

	err = Get(context.Background(), "http://localhost/error", nil)
	if err == nil {
		t.Error("expect an error, but got nil")
	} else if s := err.Error(); s != "error" {
		t.Errorf("expect error '%s', but got '%s'", "error", s)
	}

	var code int
	okdo := func(r *http.Response) error {
		code = r.StatusCode
		return nil
	}

	err = Get(context.Background(), "http://localhost/ok", okdo)
	if err != nil {
		t.Error(err)
	} else if code != 201 {
		t.Errorf("expect status code %d, but got %d", 201, code)
	}
}

func TestPost(t *testing.T) {
	SetClient(DoFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 201}, nil
	}))

	var code int
	err := Post(context.Background(), "http://localhost", nil, func(r *http.Response) error {
		code = r.StatusCode
		return nil
	})

	if err != nil {
		t.Error(err)
	} else if code != 201 {
		t.Errorf("expect status code %d, but got %d", 201, code)
	}
}
