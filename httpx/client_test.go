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
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
)

func authBearerOption(token string) Option {
	return func(r *http.Request) *http.Request {
		r.Header.Set("Authorization", "Bearer "+token)
		return r
	}
}

func byteRangeOption(start, length uint64) Option {
	end := strconv.FormatUint(start+length-1, 10)
	return func(r *http.Request) *http.Request {
		r.Header.Set("Range", fmt.Sprintf("bytes=%d-%s", start, end))
		return r
	}
}

func TestUnwrapClient(t *testing.T) {
	if UnwrapClient(http.DefaultClient) != nil {
		t.Errorf("expect a nil, but got a http client")
	}

	c := WrapClient(http.DefaultClient, func(c Client, r *http.Request) (*http.Response, error) {
		return c.Do(r)
	})

	client := UnwrapClient(c)
	if client == nil {
		t.Fatal("expect a client, but got nil")
	} else if hc, ok := client.(*http.Client); !ok {
		t.Errorf("expect a *http.Client, but got %T", client)
	} else if hc != http.DefaultClient {
		t.Errorf("expect http.DefaultClient, but got other")
	}

	c = WrapClientWithOptions(c, authBearerOption("token"))
	for {
		if _c := UnwrapClient(c); _c != nil {
			c = _c
		} else {
			break
		}
	}
	if hc, ok := c.(*http.Client); !ok {
		t.Errorf("expect a *http.Client, but got %T", client)
	} else if hc != http.DefaultClient {
		t.Errorf("expect http.DefaultClient, but got other")
	}
}

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
		return &http.Response{StatusCode: 201, Body: io.NopCloser(nil)}, nil
	}
	SetClient(WrapClientWithOptions(DoFunc(do), byteRangeOption(0, 1)))

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
		return &http.Response{StatusCode: 201, Body: io.NopCloser(nil)}, nil
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
		return &http.Response{StatusCode: 201, Body: io.NopCloser(nil)}, nil
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
