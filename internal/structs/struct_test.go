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

func mustSet(t *testing.T, f Field[string], root reflect.Value, s string) {
	t.Helper()
	if err := f.SetValue(root, s); err != nil {
		t.Fatalf("SetValue(%q): %v", s, err)
	}
}

func checkFields[T any](t *testing.T, s *Struct[T], want ...string) {
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

func fieldNames[T any](fields []Field[T]) []string {
	ns := make([]string, len(fields))
	for i, f := range fields {
		ns[i] = f.Name
	}
	return ns
}

// --- Requirements tests ---

func TestCompileAnySetter(t *testing.T) {
	s := AnyParser.Parse(reflect.TypeFor[embedNamed](), "q")
	checkFields(t, s, "a", "b", "c", "WrapTime")
}

func TestAnySetterTimeField(t *testing.T) {
	type holder struct {
		At time.Time `q:"at"`
	}

	s := AnyParser.Parse(reflect.TypeFor[holder](), "q")
	checkFields(t, s, "at")

	ts := time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)
	var dst holder
	if err := s.Fields[0].SetValue(reflect.ValueOf(&dst).Elem(), ts.Format(time.RFC3339)); err != nil {
		t.Fatalf("SetValue(time string): %v", err)
	}
	if !dst.At.Equal(ts) {
		t.Fatalf("got %v, want %v", dst.At, ts)
	}
}

// Named struct embedded anonymously from the same package — should expand.
func TestExpandNamedStruct(t *testing.T) {
	s := StringParser.Parse(reflect.TypeFor[embedNamed](), "q")
	checkFields(t, s, "a", "b", "c", "WrapTime")
}

// Pointer to named struct embedded anonymously — should dereference and expand.
func TestExpandPointerEmbed(t *testing.T) {
	s := StringParser.Parse(reflect.TypeFor[embedPointer](), "q")
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
	s := StringParser.Parse(reflect.TypeFor[embedLiteral](), "q")
	checkFields(t, s, "z")
}

// External struct (time.Time) embedded anonymously — NOT expanded, added as regular field.
func TestNotExpandExternalStruct(t *testing.T) {
	typ := reflect.TypeFor[embedExternal]()
	s := StringParser.Parse(typ, "q")
	checkFields(t, s, "Time", "c")
}

// Named type wrapping an external struct (type T time.Time) — NOT expanded.
func TestNotExpandWrappedExternal(t *testing.T) {
	typ := reflect.TypeFor[embedNamed]()
	s := StringParser.Parse(typ, "q")
	checkFields(t, s, "a", "b", "c", "WrapTime")
}

// Unexported anonymous embed with exported sub-fields — still expanded before IsExported check.
func TestExpandUnexportedEmbed(t *testing.T) {
	typ := reflect.TypeFor[embedHidden]()
	s := StringParser.Parse(typ, "q")
	checkFields(t, s, "x", "a")
}

// --- SetValue tests ---

func TestSetValueSuccess(t *testing.T) {
	typ := reflect.TypeFor[embedNamed]()
	s := StringParser.Parse(typ, "q")

	fields := make(map[string]Field[string], len(s.Fields))
	for _, f := range s.Fields {
		fields[f.Name] = f
	}

	var dst embedNamed
	root := reflect.ValueOf(&dst).Elem()
	mustSet(t, fields["a"], root, "10")
	mustSet(t, fields["b"], root, "hello")
	mustSet(t, fields["c"], root, "99")
	if dst.A != 10 || dst.B != "hello" || dst.C != 99 {
		t.Fatalf("got A=%d B=%q C=%d", dst.A, dst.B, dst.C)
	}
}

// --- Error path ---

func TestFieldByIndexAllocErrors(t *testing.T) {
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for empty field index")
			}
			if msg, ok := r.(string); !ok || msg != "empty field index" {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()
		fieldByIndexAlloc(reflect.ValueOf(embedNamed{}), nil)
	}()

	// index [2]: C int, not struct. But it's len=1 so returns C directly (no error).
	// Use [2, 0] instead: C int is not the last level -> "invalid field path".
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for invalid field path")
			}
			if msg, ok := r.(string); !ok || msg != "invalid field path" {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()
		fieldByIndexAlloc(reflect.ValueOf(embedNamed{}), []int{2, 0})
	}()
}

func TestFieldByIndexAllocPointerStruct(t *testing.T) {
	type nested struct{ N int }
	type holder struct{ P *nested }
	type badHolder struct{ P *int }

	h := holder{}
	v := fieldByIndexAlloc(reflect.ValueOf(&h).Elem(), []int{0, 0})
	if !v.IsValid() || h.P == nil {
		t.Fatalf("unexpected result: %v %#v", v, h)
	}

	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for non-struct pointer in field path")
			}
			if msg, ok := r.(string); !ok || msg != "non-struct pointer in field path" {
				t.Fatalf("unexpected panic message: %v", r)
			}
		}()
		fieldByIndexAlloc(reflect.ValueOf(badHolder{}), []int{0, 0})
	}()
}

func TestFieldByIndexAllocPointerNonNil(t *testing.T) {
	// Covers the "else" branch in fieldByIndexAlloc where a non-nil
	// pointer field is an intermediate node (not the last index element).
	type inner struct{ X int }
	type outer struct{ P *inner }

	v := &outer{P: &inner{X: 42}}
	rv := reflect.ValueOf(v).Elem()
	fv := fieldByIndexAlloc(rv, []int{0, 0})
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
	s := StringParser.Parse(typ, "q")
	checkFields(t, s, "key", "label")
}

type opaqueNamedStructOuter struct {
	Inner namedStructFieldHost `q:"inner,opaque"`
	Outer namedStructFieldHost `q:",opaque"`
	Label string               `q:"label"`
}

func TestOpaqueNamedStructField(t *testing.T) {
	typ := reflect.TypeFor[opaqueNamedStructOuter]()
	s := StringParser.Parse(typ, "q")
	checkFields(t, s, "inner", "Outer", "label")

	fields := make(map[string]Field[string], len(s.Fields))
	for _, f := range s.Fields {
		fields[f.Name] = f
	}
	if got := fields["inner"].GetValue(map[string]any{"inner": "value"}); got != "value" {
		t.Fatalf("expected opaque field value, got %v", got)
	}
}

// Named pointer-to-struct field — expanded into its sub-fields.
type taggedNestedHost struct {
	Key   string `q:"key"`
	Count int    `q:"count"`
}

type taggedNestedOuter struct {
	Inner taggedNestedHost `q:"inner"`
	Label string           `q:"label"`
}

type namedPtrStructFieldHost struct {
	N int `q:"n"`
}

type namedPtrStructFieldOuter struct {
	Inner *namedPtrStructFieldHost
	Name  string `q:"name"`
}

func TestExpandNamedPointerStructField(t *testing.T) {
	typ := reflect.TypeFor[namedPtrStructFieldOuter]()
	s := StringParser.Parse(typ, "q")
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
	s := StringParser.Parse(typ, "q")
	// Named has no exported sub-fields, so it stays as a single field.
	checkFields(t, s, "Named", "tag")
}

// Unexported named struct field with exported sub-fields
// covers the "!sf.IsExported()" branch in parseWithParent
// when processing struct-typed named fields.
type unexportedNamedStructFieldHost struct { //nolint:unused
	Val int `q:"val"`
}

type unexportedNamedStructOuter struct {
	inner unexportedNamedStructFieldHost //nolint:unused

	Tag string `q:"tag"`
}

func TestNotExpandUnexportedNamedStructField(t *testing.T) {
	typ := reflect.TypeFor[unexportedNamedStructOuter]()
	s := StringParser.Parse(typ, "q")
	// inner is unexported, so it is skipped entirely.
	checkFields(t, s, "tag")
}

// --- GetValue tests ---

func TestGetValueNonNil(t *testing.T) {
	typ := reflect.TypeFor[embedNamed]()
	s := StringParser.Parse(typ, "q")
	for _, f := range s.Fields {
		if f.GetValue == nil {
			t.Fatalf("field %q has nil GetValue", f.Name)
		}
	}
}

func TestGetValueFlatFields(t *testing.T) {
	typ := reflect.TypeFor[flatFields]()
	s := StringParser.Parse(typ, "q")

	// Nil map → returns nil
	for _, f := range s.Fields {
		if got := f.GetValue(nil); got != nil {
			t.Fatalf("field %q: expected nil, got %v", f.Name, got)
		}
	}

	// Key exists → returns value
	m := map[string]any{"name": "hello", "value": 42}
	for _, f := range s.Fields {
		switch f.Name {
		case "name":
			if got := f.GetValue(m); got != "hello" {
				t.Fatalf("expected 'hello', got %v", got)
			}
		case "value":
			if got := f.GetValue(m); got != 42 {
				t.Fatalf("expected 42, got %v", got)
			}
		}
	}

	// Key missing → returns nil
	m = map[string]any{"name": "hello"}
	for _, f := range s.Fields {
		if f.Name == "value" {
			if got := f.GetValue(m); got != nil {
				t.Fatalf("expected nil, got %v", got)
			}
			return
		}
	}
}

func TestGetValueNestedFields(t *testing.T) {
	typ := reflect.TypeFor[taggedNestedOuter]()
	s := StringParser.Parse(typ, "q")

	// Key has a nested path ["inner", "key"] because the named struct
	// field "Inner" (tag "inner") is expanded during parsing.
	var key Field[string]
	for _, f := range s.Fields {
		if f.Name == "key" {
			key = f
			break
		}
	}

	// Key exists
	if got := key.GetValue(map[string]any{
		"inner": map[string]any{"key": "v"},
	}); got != "v" {
		t.Fatalf("expected 'v', got %v", got)
	}

	// Missing intermediate key
	if got := key.GetValue(map[string]any{}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}

	// Wrong intermediate type
	if got := key.GetValue(map[string]any{"inner": "bad"}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}

	// Nil intermediate map (type assertion succeeds, nil-map read returns nil)
	if got := key.GetValue(map[string]any{"inner": map[string]any(nil)}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestGetValueNestedFieldsWithoutTag(t *testing.T) {
	type inner struct {
		Key string
	}
	type outer struct {
		Inner inner
	}

	typ := reflect.TypeFor[outer]()
	s := StringParser.Parse(typ, "")
	checkFields(t, s, "Key")

	key := s.Fields[0]
	if got := key.GetValue(map[string]any{
		"Inner": map[string]any{"Key": "v"},
	}); got != "v" {
		t.Fatalf("expected 'v', got %v", got)
	}

	if got := key.GetValue(map[string]any{
		"": map[string]any{"Key": "bad"},
	}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestGetValuePathNamesAreIndependent(t *testing.T) {
	type inner struct {
		A string
		B string
	}

	parentNames := make([]string, 1, 2)
	parentNames[0] = "Root"
	parser := _Parser[string]{CompileSetter: CompileStringSetter}
	fields := parser.parse(reflect.TypeFor[inner](), nil, parentNames)
	checkFields(t, &Struct[string]{Fields: fields}, "A", "B")

	values := map[string]any{"Root": map[string]any{"A": "a", "B": "b"}}
	for _, f := range fields {
		if f.Name == "A" {
			if got := f.GetValue(values); got != "a" {
				t.Fatalf("expected 'a', got %v", got)
			}
			return
		}
	}
	t.Fatal("missing field A")
}

func TestGetterPathsAreIndependentFromPublicPaths(t *testing.T) {
	typ := reflect.TypeFor[flatFields]()
	parser := _Parser[string]{CompileSetter: CompileStringSetter, Tag: "q"}
	s := parser.Parse(typ)

	var name *Field[string]
	for i := range s.Fields {
		if s.Fields[i].Name == "name" {
			name = &s.Fields[i]
			break
		}
	}
	if name == nil {
		t.Fatal("missing field name")
	}

	name.Names[0] = "value"
	if got := name.GetValue(map[string]any{"name": "hello", "value": "bad"}); got != "hello" {
		t.Fatalf("expected 'hello', got %v", got)
	}

	name.Indexes[0] = 1
	root := reflect.ValueOf(flatFields{Name: "hello", Value: 42})
	if got := name.GetField(root).String(); got != "hello" {
		t.Fatalf("expected 'hello', got %v", got)
	}
}

func TestMakeMapValueGetterEmptyNames(t *testing.T) {
	getter := makeMapValueGetter(nil)
	if got := getter(map[string]any{"x": 1}); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

// --- Other ---

func TestParserNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = NewParser[string](nil)
}

func TestParseCache(t *testing.T) {
	typ := reflect.TypeFor[flatFields]()
	s1 := StringParser.Parse(typ, "q")
	s2 := StringParser.Parse(typ, "q")
	if s1 != s2 {
		t.Fatal("cache miss")
	}
}

func TestParserCachesAreIndependent(t *testing.T) {
	typ := reflect.TypeFor[flatFields]()
	compile := func(prefix string) SetterCompiler[string] {
		return func(reflect.Type) SetterFunc[string] {
			return func(_ reflect.Type, dst reflect.Value, src string) error {
				dst.SetString(prefix + src)
				return nil
			}
		}
	}

	p1 := NewParser(compile("p1:"))
	p2 := NewParser(compile("p2:"))
	s1 := p1.Parse(typ, "q")
	s2 := p2.Parse(typ, "q")
	if s1 == s2 {
		t.Fatal("different parsers shared a cached struct")
	}

	root1 := reflect.ValueOf(&flatFields{}).Elem()
	root2 := reflect.ValueOf(&flatFields{}).Elem()
	if err := s1.Fields[0].SetValue(root1, "value"); err != nil {
		t.Fatal(err)
	}
	if err := s2.Fields[0].SetValue(root2, "value"); err != nil {
		t.Fatal(err)
	}
	if got := root1.FieldByName("Name").String(); got != "p1:value" {
		t.Fatalf("unexpected parser 1 value: %q", got)
	}
	if got := root2.FieldByName("Name").String(); got != "p2:value" {
		t.Fatalf("unexpected parser 2 value: %q", got)
	}
}

func TestParseHelpers(t *testing.T) {
	if name, opaque := parseTag(""); name != "" || opaque {
		t.Fatal("unexpected empty tag parse")
	}
	if name, opaque := parseTag("name,omitempty"); name != "name" || opaque {
		t.Fatal("unexpected tag parse")
	}
	if name, opaque := parseTag("name,omitempty,opaque"); name != "name" || !opaque {
		t.Fatal("expected opaque option")
	}
	if name, opaque := parseTag("name"); name != "name" || opaque {
		t.Fatal("unexpected name-only tag parse")
	}
	idx := appendSlice([]int{1, 2}, 3)
	if len(idx) != 3 || idx[2] != 3 {
		t.Fatalf("unexpected index: %#v", idx)
	}
}
