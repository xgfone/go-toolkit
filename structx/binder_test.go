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

package structx

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/mapx"
)

var errBadText = errors.New("bad text")
var errBadBind = errors.New("bad bind")

type textValue string
type bindValue string

func (t *textValue) UnmarshalText(b []byte) error {
	if string(b) == "bad" {
		return errBadText
	}

	*t = textValue("tv:" + string(b))
	return nil
}

func (b *bindValue) Bind(v any) error {
	if v == "bad" {
		return errBadBind
	}

	*b = bindValue("bind:" + fmt.Sprint(v))
	return nil
}

type bindValuesTarget struct {
	Name  string     `q:"name"`
	Age   *int       `q:"age"`
	Flag  bool       `q:"flag" default:"true"`
	Text  textValue  `q:"text"`
	PText *textValue `q:"ptext"`
}

func TestBindStringMap(t *testing.T) {
	source := map[string]string{"name": "alice", "age": "12", "text": "ok", "ptext": "pt"}

	var target bindValuesTarget
	if err := BindStringMap(&target, source, "q"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if target.Name != "alice" || target.Age == nil || *target.Age != 12 || !target.Flag ||
		target.Text != "tv:ok" || target.PText == nil || *target.PText != "tv:pt" {
		t.Fatalf("unexpected bind result: %#v", target)
	}
}

func TestBindValuesTextUnmarshaler(t *testing.T) {
	source := mapx.SMap[string]{"text": "ok", "ptext": "pt"}

	var target bindValuesTarget
	if err := BindValues(&target, source, "q"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if target.Text != "tv:ok" || target.PText == nil || *target.PText != "tv:pt" {
		t.Fatalf("unexpected bind result: %#v", target)
	}
}

func TestBindMapNested(t *testing.T) {
	type inner struct {
		Age int `json:"age"`
	}
	type targetStruct struct {
		Name  string `json:"name"`
		Inner inner  `json:"inner"`
	}

	source := map[string]any{
		"name": "alice",
		"inner": map[string]any{
			"age": 12,
		},
	}

	var target targetStruct
	if err := BindMap(&target, source, "json"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if target.Name != "alice" || target.Inner.Age != 12 {
		t.Fatalf("unexpected bind result: %#v", target)
	}
}

func TestBindMapBinder(t *testing.T) {
	type targetStruct struct {
		Value  bindValue  `json:"value"`
		PValue *bindValue `json:"pvalue"`
	}

	source := map[string]any{"value": 10, "pvalue": "ok"}

	var target targetStruct
	if err := BindMap(&target, source, "json"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if target.Value != "bind:10" || target.PValue == nil || *target.PValue != "bind:ok" {
		t.Fatalf("unexpected bind result: %#v", target)
	}
}

func TestBindMapSkipsMissingAndNilValues(t *testing.T) {
	type inner struct {
		Age int `json:"age"`
	}
	type targetStruct struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Inner inner  `json:"inner"`
	}

	target := targetStruct{Name: "keep", Email: "keep@example.com", Inner: inner{Age: 99}}
	source := map[string]any{
		"name": nil,
		"inner": map[string]any{
			"age": nil,
		},
	}

	if err := BindMap(&target, source, "json"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if target.Name != "keep" || target.Email != "keep@example.com" || target.Inner.Age != 99 {
		t.Fatalf("unexpected bind result: %#v", target)
	}
}

func TestBindMapErrors(t *testing.T) {
	var err error

	err = BindMap((*bindValuesTarget)(nil), map[string]any{}, "q")
	if err == nil || err.Error() != "dst is nil" {
		t.Fatalf("got error %v", err)
	}

	var n int
	err = BindMap(&n, map[string]any{}, "q")
	if err == nil || err.Error() != "dst is not a pointer to struct" {
		t.Fatalf("got error %v", err)
	}

	var target bindValuesTarget
	err = BindMap(&target, map[string]any{"age": "bad"}, "q")
	if err == nil || !strings.Contains(err.Error(), `"age":`) {
		t.Fatalf("got error %v", err)
	}

	type bindTarget struct {
		Value bindValue `q:"value"`
	}
	var bindTargetValue bindTarget
	err = BindMap(&bindTargetValue, map[string]any{"value": "bad"}, "q")
	if !errors.Is(err, errBadBind) || !strings.Contains(err.Error(), `"value":`) {
		t.Fatalf("got error %v", err)
	}
}

func TestBindValuesErrors(t *testing.T) {
	type _SMap = mapx.SMap[string]
	var err error

	err = BindValues((*bindValuesTarget)(nil), _SMap{}, "q")
	if err == nil || err.Error() != "dst is nil" {
		t.Fatalf("got error %v", err)
	}

	var n int
	err = BindValues(&n, _SMap{}, "q")
	if err == nil || err.Error() != "dst is not a pointer to struct" {
		t.Fatalf("got error %v", err)
	}

	var target bindValuesTarget
	err = BindValues(&target, _SMap{"age": "bad"}, "q")
	if err == nil || !strings.Contains(err.Error(), `"age":`) {
		t.Fatalf("got error %v", err)
	}

	err = BindValues(&target, _SMap{"text": "bad"}, "q")
	if err == nil || !strings.Contains(err.Error(), `"text":`) {
		t.Fatalf("got error %v", err)
	}
}
