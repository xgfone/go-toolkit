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
	"time"
)

type embedMe struct {
	A int    `q:"a"`
	B string `q:"b"`
}

type WrapTime time.Time

//nolint:unused
type noExported struct {
	x int
	y string
}

type unexportedEmb struct {
	X string `q:"x"`
}

type embedPointer struct {
	*embedMe
	P *WrapTime `q:"p"`
}

type embedNamed struct {
	embedMe
	WrapTime
	C int `q:"c"`
}

type embedExternal struct {
	time.Time
	C int `q:"c"`
}

type embedHidden struct {
	noExported //nolint:unused
	unexportedEmb
	A int `q:"a"`
}

type flatFields struct {
	Name  string `q:"name"`
	Value int    `q:"value"`
}

// --- Helper ---

func mustSet(t *testing.T, f Field, root reflect.Value, s string, flag int8) {
	t.Helper()
	if err := f.SetValue(root, s, flag); err != nil {
		t.Fatalf("SetValue(%q, %d): %v", s, flag, err)
	}
}

func checkFields(t *testing.T, s *Struct, want ...string) {
	t.Helper()
	if len(s.Fields) != len(want) {
		t.Fatalf("got %d fields, want %d: %v", len(s.Fields), len(want), fieldNames(s.Fields))
	}
	m := make(map[string]bool, len(want))
	for _, n := range want {
		m[n] = true
	}
	for _, f := range s.Fields {
		if !m[f.Name] {
			t.Fatalf("unexpected field %q in %v", f.Name, fieldNames(s.Fields))
		}
		delete(m, f.Name)
	}
	if len(m) > 0 {
		t.Fatalf("missing fields: %v", m)
	}
}

func fieldNames(fields []Field) []string {
	ns := make([]string, len(fields))
	for i, f := range fields {
		ns[i] = f.Name
	}
	return ns
}

// --- Requirements tests ---

// Named struct embedded anonymously from the same package — should expand.
func TestExpandNamedStruct(t *testing.T) {
	s := Parse(reflect.TypeFor[embedNamed](), "q")
	checkFields(t, s, "a", "b", "c", "WrapTime")
}

// Pointer to named struct embedded anonymously — should dereference and expand.
func TestExpandPointerEmbed(t *testing.T) {
	s := Parse(reflect.TypeFor[embedPointer](), "q")
	checkFields(t, s, "a", "b", "p")
}

// Literal named field with anonymous struct type (not anonymous embed) — not expanded.
type embedLiteral struct {
	_ struct {
		X int
		Y string
	} `q:"-"`
	Z int `q:"z"`
}

func TestExpandAnonymousStruct(t *testing.T) {
	s := Parse(reflect.TypeFor[embedLiteral](), "q")
	checkFields(t, s, "z")
}

// External struct (time.Time) embedded anonymously — NOT expanded, added as regular field.
func TestNotExpandExternalStruct(t *testing.T) {
	typ := reflect.TypeFor[embedExternal]()
	s := Parse(typ, "q")
	checkFields(t, s, "Time", "c")
}

// Named type wrapping an external struct (type T time.Time) — NOT expanded.
func TestNotExpandWrappedExternal(t *testing.T) {
	typ := reflect.TypeFor[embedNamed]()
	s := Parse(typ, "q")
	checkFields(t, s, "a", "b", "c", "WrapTime")
}

// Unexported anonymous embed with exported sub-fields — still expanded before IsExported check.
func TestExpandUnexportedEmbed(t *testing.T) {
	typ := reflect.TypeFor[embedHidden]()
	s := Parse(typ, "q")
	checkFields(t, s, "x", "a")
}

// --- SetValue tests ---

func TestSetValueForce(t *testing.T) {
	typ := reflect.TypeFor[embedNamed]()
	s := Parse(typ, "q")

	fields := make(map[string]Field, len(s.Fields))
	for _, f := range s.Fields {
		fields[f.Name] = f
	}

	var dst embedNamed
	root := reflect.ValueOf(&dst).Elem()
	mustSet(t, fields["a"], root, "10", SetFlagForce)
	mustSet(t, fields["b"], root, "hello", SetFlagForce)
	mustSet(t, fields["c"], root, "99", SetFlagForce)
	if dst.A != 10 || dst.B != "hello" || dst.C != 99 {
		t.Fatalf("got A=%d B=%q C=%d", dst.A, dst.B, dst.C)
	}
}

func TestSetValueOnlyZero(t *testing.T) {
	typ := reflect.TypeFor[flatFields]()
	s := Parse(typ, "q")

	fields := make(map[string]Field, len(s.Fields))
	for _, f := range s.Fields {
		fields[f.Name] = f
	}

	var dst flatFields
	root := reflect.ValueOf(&dst).Elem()
	mustSet(t, fields["name"], root, "initial", SetFlagForce)
	mustSet(t, fields["value"], root, "42", SetFlagForce)

	// Non-zero field with SetFlagOnlyZero → skip.
	mustSet(t, fields["name"], root, "changed", SetFlagOnlyZero)
	if dst.Name != "initial" {
		t.Fatalf("SetFlagOnlyZero overwrote non-zero Name: got %q", dst.Name)
	}

	// Zero field with SetFlagOnlyZero → set value.
	var dst2 flatFields
	root2 := reflect.ValueOf(&dst2).Elem()
	mustSet(t, fields["name"], root2, "new", SetFlagOnlyZero)
	if dst2.Name != "new" {
		t.Fatalf("SetFlagOnlyZero on zero Name: got %q", dst2.Name)
	}
}

// --- Error path ---

func TestSetValueError(t *testing.T) {
	var dst embedNamed
	root := reflect.ValueOf(&dst).Elem()

	// index [0, 0, 0]: embedMe.A is int, not struct -> "invalid field path"
	err := makeValueSetter([]int{0, 0, 0}, reflect.TypeFor[int]())(root, "1", SetFlagForce)
	if err == nil || err.Error() != "invalid field path" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFieldByIndexAllocErrors(t *testing.T) {
	var err error

	_, err = fieldByIndexAlloc(reflect.ValueOf(embedNamed{}), nil)
	if err == nil || err.Error() != "empty field index" {
		t.Fatalf("unexpected error: %v", err)
	}

	// index [2]: C int, not struct. But it's len=1 so returns C directly (no error).
	// Use [2, 0] instead: C int is not the last level -> "invalid field path".
	_, err = fieldByIndexAlloc(reflect.ValueOf(embedNamed{}), []int{2, 0})
	if err == nil || err.Error() != "invalid field path" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFieldByIndexAllocPointerStruct(t *testing.T) {
	type nested struct{ N int }
	type holder struct{ P *nested }
	type badHolder struct{ P *int }

	h := holder{}
	v, err := fieldByIndexAlloc(reflect.ValueOf(&h).Elem(), []int{0, 0})
	if err != nil || !v.IsValid() || h.P == nil {
		t.Fatalf("unexpected result: %v %v %#v", v, err, h)
	}

	_, err = fieldByIndexAlloc(reflect.ValueOf(badHolder{}), []int{0, 0})
	if err == nil || err.Error() != "non-struct pointer in field path" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFieldByIndexAllocPointerNonNil(t *testing.T) {
	// Covers the "else" branch in fieldByIndexAlloc where a non-nil
	// pointer field is an intermediate node (not the last index element).
	type inner struct{ X int }
	type outer struct{ P *inner }

	v := &outer{P: &inner{X: 42}}
	rv := reflect.ValueOf(v).Elem()
	fv, err := fieldByIndexAlloc(rv, []int{0, 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !fv.IsValid() {
		t.Fatal("returned value is invalid")
	}
	if fv.Int() != 42 {
		t.Fatalf("expected 42, got %d", fv.Int())
	}
}

// Named struct field (not anonymous) — expanded into its sub-fields.
type namedStructFieldHost struct {
	Key string `q:"key"`
}

type namedStructFieldOuter struct {
	Inner namedStructFieldHost
	Label string `q:"label"`
}

func TestExpandNamedStructField(t *testing.T) {
	typ := reflect.TypeFor[namedStructFieldOuter]()
	s := Parse(typ, "q")
	checkFields(t, s, "key", "label")
}

// Named pointer-to-struct field — expanded into its sub-fields.
type namedPtrStructFieldHost struct {
	N int `q:"n"`
}

type namedPtrStructFieldOuter struct {
	Inner *namedPtrStructFieldHost
	Name  string `q:"name"`
}

func TestExpandNamedPointerStructField(t *testing.T) {
	typ := reflect.TypeFor[namedPtrStructFieldOuter]()
	s := Parse(typ, "q")
	checkFields(t, s, "n", "name")
}

// Named struct field without exported sub-fields — not expanded.
type namedStructNoExport struct {
	Named struct {
		x int // unexported
	}
	Tag string `q:"tag"`
}

func TestNotExpandNamedStructNoExportedFields(t *testing.T) {
	typ := reflect.TypeFor[namedStructNoExport]()
	s := Parse(typ, "q")
	// Named has no exported sub-fields, so it stays as a single field.
	checkFields(t, s, "Named", "tag")
}

// Unexported named struct field with exported sub-fields
// covers the "!sf.IsExported()" branch in parseWithParent
// when processing struct-typed named fields.
type unexportedNamedStructFieldHost struct {
	Val int `q:"val"`
}

type unexportedNamedStructOuter struct {
	inner unexportedNamedStructFieldHost
	Tag   string `q:"tag"`
}

func TestNotExpandUnexportedNamedStructField(t *testing.T) {
	typ := reflect.TypeFor[unexportedNamedStructOuter]()
	s := Parse(typ, "q")
	// inner is unexported, so it is skipped entirely.
	checkFields(t, s, "tag")
}

// --- Other ---

func TestParseCache(t *testing.T) {
	typ := reflect.TypeFor[flatFields]()
	s1 := Parse(typ, "q")
	s2 := Parse(typ, "q")
	if s1 != s2 {
		t.Fatal("cache miss")
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
