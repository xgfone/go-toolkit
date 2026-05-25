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
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/codeint"
)

var errInvalidBindTarget = errors.New("invalid bind target")
var errReadForm = errors.New("read form")

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errReadForm }

type bindValidatingBody struct {
	Name string `json:"name" xml:"name" form:"name" default:"anonymous"`
	Age  int    `json:"age" xml:"age" form:"age"`
}

func (b *bindValidatingBody) Validate() error {
	if b.Age < 0 {
		return errInvalidBindTarget
	}
	return nil
}

func newBindRequest(method, target, contentType string, body any) *http.Request {
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	case io.Reader:
		reader = b
	default:
		panic("unsupported request body")
	}

	req := httptest.NewRequest(method, target, reader)
	if contentType != "" {
		req.Header.Set(HeaderContentType, contentType)
	}
	return req
}

func newMultipartBody(t *testing.T, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}

func assertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("got error %v", err)
	}
}

func assertHasError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBindBody(t *testing.T) {
	form := url.Values{"name": {"alice"}, "age": {"10"}}
	multipartBody, multipartContentType := newMultipartBody(t, map[string]string{"name": "alice", "age": "11"})

	tests := []struct {
		name     string
		req      *http.Request
		dst      any
		wantName string
		wantAge  int
	}{
		{
			name:     "json",
			req:      newBindRequest(http.MethodPost, "/", MIMEApplicationJSONCharsetUTF8, `{"age":12}`),
			dst:      new(bindValidatingBody),
			wantName: "anonymous",
			wantAge:  12,
		},
		{
			name:     "xml",
			req:      newBindRequest(http.MethodPost, "/", MIMEApplicationXML, `<request><name>alice</name><age>9</age></request>`),
			dst:      new(bindValidatingBody),
			wantName: "alice",
			wantAge:  9,
		},
		{
			name:     "form",
			req:      newBindRequest(http.MethodPost, "/?age=99", MIMEApplicationForm, form.Encode()),
			dst:      new(bindValidatingBody),
			wantName: "alice",
			wantAge:  10,
		},
		{
			name:     "multipart form",
			req:      newBindRequest(http.MethodPost, "/", multipartContentType, multipartBody),
			dst:      new(bindValidatingBody),
			wantName: "alice",
			wantAge:  11,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := BindBody(tt.req, tt.dst); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := tt.dst.(*bindValidatingBody)
			if got.Name != tt.wantName || got.Age != tt.wantAge {
				t.Fatalf("unexpected bind result: %#v", got)
			}
		})
	}
}

func TestBindBodyValidate(t *testing.T) {
	var dst bindValidatingBody
	req := newBindRequest(http.MethodPost, "/", MIMEApplicationJSON, `{"age":-1}`)
	assertErrorIs(t, BindBody(req, &dst), errInvalidBindTarget)
}

func TestBindBodyErrors(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		assertErrorIs(t, BindBody(nil, &bindValidatingBody{}), errNilRequest)
	})

	t.Run("missing content type", func(t *testing.T) {
		req := newBindRequest(http.MethodPost, "/", "", nil)
		assertErrorIs(t, BindBody(req, &bindValidatingBody{}), codeint.ErrMissingContentType)
	})

	t.Run("unsupported content type", func(t *testing.T) {
		req := newBindRequest(http.MethodPost, "/", MIMETextPlain, "x")
		assertErrorIs(t, BindBody(req, &bindValidatingBody{}), codeint.ErrUnsupportedMediaType)
	})

	t.Run("invalid json", func(t *testing.T) {
		req := newBindRequest(http.MethodPost, "/", MIMEApplicationJSON, `{"age":`)
		assertHasError(t, BindBody(req, &bindValidatingBody{}))
	})

	t.Run("invalid xml", func(t *testing.T) {
		req := newBindRequest(http.MethodPost, "/", MIMEApplicationXML, `<request>`)
		assertHasError(t, BindBody(req, &bindValidatingBody{}))
	})

	t.Run("parse form", func(t *testing.T) {
		req := newBindRequest(http.MethodPost, "/", MIMEApplicationForm, io.NopCloser(errReader{}))
		assertErrorIs(t, BindBody(req, &bindValidatingBody{}), errReadForm)
	})

	t.Run("bind form", func(t *testing.T) {
		form := url.Values{"age": {"bad"}}
		req := newBindRequest(http.MethodPost, "/", MIMEApplicationForm, form.Encode())
		assertHasError(t, BindBody(req, &bindValidatingBody{}))
	})

	t.Run("parse multipart form", func(t *testing.T) {
		req := newBindRequest(http.MethodPost, "/", MIMEMultipartForm, "bad multipart")
		assertHasError(t, BindBody(req, &bindValidatingBody{}))
	})

	t.Run("bind multipart form", func(t *testing.T) {
		body, contentType := newMultipartBody(t, map[string]string{"age": "bad"})
		req := newBindRequest(http.MethodPost, "/", contentType, body)
		assertHasError(t, BindBody(req, &bindValidatingBody{}))
	})

	t.Run("set default", func(t *testing.T) {
		var dst int
		req := newBindRequest(http.MethodPost, "/", MIMEApplicationJSON, `1`)
		err := BindBody(req, &dst)
		if err == nil || err.Error() != "SetDefault: structptr is not a pointer to struct" {
			t.Fatalf("got error %v", err)
		}
	})
}

func TestBindHeader(t *testing.T) {
	type headerTarget struct {
		RequestID string `header:"X-Request-Id"`
		TraceID   string `header:"X-Trace-Id" default:"missing"`
	}

	var dst headerTarget

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderXRequestID, "rid")
	if err := BindHeader(req, &dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.RequestID != "rid" || dst.TraceID != "missing" {
		t.Fatalf("unexpected bind result: %#v", dst)
	}
}

func TestBindHeaderErrors(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		assertErrorIs(t, BindHeader(nil, &struct{}{}), errNilRequest)
	})

	t.Run("bind header", func(t *testing.T) {
		type headerTarget struct {
			Age int `header:"X-Age"`
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Age", "bad")
		assertHasError(t, BindHeader(req, &headerTarget{}))
	})
}

func TestBindQuery(t *testing.T) {
	type queryTarget struct {
		Name  string `query:"name"`
		Age   int    `query:"age"`
		Limit int    `query:"limit" default:"20"`
	}

	var dst queryTarget

	req := httptest.NewRequest(http.MethodGet, "/?name=alice&age=12", nil)
	if err := BindQuery(req, &dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dst.Name != "alice" || dst.Age != 12 || dst.Limit != 20 {
		t.Fatalf("unexpected bind result: %#v", dst)
	}
}

func TestBindQueryErrors(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		assertErrorIs(t, BindQuery(nil, &struct{}{}), errNilRequest)
	})

	t.Run("bind query", func(t *testing.T) {
		type queryTarget struct {
			Age int `query:"age"`
		}

		req := httptest.NewRequest(http.MethodGet, "/?age=bad", nil)
		assertHasError(t, BindQuery(req, &queryTarget{}))
	})
}
