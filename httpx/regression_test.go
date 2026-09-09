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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestRegressionInformationalResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := AcquireContext()
		defer ReleaseContext(c)

		c.Reset(w, r)
		c.WriteHeader(103)
		if c.StatusCode() != 0 {
			t.Errorf("informational response committed final status %d", c.StatusCode())
		}

		c.WriteHeader(201)
		_, _ = c.ResponseWriter.Write([]byte("created"))
	}))

	defer server.Close()
	r, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	defer r.Body.Close() //nolint:errcheck
	if r.StatusCode != 201 {
		t.Fatalf("103 followed by 201 became %d", r.StatusCode)
	}
}

type regressionRoundTripper func(*http.Request) (*http.Response, error)

func (f regressionRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestRegressionRequestBodyLifetime(t *testing.T) {
	old := GetClient()
	defer SetClient(old)

	release := make(chan struct{})
	read := make(chan string, 1)
	SetClient(&http.Client{Transport: regressionRoundTripper(func(r *http.Request) (*http.Response, error) {
		go func() {
			<-release

			b, err := io.ReadAll(r.Body)
			_ = r.Body.Close()

			if err != nil {
				t.Error(err)
			}

			read <- string(b)
		}()
		return nil, fmt.Errorf("early transport error")
	})})

	_ = Post(context.Background(), "http://audit.invalid", nil, map[string]string{"key": "value"})
	close(release)
	if got := <-read; !strings.Contains(got, `"key":"value"`) {
		t.Fatalf("request body reset before asynchronous Body.Close: %q", got)
	}
}

func TestRequestBodyReplayAfterReturn(t *testing.T) {
	old := GetClient()
	defer SetClient(old)

	var request *http.Request
	SetClient(&http.Client{Transport: regressionRoundTripper(func(r *http.Request) (*http.Response, error) {
		request = r
		_ = r.Body.Close()
		return nil, errors.New("transport failed")
	})})

	_ = Post(context.Background(), "http://example.invalid", nil, map[string]int{"value": 42})
	if request.GetBody == nil {
		t.Fatal("encoded request must support body replay")
	}

	for range 2 {
		body, err := request.GetBody()
		if err != nil {
			t.Fatal(err)
		}

		data, err := io.ReadAll(body)
		_ = body.Close()
		if err != nil || string(data) != "{\"value\":42}\n" {
			t.Fatalf("unexpected replay: %q, %v", data, err)
		}
	}
}

func TestAcceptQuotedParametersAndMultipleLines(t *testing.T) {
	h := http.Header{}
	h.Add("Accept", `text/html;note="one,two;three";q=0.9,application/json;q=0.8`)
	h.Add("Accept", `text/plain;q=1, image/png;q=NaN, image/jpeg;q=0, image/webp;q=1.1`)
	want := []string{"text/plain", "text/html", "application/json"}

	if got := Accept(h); !slices.Equal(got, want) {
		t.Fatalf("Accept = %v, want %v", got, want)
	}

	h.Set("Connection", "keep-alive")
	h.Add("Connection", " Upgrade ")
	h.Set("Upgrade", "WebSocket")
	r := httptest.NewRequest("GET", "/", nil)
	r.Header = h

	if !IsWebSocket(r) {
		t.Fatal("Upgrade token on the second header line was not recognized")
	}
}

func TestRegressionWebSocketConnectionTokens(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Connection", "keep-alive, Upgrade")
	r.Header.Set("Upgrade", "websocket")

	if !IsWebSocket(r) {
		t.Fatal("valid Upgrade token in Connection list is missed")
	}
}

func TestRegressionCharsetExtraParameter(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Type", `text/plain; charset="utf-8"; format=flowed`)

	if got := Charset(h); got != "utf-8" {
		t.Fatalf("charset includes quotes and following parameter: %q", got)
	}
}

func TestRegressionAcceptMediaParameter(t *testing.T) {
	h := http.Header{}
	h.Set("Accept", "text/html;level=1;q=0.9, application/json;q=0.5")

	got := Accept(h)
	if len(got) != 2 || got[0] != "text/html" {
		t.Fatalf("valid media parameters discarded the preferred type: %v", got)
	}
}

func TestRegressionContentTypeCase(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"Value":1}`))
	r.Header.Set("Content-Type", "Application/JSON")

	var dst struct{ Value int }
	if err := BindBody(r, &dst); err != nil {
		t.Fatalf("valid case-insensitive media type rejected: %v", err)
	}
}
