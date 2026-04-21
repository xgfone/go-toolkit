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

package structs

import (
	"reflect"
	"testing"
)

type Embed struct {
	E string `q:"e"`
}

type sample struct {
	hidden string
	Skip   string `q:"-"`
	Plain  string `q:"plain"`
	NoTag  uint
	Def    int        `q:"def" default:"7"`
	Ptr    *int       `q:"ptr"`
	Text   textValue  `q:"text"`
	PText  *textValue `q:"ptext"`

	Embed
}

type nested struct{ N int }
type holder struct{ P *nested }
type badHolder struct{ P *int }

func TestParseCacheAndSetValue(t *testing.T) {
	typ := reflect.TypeFor[sample]()
	s1 := Parse(typ, "q")
	s2 := Parse(typ, "q")

	if s1 != s2 {
		t.Fatal("cache miss")
	}
	if len(s1.Fields) != 7 {
		t.Fatalf("unexpected field count: %d", len(s1.Fields))
	}

	fields := map[string]Field{}
	for _, f := range s1.Fields {
		fields[f.Name] = f
	}

	var dst sample
	root := reflect.ValueOf(&dst).Elem()
	inputs := map[string]string{
		"e":     "x",
		"plain": "y",
		"def":   "8",
		"NoTag": "10",
		"ptr":   "9",
		"text":  "ok",
		"ptext": "pt",
	}
	for name, val := range inputs {
		if err := fields[name].SetValue(root, val); err != nil {
			t.Fatalf("set %s: %v", name, err)
		}
	}

	if dst.E != "x" || dst.Plain != "y" || dst.NoTag != 10 || dst.Def != 8 ||
		dst.Ptr == nil || *dst.Ptr != 9 || dst.Text != "tv:ok" ||
		dst.PText == nil || *dst.PText != "tv:pt" {
		t.Fatalf("unexpected result: %#v", dst)
	}

	if fields["def"].Default != "7" {
		t.Fatalf("unexpected default: %q", fields["def"].Default)
	}
}

func TestParseHelpers(t *testing.T) {
	if parseTagName("") != "" || parseTagName("name,omitempty") != "name" {
		t.Fatal("unexpected tag parse")
	}

	idx := appendIndex([]int{1, 2}, 3)
	if len(idx) != 3 || idx[2] != 3 {
		t.Fatalf("unexpected index: %#v", idx)
	}
}

func TestFieldByIndexAllocErrors(t *testing.T) {
	var err error

	_, err = fieldByIndexAlloc(reflect.ValueOf(sample{}), nil)
	if err == nil || err.Error() != "empty field index" {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = fieldByIndexAlloc(reflect.ValueOf(sample{}), []int{0, 0})
	if err == nil || err.Error() != "invalid field path" {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = fieldByIndexAlloc(reflect.ValueOf(badHolder{}), []int{0, 0})
	if err == nil || err.Error() != "non-struct pointer in field path" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFieldByIndexAllocPointerStruct(t *testing.T) {
	h := holder{}
	v, err := fieldByIndexAlloc(reflect.ValueOf(&h).Elem(), []int{0, 0})
	if err != nil || !v.IsValid() || h.P == nil {
		t.Fatalf("unexpected result: %v %v %#v", v, err, h)
	}
}

func TestMakeValueSetterError(t *testing.T) {
	err := makeValueSetter([]int{0, 0}, reflect.TypeFor[int]())(reflect.ValueOf(sample{}), "1")
	if err == nil || err.Error() != "invalid field path" {
		t.Fatalf("unexpected error: %v", err)
	}
}
