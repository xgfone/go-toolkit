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

package binder

import (
	"errors"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/mapx"
)

var errBad = errors.New("bad text")

type textValue string

func (t *textValue) UnmarshalText(b []byte) error {
	if string(b) == "bad" {
		return errBad
	}

	*t = textValue("tv:" + string(b))
	return nil
}

type bindTarget struct {
	Name  string     `q:"name"`
	Age   *int       `q:"age"`
	Flag  bool       `q:"flag" default:"true"`
	Text  textValue  `q:"text"`
	PText *textValue `q:"ptext"`
}

func TestBindSMap(t *testing.T) {
	src := map[string]string{"name": "alice", "age": "12", "text": "ok", "ptext": "pt"}

	var dst bindTarget
	if err := BindSMap(src, &dst, "q"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dst.Name != "alice" || dst.Age == nil || *dst.Age != 12 || !dst.Flag ||
		dst.Text != "tv:ok" || dst.PText == nil || *dst.PText != "tv:pt" {
		t.Fatalf("unexpected bind result: %#v", dst)
	}
}

func TestBindGetterErrors(t *testing.T) {
	type _SMap = mapx.SMap[string]
	var err error

	err = BindGetter(_SMap{}, (*bindTarget)(nil), "q")
	if err == nil || err.Error() != "dst is nil" {
		t.Fatalf("got error %v", err)
	}

	var n int
	err = BindGetter(_SMap{}, &n, "q")
	if err == nil || err.Error() != "dst is not a pointer to struct" {
		t.Fatalf("got error %v", err)
	}

	var dst bindTarget
	err = BindGetter(_SMap{"age": "bad"}, &dst, "q")
	if err == nil || !strings.Contains(err.Error(), `"age":`) {
		t.Fatalf("got error %v", err)
	}

	err = BindGetter(_SMap{"text": "bad"}, &dst, "q")
	if err == nil || !strings.Contains(err.Error(), `"text":`) {
		t.Fatalf("got error %v", err)
	}
}
