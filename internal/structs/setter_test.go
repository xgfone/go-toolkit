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
	"errors"
	"reflect"
	"testing"
)

var errBadText = errors.New("bad text")

type textValue string

type unsupported complex64

func (t *textValue) UnmarshalText(b []byte) error {
	if string(b) == "bad" {
		return errBadText
	}

	*t = textValue("tv:" + string(b))
	return nil
}

func TestUnmarshalText(t *testing.T) {
	var v textValue
	var err error

	err = unmarshalText(reflect.ValueOf(&v), "ok")
	if err != nil || v != "tv:ok" {
		t.Fatalf("unexpected result: %v %q", err, v)
	}

	err = unmarshalText(reflect.ValueOf(&v), "bad")
	if !errors.Is(err, errBadText) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileSetterValueInterface(t *testing.T) {
	setter := CompileSetter(reflect.TypeFor[textValue]())

	var v textValue
	var err error

	err = setter(reflect.TypeFor[textValue](), reflect.ValueOf(&v).Elem(), "ok")
	if err != nil || v != "tv:ok" {
		t.Fatalf("unexpected result: %v %q", err, v)
	}

	err = setter(reflect.TypeFor[textValue](), reflect.ValueOf(&v).Elem(), "bad")
	if !errors.Is(err, errBadText) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileSetterPointerInterface(t *testing.T) {
	setter := CompileSetter(reflect.TypeFor[*textValue]())

	var v *textValue
	var err error

	err = setter(reflect.TypeFor[*textValue](), reflect.ValueOf(&v).Elem(), "ok")
	if err != nil || v == nil || *v != "tv:ok" {
		t.Fatalf("unexpected result: %v %#v", err, v)
	}

	err = setter(reflect.TypeFor[*textValue](), reflect.ValueOf(&v).Elem(), "bad")
	if !errors.Is(err, errBadText) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileSetterPointerValue(t *testing.T) {
	setter := CompileSetter(reflect.TypeFor[*int]())

	var v *int
	var err error

	err = setter(reflect.TypeFor[*int](), reflect.ValueOf(&v).Elem(), "11")
	if err != nil || v == nil || *v != 11 {
		t.Fatalf("unexpected result: %v %#v", err, v)
	}

	err = setter(reflect.TypeFor[*int](), reflect.ValueOf(&v).Elem(), "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCompileSetterValueKinds(t *testing.T) {
	sv := new(string)
	if err := testCompileSetter(sv, "s"); err != nil || *sv != "s" {
		t.Fatalf("unexpected string result: %v %q", err, *sv)
	}

	bv := new(bool)
	if err := testCompileSetter(bv, "true"); err != nil || !*bv {
		t.Fatalf("unexpected bool result: %v %v", err, *bv)
	}
	if err := testCompileSetter(new(bool), "bad"); err == nil {
		t.Fatal("expected bool error")
	}

	iv := new(int)
	if err := testCompileSetter(iv, "3"); err != nil || *iv != 3 {
		t.Fatalf("unexpected int result: %v %v", err, *iv)
	}
	if err := testCompileSetter(new(int), "bad"); err == nil {
		t.Fatal("expected int error")
	}

	uv := new(uint)
	if err := testCompileSetter(uv, "5"); err != nil || *uv != 5 {
		t.Fatalf("unexpected uint result: %v %v", err, *uv)
	}
	if err := testCompileSetter(new(uint), "bad"); err == nil {
		t.Fatal("expected uint error")
	}

	fv := new(float64)
	if err := testCompileSetter(fv, "1.5"); err != nil || *fv != 1.5 {
		t.Fatalf("unexpected float result: %v %v", err, *fv)
	}
	if err := testCompileSetter(new(float64), "bad"); err == nil {
		t.Fatal("expected float error")
	}
}

func TestCompileSetterUnsupported(t *testing.T) {
	err := testCompileSetter(new(unsupported), "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

func testCompileSetter[T any](ptr *T, tag string) error {
	rtype := reflect.TypeFor[T]()
	return CompileSetter(rtype)(rtype, reflect.ValueOf(ptr).Elem(), tag)
}
