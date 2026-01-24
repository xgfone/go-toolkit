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
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	Handler201.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Errorf("expect status code %d, but got %d", 201, rec.Code)
	}

	rec = httptest.NewRecorder()
	Handler404.ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Errorf("expect status code %d, but got %d", 404, rec.Code)
	}
}

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := JSON(rec, 400, nil); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s != "" {
		t.Errorf("expect response body '%s', but got '%s'", "", s)
	}

	rec = httptest.NewRecorder()
	if err := JSON(rec, 400, map[string]string{"a": "b"}); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s != `{"a":"b"}`+"\n" {
		t.Errorf("expect response body '%s', but got '%s'", `{"a":"b"}`, s)
	}

	if rec.Code != 400 {
		t.Errorf("expect status code %d, but got %d", 400, rec.Code)
	}

	expectbody := `{"a":"b"}` + "\n"
	if body := rec.Body.String(); body != expectbody {
		t.Errorf("expect response body '%s', but got '%s'", expectbody, body)
	}
}

func TestXML(t *testing.T) {
	var req struct {
		XMLName xml.Name `xml:"outer"`
		A       string   `xml:"a"`
	}
	req.A = "b"

	rec := httptest.NewRecorder()
	if err := XML(rec, 400, nil); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s != "" {
		t.Errorf("expect response body '%s', but got '%s'", "", s)
	}

	rec = httptest.NewRecorder()
	if err := XML(rec, 400, req); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s == "" {
		t.Error("unexpected empty response body")
	}

	if rec.Code != 400 {
		t.Errorf("expect status code %d, but got %d", 400, rec.Code)
	}

	expectbody := `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<outer><a>b</a></outer>`
	if body := rec.Body.String(); body != expectbody {
		t.Errorf("expect response body '%s', but got '%s'", expectbody, body)
	}
}

func TestWriter(t *testing.T) {
	err := write(&mockResponseWriter{}, bytes.NewBufferString("abc"))
	if err != io.ErrShortWrite {
		t.Errorf("expect an io.ErrShortWrite, but got %v", err)
	}
}

type mockResponseWriter struct{}

func (m *mockResponseWriter) WriteHeader(statusCode int)  {}
func (m *mockResponseWriter) Write(p []byte) (int, error) { return 0, nil }
func (m *mockResponseWriter) Header() http.Header         { return nil }
