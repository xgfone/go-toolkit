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
	"fmt"
	"mime"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func FuzzMediaTypeHeaders(f *testing.F) {
	for _, value := range []string{
		"", "application/json", "Application/JSON; Charset=UTF-8",
		"text/plain; charset = utf-8", "text/plain; version=1; charset=utf-8",
		`text/plain; charset="utf-8"`, `text/plain; note="one;two";charset=utf-8`,
		"text/plain; charset=utf-8; CHARSET=ascii", "text/plain; charset=utf-8; charset=utf-8",
		"text/plain; charset*=UTF-8''utf-8", "text/plain; charset*0=utf-; charset*1=8",
		"text/plain; broken", "text/plain;", "text/plain;;", "text/plain; key=",
		"text/plain; charset=utf-8 \t", "\u00a0text/plain\u00a0; charset\u00a0=\u00a0utf-8",
		"text /plain", "text/plain/extra", "attachment", "charset=UTF-8", "; charset=UTF-8",
	} {
		f.Add(value)
	}

	f.Fuzz(func(t *testing.T, value string) {
		header := http.Header{HeaderContentType: {value}}
		wantType, _, _ := mime.ParseMediaType(value)
		if got := ContentType(header); got != wantType {
			t.Fatalf("ContentType(%q) = %q, mime returned %q", value, got, wantType)
		}

		// Retain the library's legacy parameters-only Charset inputs.
		legacy := strings.TrimSpace(value)
		if strings.HasPrefix(legacy, ";") {
			legacy = MIMEApplicationOctetStream + legacy
		} else if strings.HasPrefix(strings.ToLower(legacy), "charset=") {
			legacy = MIMEApplicationOctetStream + ";" + legacy
		}

		_, params, err := mime.ParseMediaType(legacy)
		wantCharset := ""
		if err == nil {
			wantCharset = params["charset"]
		}

		if got := Charset(header); got != wantCharset {
			t.Fatalf("Charset(%q) = %q, mime returned %q", value, got, wantCharset)
		}
	})
}

func FuzzSimpleMediaParameter(f *testing.F) {
	for _, value := range []string{
		"q=0.9",
		"Q = 0.8",
		"q=NaN",
		"note=hello",
		`q="0.5"`,
		"q*0=0.;q*1=5",
		"q=0.5;q=0.9",
		"note=x;q=0.8",
		"q=",
		"q=.5\u00a0",
	} {
		f.Add(value)
	}

	f.Fuzz(func(t *testing.T, parameter string) {
		key, value, ok := parseSimpleMediaParameter(parameter)
		if !ok {
			return
		}

		_, params, err := mime.ParseMediaType("value;" + parameter)
		if err != nil || len(params) != 1 || params[strings.ToLower(key)] != value {
			t.Fatalf("fast parameter parse differs from mime for %q: %q=%q, %v, %v",
				parameter, key, value, params, err)
		}
	})
}

func TestAcceptManyValues(t *testing.T) {
	var inputs, want []string
	for i := range 32 {
		value := fmt.Sprintf("application/type%d", i)
		inputs = append(inputs, value+";q=0.5")
		want = append(want, value)
	}

	inputs = append(inputs, "text/plain;q=1")
	want = append([]string{"text/plain"}, want...)
	for _, name := range []string{HeaderAccept, HeaderAcceptEncoding, HeaderAcceptLanguage} {
		header := http.Header{name: {
			strings.Join(inputs[:16], ","),
			strings.Join(inputs[16:], ","),
		}}

		var got []string
		switch name {
		case HeaderAccept:
			got = Accept(header)

		case HeaderAcceptEncoding:
			got = AcceptEncoding(header)

		case HeaderAcceptLanguage:
			got = AcceptLanguage(header)
		}

		if !slices.Equal(got, want) {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
}

func TestAcceptParameterFallback(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  []string
	}{
		{`text/plain;q="0.9",application/json;q=0.8`, []string{"text/plain", "application/json"}},
		{"text/plain;q*=UTF-8''0.9,application/json;q=0.8", []string{"text/plain", "application/json"}},
		{"text/plain;q=0.9;q=0.8,application/json", []string{"application/json"}},
		{"text/plain;q=0.8;Q=0.8,application/json;q=0.9", []string{"application/json", "text/plain"}},
		{"text/plain;level=1,application/json;q=0.9", []string{"text/plain", "application/json"}},
	} {
		if got := Accept(http.Header{HeaderAccept: {tc.value}}); !slices.Equal(got, tc.want) {
			t.Fatalf("Accept(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestSplitHeaderListEarlyStop(t *testing.T) {
	const value = `text/plain;note="one,two",application/json,image/png,text/html`
	want := []string{`text/plain;note="one,two"`, "application/json"}
	for _, limit := range []int{1, 2} {
		var got []string
		for part := range splitHeaderList(value) {
			got = append(got, part)
			if len(got) == limit {
				break
			}
		}

		if !slices.Equal(got, want[:limit]) {
			t.Errorf("stop after %d entries: got %v, want %v", limit, got, want[:limit])
		}
	}
}
