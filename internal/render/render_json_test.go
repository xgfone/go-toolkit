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

package render

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := JSON(rec, 400, nil); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s != "" {
		t.Errorf("expect response body '%s', but got '%s'", "", s)
	}

	rec = httptest.NewRecorder()
	const expectbody = `{"a":"<b>&"}`
	if err := JSON(rec, 400, map[string]string{"a": "<b>&"}); err != nil {
		t.Fatal(err)
	}

	if rec.Code != 400 {
		t.Errorf("expect status code %d, but got %d", 400, rec.Code)
	}

	if body := rec.Body.String(); body != expectbody {
		t.Errorf("expect response body '%s', but got '%s'", expectbody, body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=UTF-8" {
		t.Errorf("unexpected Content-Type: %q", ct)
	}
}

func TestJSONEncodingFailure(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Length", "1")

	if err := JSON(rec, http.StatusOK, []any{"private data", make(chan int)}); err == nil {
		t.Fatal("expected an encoding error")
	}

	const want = `{"Code":500,"Message":"Internal Server Error"}`
	if rec.Code != http.StatusInternalServerError || rec.Body.String() != want {
		t.Errorf("response = %d %q, want 500 %q", rec.Code, rec.Body.String(), want)
	}

	header := rec.Result().Header
	if ct := header.Get("Content-Type"); ct != "application/json; charset=UTF-8" {
		t.Errorf("unexpected Content-Type: %q", ct)
	}
	if length := header.Get("Content-Length"); length != "" {
		t.Errorf("stale Content-Length: %q", length)
	}
}

func TestJSONWriteFailure(t *testing.T) {
	w := &failingJSONWriter{ResponseRecorder: httptest.NewRecorder()}

	if err := JSON(w, http.StatusConflict, "conflict"); err != io.ErrClosedPipe {
		t.Errorf("write error = %v, want %v", err, io.ErrClosedPipe)
	}
	if w.Code != http.StatusConflict || w.writes != 1 {
		t.Errorf("status = %d, body writes = %d; want 409, 1", w.Code, w.writes)
	}
}

type failingJSONWriter struct {
	*httptest.ResponseRecorder
	writes int
}

func (w *failingJSONWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, io.ErrClosedPipe
}
