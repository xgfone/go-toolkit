// Copyright 2024 xgfone
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
	"net/http"
	"testing"
)

func TestIsWebSocket(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderConnection, HeaderUpgrade)

	if IsWebSocket(req) {
		t.Errorf("expect false, but got true")
	}

	req.Header.Set(HeaderUpgrade, "websocket")

	if !IsWebSocket(req) {
		t.Errorf("expect true, but got false")
	}
}

func TestContentType(t *testing.T) {
	header := make(http.Header)
	if ct := ContentType(header); ct != "" {
		t.Errorf("unexpect Content-Type '%s'", ct)
	}

	header.Set("Content-Type", "application/json")
	if ct := ContentType(header); ct != "application/json" {
		t.Errorf("unexpect Content-Type '%s'", ct)
	}

	header.Set("Content-Type", "; charset=UTF-8")
	if ct := ContentType(header); ct != "" {
		t.Errorf("unexpect Content-Type '%s'", ct)
	}

	header.Set("Content-Type", "application/json; charset=UTF-8")
	if ct := ContentType(header); ct != "application/json" {
		t.Errorf("expect Content-Type '%s', but got '%s'", "application/json", ct)
	}
}

func TestCharset(t *testing.T) {
	header := make(http.Header)
	if charset := Charset(header); charset != "" {
		t.Errorf("unexpect charset '%s'", charset)
	}

	header.Set("Content-Type", "application/json")
	if charset := Charset(header); charset != "" {
		t.Errorf("unexpect charset '%s'", charset)
	}

	header.Set("Content-Type", "charset=UTF-8")
	if charset := Charset(header); charset != "UTF-8" {
		t.Errorf("expect charset '%s', but got '%s'", "UTF-8", charset)
	}

	header.Set("Content-Type", "; charset=UTF-8")
	if charset := Charset(header); charset != "UTF-8" {
		t.Errorf("expect charset '%s', but got '%s'", "UTF-8", charset)
	}

	header.Set("Content-Type", "application/json; charset=UTF-8")
	if charset := Charset(header); charset != "UTF-8" {
		t.Errorf("expect charset '%s', but got '%s'", "UTF-8", charset)
	}

	header.Set("Content-Type", "application/json; version=1; charset=UTF-8")
	if charset := Charset(header); charset != "UTF-8" {
		t.Errorf("expect charset '%s', but got '%s'", "UTF-8", charset)
	}
}

func TestAccept(t *testing.T) {
	header := make(http.Header)

	if accepts := Accept(header); accepts != nil {
		t.Errorf("expect nil, but got %v", accepts)
	}

	expects := []string{
		"text/html",
		"image/webp",
		"application/",
		"",
	}

	header.Set(HeaderAccept, "text/html, application/*;q=0.9, image/webp, no/e1;, no/e2;q=, no/e3;q=a, , */*;q=0.8")
	accepts := Accept(header)

	if len(expects) != len(accepts) {
		t.Errorf("expect %d accepts, but got %d", len(expects), len(accepts))
	} else {
		for i := range expects {
			if expects[i] != accepts[i] {
				t.Errorf("%d: expect '%s', got '%s'", i, expects[i], accepts[i])
			}
		}
	}
}

func TestAcceptEncoding(t *testing.T) {
	header := make(http.Header)

	if accepts := AcceptEncoding(header); accepts != nil {
		t.Errorf("expect nil, but got %v", accepts)
	}

	expects := []string{
		"deflate",
		"gzip",
		"",
	}

	header.Set(HeaderAcceptEncoding, "deflate, gzip;q=1.0, , *;q=0.5")
	accepts := AcceptEncoding(header)

	if len(expects) != len(accepts) {
		t.Errorf("expect %d accepts, but got %d", len(expects), len(accepts))
	} else {
		for i := range expects {
			if expects[i] != accepts[i] {
				t.Errorf("%d: expect '%s', got '%s'", i, expects[i], accepts[i])
			}
		}
	}
}

func TestAcceptLanguage(t *testing.T) {
	header := make(http.Header)

	if accepts := AcceptLanguage(header); accepts != nil {
		t.Errorf("expect nil, but got %v", accepts)
	}

	expects := []string{
		"fr-CH",
		"fr",
		"en",
		"de",
		"",
	}

	header.Set(HeaderAcceptLanguage, "fr-CH, fr;q=0.9, en;q=0.8, de;q=0.7, *;q=0.5")
	accepts := AcceptLanguage(header)

	if len(expects) != len(accepts) {
		t.Errorf("expect %d accepts, but got %d", len(expects), len(accepts))
	} else {
		for i := range expects {
			if expects[i] != accepts[i] {
				t.Errorf("%d: expect '%s', got '%s'", i, expects[i], accepts[i])
			}
		}
	}
}
