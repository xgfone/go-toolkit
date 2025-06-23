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
	"io"
	"net/http"
	"testing"
)

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

	c = WrapClient(c, func(c Client, r *http.Request) (*http.Response, error) { return c.Do(r) })
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
	SetClient(WrapClient(DoFunc(do), func(c Client, r *http.Request) (*http.Response, error) { return c.Do(r) }))

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
