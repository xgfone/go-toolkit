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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type pathText string

func (p *pathText) UnmarshalText(data []byte) error {
	if string(data) == "bad" {
		return errInvalidBindTarget
	}

	*p = pathText(strings.ToUpper(string(data)))
	return nil
}

type pathValidatingTarget struct {
	ID    int64 `path:"id"`
	Limit int   `default:"20"`
}

func (p *pathValidatingTarget) Validate() error {
	if p.ID <= 0 || p.Limit != 20 {
		return errInvalidBindTarget
	}
	return nil
}

func ExampleBindPath() {
	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	req.SetPathValue("id", "42")

	var dst struct {
		ID int64 `path:"id"`
	}

	if err := BindPath(req, &dst); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(dst.ID)
	// Output: 42
}

func TestBindPathFromServeMux(t *testing.T) {
	called := false

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}/files/{name}", func(w http.ResponseWriter, r *http.Request) {
		called = true

		// Use a failing reader to ensure path binding never reads the body.
		r.Body = io.NopCloser(errReader{})
		r.SetPathValue("Label", "label")
		r.SetPathValue("optional", "7")
		r.SetPathValue("text", "hello")
		r.SetPathValue("ignored", "changed")

		var dst struct {
			ID       int64  `path:"id" query:"id"`
			Name     string `path:"name"`
			Label    string
			Optional *int     `path:"optional"`
			Text     pathText `path:"text"`
			Ignored  string   `path:"-"`
			Limit    int      `default:"20"`
		}

		dst.Ignored = "kept"
		if err := BindPath(r, &dst); err != nil {
			t.Fatal(err)
		}

		if dst.ID != 42 || dst.Name != "a/b" || dst.Label != "label" ||
			dst.Optional == nil || *dst.Optional != 7 || dst.Text != "HELLO" ||
			dst.Ignored != "kept" || dst.Limit != 20 {
			t.Fatalf("unexpected binding: %+v", dst)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42/files/a%2Fb?id=999", nil)
	mux.ServeHTTP(httptest.NewRecorder(), req)
	if !called {
		t.Fatal("route did not match")
	}
}

func TestBindPathMissingAndEmpty(t *testing.T) {
	for _, value := range []string{"missing", "empty"} {
		t.Run(value, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/?id=99", nil)
			if value == "empty" {
				req.SetPathValue("id", "")
			}

			dst := struct {
				ID int `path:"id"`
			}{ID: 7}
			if err := BindPath(req, &dst); err != nil {
				t.Fatal(err)
			}
			if dst.ID != 7 {
				t.Fatalf("missing/empty value changed destination: %v", dst)
			}

			var required pathValidatingTarget
			if err := BindPath(req, &required); !errors.Is(err, errInvalidBindTarget) {
				t.Fatalf("missing value bypassed validation: %v", err)
			}
		})
	}
}

func TestBindPathValidation(t *testing.T) {
	for _, value := range []string{"42", "0", "-1"} {
		var dst pathValidatingTarget
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("id", value)
		err := BindPath(req, &dst)
		if value == "42" {
			if err != nil || dst.Limit != 20 {
				t.Fatalf("defaults before validation: %+v, %v", dst, err)
			}
		} else if !errors.Is(err, errInvalidBindTarget) {
			t.Fatalf("id=%s: %v", value, err)
		}
	}
}

func TestBindPathErrors(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		if err := BindPath(nil, &struct{}{}); !errors.Is(err, errNilRequest) {
			t.Fatal(err)
		}
	})

	t.Run("invalid destination", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		var nilDst *struct{}
		if err := BindPath(req, nilDst); err == nil {
			t.Error("accepted nil destination")
		}

		var scalar int
		if err := BindPath(req, &scalar); err == nil {
			t.Error("accepted non-struct destination")
		}
	})

	for _, value := range []string{"bad", "9223372036854775808"} {
		t.Run(value, func(t *testing.T) {
			var dst struct {
				ID int64 `path:"id"`
			}

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.SetPathValue("id", value)
			err := BindPath(req, &dst)

			var numErr *strconv.NumError
			if !errors.As(err, &numErr) || !strings.Contains(err.Error(), "id") {
				t.Fatalf("lost field/parse error: %v", err)
			}
		})
	}

	t.Run("text unmarshaler", func(t *testing.T) {
		var dst struct {
			Text pathText `path:"text"`
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetPathValue("text", "bad")
		if err := BindPath(req, &dst); !errors.Is(err, errInvalidBindTarget) {
			t.Fatal(err)
		}
	})

	t.Run("invalid default", func(t *testing.T) {
		var dst struct {
			Limit int `default:"bad"`
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if err := BindPath(req, &dst); err == nil {
			t.Fatal("accepted invalid default")
		}
	})
}
