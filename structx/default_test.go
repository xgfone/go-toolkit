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
	"testing"
	"time"
)

// ============================================================================
//  Basic type coverage
// ============================================================================

type testDefaultSimple struct {
	Name string `default:"hello"`
	Age  int    `default:"18"`
}

type testDefaultAllTypes struct {
	String  string  `default:"str"`
	Int     int     `default:"-10"`
	Int8    int8    `default:"-8"`
	Int16   int16   `default:"-16"`
	Int32   int32   `default:"-32"`
	Int64   int64   `default:"-64"`
	Uint    uint    `default:"10"`
	Uint8   uint8   `default:"8"`
	Uint16  uint16  `default:"16"`
	Uint32  uint32  `default:"32"`
	Uint64  uint64  `default:"64"`
	Float32 float32 `default:"3.14"`
	Float64 float64 `default:"2.718"`
	Bool    bool    `default:"true"`
}

func TestSetDefaultSimple(t *testing.T) {
	v := &testDefaultSimple{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "hello" {
		t.Errorf("Name: expect %q, got %q", "hello", v.Name)
	}
	if v.Age != 18 {
		t.Errorf("Age: expect %d, got %d", 18, v.Age)
	}
}

func TestSetDefaultAllTypes(t *testing.T) {
	v := &testDefaultAllTypes{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.String != "str" {
		t.Errorf("String: expect %q, got %q", "str", v.String)
	}
	if v.Int != -10 {
		t.Errorf("Int: expect %d, got %d", -10, v.Int)
	}
	if v.Int8 != -8 {
		t.Errorf("Int8: expect %d, got %d", -8, v.Int8)
	}
	if v.Int16 != -16 {
		t.Errorf("Int16: expect %d, got %d", -16, v.Int16)
	}
	if v.Int32 != -32 {
		t.Errorf("Int32: expect %d, got %d", -32, v.Int32)
	}
	if v.Int64 != -64 {
		t.Errorf("Int64: expect %d, got %d", -64, v.Int64)
	}
	if v.Uint != 10 {
		t.Errorf("Uint: expect %d, got %d", 10, v.Uint)
	}
	if v.Uint8 != 8 {
		t.Errorf("Uint8: expect %d, got %d", 8, v.Uint8)
	}
	if v.Uint16 != 16 {
		t.Errorf("Uint16: expect %d, got %d", 16, v.Uint16)
	}
	if v.Uint32 != 32 {
		t.Errorf("Uint32: expect %d, got %d", 32, v.Uint32)
	}
	if v.Uint64 != 64 {
		t.Errorf("Uint64: expect %d, got %d", 64, v.Uint64)
	}
	if v.Float32 != float32(3.14) {
		t.Errorf("Float32: expect %v, got %v", float32(3.14), v.Float32)
	}
	if v.Float64 != 2.718 {
		t.Errorf("Float64: expect %v, got %v", 2.718, v.Float64)
	}
	if !v.Bool {
		t.Errorf("Bool: expect true, got false")
	}
}

// ============================================================================
//  No defaults / partial defaults
// ============================================================================

type testDefaultNoDefaults struct {
	Name string
	Age  int
}

type testDefaultPartial struct {
	Name    string  `default:"partial_name"`
	Age     int     `default:"20"`
	Score   float64 // no default
	Active  bool    // no default
	Country string  `default:"CN"`
}

func TestSetDefaultNoDefaults(t *testing.T) {
	v := &testDefaultNoDefaults{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "" {
		t.Errorf("Name: expect empty, got %q", v.Name)
	}
	if v.Age != 0 {
		t.Errorf("Age: expect 0, got %d", v.Age)
	}
}

func TestSetDefaultPartial(t *testing.T) {
	v := &testDefaultPartial{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "partial_name" {
		t.Errorf("Name: expect %q, got %q", "partial_name", v.Name)
	}
	if v.Age != 20 {
		t.Errorf("Age: expect %d, got %d", 20, v.Age)
	}
	if v.Score != 0 {
		t.Errorf("Score: expect 0, got %v", v.Score)
	}
	if v.Active {
		t.Errorf("Active: expect false, got true")
	}
	if v.Country != "CN" {
		t.Errorf("Country: expect %q, got %q", "CN", v.Country)
	}
}

// ============================================================================
//  Already-set / zero-value semantics
// ============================================================================

type testDefaultAlreadySet struct {
	Name   string  `default:"should_not_change"`
	Age    int     `default:"99"`
	Score  float64 `default:"3.14"`
	Active bool    `default:"true"`
}

func TestSetDefaultAlreadySet(t *testing.T) {
	v := &testDefaultAlreadySet{
		Name: "custom_name",
		Age:  50,
	}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}

	// Non-zero fields should NOT be overwritten.
	if v.Name != "custom_name" {
		t.Errorf("Name: expect %q, got %q (should not be overwritten)", "custom_name", v.Name)
	}
	if v.Age != 50 {
		t.Errorf("Age: expect %d, got %d (should not be overwritten)", 50, v.Age)
	}

	// Zero fields SHOULD be set.
	if v.Score != 3.14 {
		t.Errorf("Score: expect %v, got %v", 3.14, v.Score)
	}
	if !v.Active {
		t.Errorf("Active: expect true, got false")
	}
}

// ============================================================================
//  Error paths
// ============================================================================

type testDefaultBadValue struct {
	Age int `default:"not_a_number"`
}

func TestSetDefaultBadValue(t *testing.T) {
	v := &testDefaultBadValue{}
	err := SetDefault(v)
	if err == nil {
		t.Fatal("expected error for bad default value")
	}
	if errors.Is(err, errDefaultNotStruct) {
		t.Fatal("expected a field parsing error, not errDefaultNotStruct")
	}
	if errors.Is(err, errDefaultNilPointer) {
		t.Fatal("expected a field parsing error, not errDefaultNilPointer")
	}
}

func TestSetDefaultNilPointer(t *testing.T) {
	var v *testDefaultSimple
	err := SetDefault(v)
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
	if !errors.Is(err, errDefaultNilPointer) {
		t.Errorf("expected errDefaultNilPointer, got %v", err)
	}
}

func TestSetDefaultNonStruct(t *testing.T) {
	v := 42
	err := SetDefault(&v)
	if err == nil {
		t.Fatal("expected error for non-struct")
	}
	if !errors.Is(err, errDefaultNotStruct) {
		t.Errorf("expected errDefaultNotStruct, got %v", err)
	}
}

func TestSetDefaultIntOverflow(t *testing.T) {
	type testInt8 struct {
		V int8 `default:"999"`
	}
	v := &testInt8{}
	err := SetDefault(v)
	if err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestSetDefaultEmptyDefaultTag(t *testing.T) {
	type testEmptyDefault struct {
		Name string `default:""`
		Age  int    `default:""`
	}
	v := &testEmptyDefault{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// Fields with empty default values should be left as zero.
	if v.Name != "" || v.Age != 0 {
		t.Errorf("expected zero values, got %+v", v)
	}
}

// ============================================================================
//  Edge cases
// ============================================================================

type testDefaultSet struct {
	Value int `default:"42"`
}

func TestSetDefaultMultipleCalls(t *testing.T) {
	v := &testDefaultSet{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Value != 42 {
		t.Fatalf("Value: expect %d, got %d", 42, v.Value)
	}

	// Second call should not change anything (already set).
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Value != 42 {
		t.Errorf("second call changed Value to %d", v.Value)
	}
}

func TestSetDefaultUnexportedField(t *testing.T) {
	type testUnexported struct {
		Name   string `default:"visible"`
		hidden string `default:"invisible"`
	}
	v := &testUnexported{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Name != "visible" {
		t.Errorf("Name: expect %q, got %q", "visible", v.Name)
	}
	// Unexported field should be silently ignored; no error expected.
}

func TestSetDefaultWithPointerField(t *testing.T) {
	type testPtrField struct {
		Name *string `default:"ptr_default"`
	}
	v := &testPtrField{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Name == nil {
		t.Fatal("Name pointer should not be nil after SetDefault")
	}
	if *v.Name != "ptr_default" {
		t.Errorf("Name: expect %q, got %q", "ptr_default", *v.Name)
	}
}

func TestSetDefaultWithPointerFieldNonNil(t *testing.T) {
	existing := "existing"
	type testPtrField struct {
		Name *string `default:"should_not_apply"`
	}
	v := &testPtrField{Name: &existing}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Name != &existing {
		t.Fatal("pointer was replaced when it was already non-nil")
	}
	if *v.Name != "existing" {
		t.Errorf("Name: expect %q, got %q", "existing", *v.Name)
	}
}

// ============================================================================
//  Anonymous struct embedding — same-package named struct
// ============================================================================

type testDefaultInner struct {
	Host string `default:"localhost"`
	Port int    `default:"8080"`
}

type testDefaultOuter struct {
	testDefaultInner
	Label string `default:"outer_label"`
}

func TestSetDefaultEmbeddedNamedStruct(t *testing.T) {
	v := &testDefaultOuter{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Host != "localhost" {
		t.Errorf("Host: expect %q, got %q", "localhost", v.Host)
	}
	if v.Port != 8080 {
		t.Errorf("Port: expect %d, got %d", 8080, v.Port)
	}
	if v.Label != "outer_label" {
		t.Errorf("Label: expect %q, got %q", "outer_label", v.Label)
	}
}

func TestSetDefaultEmbeddedNamedStructAlreadySet(t *testing.T) {
	v := &testDefaultOuter{
		testDefaultInner: testDefaultInner{
			Host: "custom-host",
		},
	}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// Already-set non-zero fields should not be overwritten.
	if v.Host != "custom-host" {
		t.Errorf("Host: expect %q, got %q (should not be overwritten)", "custom-host", v.Host)
	}
	// Zero fields should be set.
	if v.Port != 8080 {
		t.Errorf("Port: expect %d, got %d", 8080, v.Port)
	}
	if v.Label != "outer_label" {
		t.Errorf("Label: expect %q, got %q", "outer_label", v.Label)
	}
}

// ============================================================================
//  Pointer to embedded named struct
// ============================================================================

type TestDefaultPointerInner struct {
	Key   string `default:"key_default"`
	Value int    `default:"100"`
}

type TestDefaultPointerOuter struct {
	*TestDefaultPointerInner
	Name string `default:"ptr_embed_name"`
}

func TestSetDefaultEmbeddedPointerStruct(t *testing.T) {
	v := &TestDefaultPointerOuter{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.TestDefaultPointerInner == nil {
		t.Fatal("embedded pointer should be auto-allocated")
	}
	if v.Key != "key_default" {
		t.Errorf("Key: expect %q, got %q", "key_default", v.Key)
	}
	if v.Value != 100 {
		t.Errorf("Value: expect %d, got %d", 100, v.Value)
	}
	if v.Name != "ptr_embed_name" {
		t.Errorf("Name: expect %q, got %q", "ptr_embed_name", v.Name)
	}
}

func TestSetDefaultEmbeddedPointerStructAlreadySet(t *testing.T) {
	v := &TestDefaultPointerOuter{
		TestDefaultPointerInner: &TestDefaultPointerInner{
			Key:   "existing_key",
			Value: 999,
		},
	}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// Already-set fields should not be overwritten.
	if v.Key != "existing_key" {
		t.Errorf("Key: expect %q, got %q (should not be overwritten)", "existing_key", v.Key)
	}
	if v.Value != 999 {
	}
}

// ============================================================================
//  External struct embedding (e.g. time.Time) — NOT expanded
// ============================================================================

type testDefaultExternalEmbed struct {
	time.Time
	Label string `default:"ext_embed_label"`
	Stamp int64  `default:"12345"`
}

func TestSetDefaultExternalEmbed(t *testing.T) {
	v := &testDefaultExternalEmbed{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// time.Time has no "default" tag, so it should remain zero.
	if !v.Time.IsZero() {
		t.Errorf("Time: expect zero, got %v", v.Time)
	}
	if v.Label != "ext_embed_label" {
		t.Errorf("Label: expect %q, got %q", "ext_embed_label", v.Label)
	}
	if v.Stamp != 12345 {
		t.Errorf("Stamp: expect %d, got %d", 12345, v.Stamp)
	}
}

// ============================================================================
//  Named type wrapping an external struct — NOT expanded
// ============================================================================

type testDefaultWrapTime time.Time

type testDefaultWrapEmbed struct {
	testDefaultWrapTime
	Desc string `default:"wrap_desc"`
}

func TestSetDefaultWrappedTypeEmbed(t *testing.T) {
	v := &testDefaultWrapEmbed{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// testDefaultWrapTime is a named type wrapping time.Time; it has no "default"
	// tag and no exported fields of its own (it's just a time.Time alias),
	// so it should appear as a single regular field with no default.
	if v.Desc != "wrap_desc" {
		t.Errorf("Desc: expect %q, got %q", "wrap_desc", v.Desc)
	}
}

// ============================================================================
//  Deeply nested embedding
// ============================================================================

type testDefaultLevel0 struct {
	A string `default:"level0_a"`
	B int    `default:"10"`
}

type testDefaultLevel1 struct {
	testDefaultLevel0
	C string `default:"level1_c"`
}

type testDefaultLevel2 struct {
	testDefaultLevel1
	D string `default:"level2_d"`
}

func TestSetDefaultDeepNested(t *testing.T) {
	v := &testDefaultLevel2{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.A != "level0_a" {
		t.Errorf("A: expect %q, got %q", "level0_a", v.A)
	}
	if v.B != 10 {
		t.Errorf("B: expect %d, got %d", 10, v.B)
	}
	if v.C != "level1_c" {
		t.Errorf("C: expect %q, got %q", "level1_c", v.C)
	}
	if v.D != "level2_d" {
		t.Errorf("D: expect %q, got %q", "level2_d", v.D)
	}
}

// ============================================================================
//  Mixed embedding: regular named field + embedded + embedded pointer
// ============================================================================

type TestDefaultMixA struct {
	X string `default:"mix_a_x"`
}

type TestDefaultMixB struct {
	*TestDefaultMixA
	Y string `default:"mix_b_y"`
}

type TestDefaultMixC struct {
	TestDefaultMixB
	Z int `default:"999"`
}

func TestSetDefaultMixedEmbedding(t *testing.T) {
	v := &TestDefaultMixC{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.TestDefaultMixB.TestDefaultMixA == nil {
		t.Fatal("embedded pointer chain should be auto-allocated")
	}
	if v.X != "mix_a_x" {
		t.Errorf("X: expect %q, got %q", "mix_a_x", v.X)
	}
	if v.Y != "mix_b_y" {
		t.Errorf("Y: expect %q, got %q", "mix_b_y", v.Y)
	}
	if v.Z != 999 {
		t.Errorf("Z: expect %d, got %d", 999, v.Z)
	}
}

// ============================================================================
//  Unexported anonymous embed with exported sub-fields
// ============================================================================

// unexportedEmb is used via embedding in testDefaultHidden — it is in the
// same package, so its exported field "X" should be expanded.
type testDefaultUnexportedEmb struct {
	X string `default:"unexported_emb_x"`
}

// unexportedEmbNoExport has no exported fields at all — should not be expanded.
// Its containing fields should be skipped entirely.
type testDefaultEmbNoExport struct {
	x int
	y string
}

type testDefaultHidden struct {
	testDefaultEmbNoExport
	testDefaultUnexportedEmb
	A int `default:"100"`
}

func TestSetDefaultHiddenEmbed(t *testing.T) {
	v := &testDefaultHidden{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// testDefaultEmbNoExport has no exported fields, so it should be skipped.
	// testDefaultUnexportedEmb has exported field "X", so it should be expanded.
	if v.X != "unexported_emb_x" {
		t.Errorf("X: expect %q, got %q", "unexported_emb_x", v.X)
	}
	if v.A != 100 {
		t.Errorf("A: expect %d, got %d", 100, v.A)
	}
}

// ============================================================================
//  Named field (not anonymous embed) with anonymous struct type
// ============================================================================

type testDefaultLiteralEmbed struct {
	_ struct {
		IgnoreMe int    `default:"should_be_skipped"`
		AlsoMe   string `default:"also_skipped"`
	}
	Z int `default:"42"`
}

func TestSetDefaultLiteralEmbed(t *testing.T) {
	v := &testDefaultLiteralEmbed{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// The anonymous struct field is a literal field (not embedded), so it's not
	// expanded. It has no name (blank identifier), so it's unexported and skipped.
	if v.Z != 42 {
		t.Errorf("Z: expect %d, got %d", 42, v.Z)
	}
}

// ============================================================================
//  Pointer field with automatic allocation (non-embedded)
// ============================================================================

type testDefaultPtrStruct struct {
	Name string `default:"ptr_struct_name"`
}

type testDefaultOuterWithPtr struct {
	Inner *testDefaultPtrStruct // inner sub-fields have "default" tags
	Label string                `default:"outer_ptr_label"`
}

func TestSetDefaultPointerFieldAutoAlloc(t *testing.T) {
	// The named pointer-to-struct field Inner is now expanded into its
	// sub-fields. Since Name has a "default" tag, Inner is auto-allocated
	// and Name is set.
	v := &testDefaultOuterWithPtr{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Inner == nil {
		t.Fatal("Inner: expected non-nil (auto-allocated for sub-field defaults)")
	}
	if v.Inner.Name != "ptr_struct_name" {
		t.Errorf("Inner.Name: expect %q, got %q", "ptr_struct_name", v.Inner.Name)
	}
	if v.Label != "outer_ptr_label" {
		t.Errorf("Label: expect %q, got %q", "outer_ptr_label", v.Label)
	}
}

// ============================================================================
//  Multiple embedded structs with overlapping field names
// ============================================================================

type testDefaultOverlapA struct {
	Common string `default:"from_a"`
}

type testDefaultOverlapB struct {
	Common  string `default:"from_b"`
	UniqueB int    `default:"42"`
}

type testDefaultOverlapOuter struct {
	testDefaultOverlapA
	testDefaultOverlapB
	Own string `default:"own_val"`
}

func TestSetDefaultOverlappingEmbed(t *testing.T) {
	v := &testDefaultOverlapOuter{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// When the same field name appears in multiple embedded structs,
	// Parse should include both as separate fields with distinct index paths.
	// SetDefault should be able to set both without error.
	if v.testDefaultOverlapA.Common != "from_a" {
		t.Errorf("testDefaultOverlapA.Common: expect %q, got %q", "from_a", v.testDefaultOverlapA.Common)
	}
	if v.testDefaultOverlapB.Common != "from_b" {
		t.Errorf("testDefaultOverlapB.Common: expect %q, got %q", "from_b", v.testDefaultOverlapB.Common)
	}
	if v.UniqueB != 42 {
		t.Errorf("UniqueB: expect %d, got %d", 42, v.UniqueB)
	}
	if v.Own != "own_val" {
		t.Errorf("Own: expect %q, got %q", "own_val", v.Own)
	}
}

// ============================================================================
//  Pointer *bool field
// ============================================================================

func TestSetDefaultPointerBool(t *testing.T) {
	type testPtrBool struct {
		Flag *bool `default:"true"`
	}
	v := &testPtrBool{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Flag == nil {
		t.Fatal("Flag pointer should not be nil after SetDefault")
	}
	if !*v.Flag {
		t.Errorf("Flag: expect true, got false")
	}
}

func TestSetDefaultPointerBoolNonNil(t *testing.T) {
	f := false
	type testPtrBool struct {
		Flag *bool `default:"true"` // should NOT overwrite
	}
	v := &testPtrBool{Flag: &f}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Flag != &f {
		t.Fatal("Flag pointer was replaced")
	}
	if *v.Flag {
		t.Errorf("Flag: expect false (should not be overwritten), got true")
	}
}

// ============================================================================
//  *int / *float / *uint pointer fields
// ============================================================================

func TestSetDefaultPointerInt(t *testing.T) {
	type testPtrInt struct {
		V *int `default:"77"`
	}
	v := &testPtrInt{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.V == nil || *v.V != 77 {
		t.Errorf("V: expect 77, got %v", v.V)
	}
}

func TestSetDefaultPointerUint(t *testing.T) {
	type testPtrUint struct {
		V *uint `default:"55"`
	}
	v := &testPtrUint{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.V == nil || *v.V != 55 {
		t.Errorf("V: expect 55, got %v", v.V)
	}
}

func TestSetDefaultPointerFloat(t *testing.T) {
	type testPtrFloat struct {
		V *float64 `default:"3.5"`
	}
	v := &testPtrFloat{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.V == nil || *v.V != 3.5 {
		t.Errorf("V: expect 3.5, got %v", v.V)
	}
}

// ============================================================================
//  Named struct field (not anonymous)
// ============================================================================

type testDefaultNamedInner struct {
	InnerName string `default:"inner_name_val"`
	InnerAge  int    `default:"25"`
}

type testDefaultNamedOuter struct {
	Config testDefaultNamedInner // sub-fields have "default" tags, expanded
	Tag    string                `default:"named_outer_tag"`
}

func TestSetDefaultNamedStructField(t *testing.T) {
	// Named struct fields are now expanded by Parse; sub-fields with
	// "default" tags are discovered and set individually.
	v := &testDefaultNamedOuter{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Config.InnerName != "inner_name_val" {
		t.Errorf("InnerName: expect %q, got %q", "inner_name_val", v.Config.InnerName)
	}
	if v.Config.InnerAge != 25 {
		t.Errorf("InnerAge: expect %d, got %d", 25, v.Config.InnerAge)
	}
	if v.Tag != "named_outer_tag" {
		t.Errorf("Tag: expect %q, got %q", "named_outer_tag", v.Tag)
	}
}

// ============================================================================
//  Recursive named struct field (Scheme B)
// ============================================================================

type testDefaultRecurseInner struct {
	A int `default:"1"`
}

type testDefaultRecurseOuter struct {
	Inner testDefaultRecurseInner
	Tag   string `default:"outer_tag"`
}

func TestSetDefaultRecurseNamedStructField(t *testing.T) {
	v := &testDefaultRecurseOuter{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Inner.A != 1 {
		t.Errorf("Inner.A: expect %d, got %d", 1, v.Inner.A)
	}
	if v.Tag != "outer_tag" {
		t.Errorf("Tag: expect %q, got %q", "outer_tag", v.Tag)
	}
}

func TestSetDefaultRecurseNamedStructFieldAlreadySet(t *testing.T) {
	v := &testDefaultRecurseOuter{Inner: testDefaultRecurseInner{A: 99}}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// Inner.A is non-zero, should NOT be overwritten.
	if v.Inner.A != 99 {
		t.Errorf("Inner.A: expect %d (should not be overwritten), got %d", 99, v.Inner.A)
	}
	// Tag is zero, should be set.
	if v.Tag != "outer_tag" {
		t.Errorf("Tag: expect %q, got %q", "outer_tag", v.Tag)
	}
}

type testDefaultRecurseMultiLevel struct {
	Level1 struct {
		X      string `default:"l1_x"`
		Level2 struct {
			Y      int `default:"42"`
			Level3 struct {
				Z bool `default:"true"`
			}
		}
	}
	Top string `default:"top_val"`
}

func TestSetDefaultRecurseMultiLevel(t *testing.T) {
	v := &testDefaultRecurseMultiLevel{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Level1.X != "l1_x" {
		t.Errorf("Level1.X: expect %q, got %q", "l1_x", v.Level1.X)
	}
	if v.Level1.Level2.Y != 42 {
		t.Errorf("Level1.Level2.Y: expect %d, got %d", 42, v.Level1.Level2.Y)
	}
	if !v.Level1.Level2.Level3.Z {
		t.Errorf("Level1.Level2.Level3.Z: expect true, got false")
	}
	if v.Top != "top_val" {
		t.Errorf("Top: expect %q, got %q", "top_val", v.Top)
	}
}

func TestSetDefaultRecurseMultiLevelPartialSet(t *testing.T) {
	v := &testDefaultRecurseMultiLevel{}
	v.Level1.Level2.Y = 100
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// Level2.Y is non-zero → should NOT be overwritten.
	if v.Level1.Level2.Y != 100 {
		t.Errorf("Level1.Level2.Y: expect %d (should not be overwritten), got %d", 100, v.Level1.Level2.Y)
	}
	// Level1.X and Level3.Z are zero → should be set.
	if v.Level1.X != "l1_x" {
		t.Errorf("Level1.X: expect %q, got %q", "l1_x", v.Level1.X)
	}
	if !v.Level1.Level2.Level3.Z {
		t.Errorf("Level1.Level2.Level3.Z: expect true, got false")
	}
	if v.Top != "top_val" {
		t.Errorf("Top: expect %q, got %q", "top_val", v.Top)
	}
}

// Named pointer-to-struct field expanded into sub-fields.
type testDefaultNamedPtrInner struct {
	Val int `default:"77"`
}

type testDefaultNamedPtrOuter struct {
	Inner *testDefaultNamedPtrInner
	Tag   string `default:"ptr_outer_tag"`
}

func TestSetDefaultRecurseNamedPointerStructField(t *testing.T) {
	v := &testDefaultNamedPtrOuter{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Inner == nil {
		t.Fatal("Inner should be auto-allocated for sub-field defaults")
	}
	if v.Inner.Val != 77 {
		t.Errorf("Inner.Val: expect %d, got %d", 77, v.Inner.Val)
	}
	if v.Tag != "ptr_outer_tag" {
		t.Errorf("Tag: expect %q, got %q", "ptr_outer_tag", v.Tag)
	}
}

// ============================================================================
//  Exported named struct field — additional coverage
// ============================================================================

type testDefaultNamedSub struct {
	SubName string `default:"sub_name_val"`
	SubAge  int    `default:"30"`
}

type testDefaultNamedParent struct {
	Child       testDefaultNamedSub
	ParentLabel string `default:"parent_label_val"`
}

// TestSetDefaultNamedStructFieldWholePreSet: the entire named struct field is
// already non-zero → sub-field defaults should NOT be applied.
func TestSetDefaultNamedStructFieldWholePreSet(t *testing.T) {
	v := &testDefaultNamedParent{
		Child: testDefaultNamedSub{
			SubName: "preset_name",
			SubAge:  99,
		},
	}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Child.SubName != "preset_name" {
		t.Errorf("SubName: expect %q (should not be overwritten), got %q", "preset_name", v.Child.SubName)
	}
	if v.Child.SubAge != 99 {
		t.Errorf("SubAge: expect %d (should not be overwritten), got %d", 99, v.Child.SubAge)
	}
	if v.ParentLabel != "parent_label_val" {
		t.Errorf("ParentLabel: expect %q, got %q", "parent_label_val", v.ParentLabel)
	}
}

// TestSetDefaultNamedStructFieldPartialSet: some sub-fields are already set,
// others are zero. Only zero sub-fields should receive defaults.
func TestSetDefaultNamedStructFieldPartialSet(t *testing.T) {
	v := &testDefaultNamedParent{}
	v.Child.SubName = "custom_name"
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	// SubName is non-zero → should NOT be overwritten.
	if v.Child.SubName != "custom_name" {
		t.Errorf("SubName: expect %q (should not be overwritten), got %q", "custom_name", v.Child.SubName)
	}
	// SubAge is zero → should be set.
	if v.Child.SubAge != 30 {
		t.Errorf("SubAge: expect %d, got %d", 30, v.Child.SubAge)
	}
	if v.ParentLabel != "parent_label_val" {
		t.Errorf("ParentLabel: expect %q, got %q", "parent_label_val", v.ParentLabel)
	}
}

// TestSetDefaultEmptyNamedStruct: a named struct field whose inner sub-fields
// have no "default" tags at all should leave everything at zero values.
func TestSetDefaultEmptyNamedStruct(t *testing.T) {
	type emptyInner struct {
		A string
		B int
	}
	type emptyOuter struct {
		Inner emptyInner
		C     string `default:"outer_c"`
	}

	v := &emptyOuter{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Inner.A != "" {
		t.Errorf("Inner.A: expect empty, got %q", v.Inner.A)
	}
	if v.Inner.B != 0 {
		t.Errorf("Inner.B: expect 0, got %d", v.Inner.B)
	}
	if v.C != "outer_c" {
		t.Errorf("C: expect %q, got %q", "outer_c", v.C)
	}
}

// TestSetDefaultDoubleNamedStruct: an outer named struct field that itself
// contains another named struct field, testing multi-level named expansion.
func TestSetDefaultDoubleNamedStruct(t *testing.T) {
	type grandchild struct {
		Val string `default:"grandchild_val"`
	}
	type child struct {
		Gc   grandchild
		Name string `default:"child_name"`
	}
	type parent struct {
		Ch  child
		Top string `default:"parent_top"`
	}

	v := &parent{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.Ch.Gc.Val != "grandchild_val" {
		t.Errorf("Ch.Gc.Val: expect %q, got %q", "grandchild_val", v.Ch.Gc.Val)
	}
	if v.Ch.Name != "child_name" {
		t.Errorf("Ch.Name: expect %q, got %q", "child_name", v.Ch.Name)
	}
	if v.Top != "parent_top" {
		t.Errorf("Top: expect %q, got %q", "parent_top", v.Top)
	}
}

// ============================================================================
//  Large struct (many fields) — stress the parser
// ============================================================================

type testDefaultLarge struct {
	F01 string `default:"v01"`
	F02 string `default:"v02"`
	F03 string `default:"v03"`
	F04 string `default:"v04"`
	F05 string `default:"v05"`
	F06 string `default:"v06"`
	F07 string `default:"v07"`
	F08 string `default:"v08"`
	F09 string `default:"v09"`
	F10 string `default:"v10"`
	F11 int    `default:"11"`
	F12 int    `default:"12"`
	F13 int    `default:"13"`
	F14 int    `default:"14"`
	F15 int    `default:"15"`
}

func TestSetDefaultLargeStruct(t *testing.T) {
	v := &testDefaultLarge{}
	if err := SetDefault(v); err != nil {
		t.Fatal(err)
	}
	if v.F01 != "v01" || v.F05 != "v05" || v.F10 != "v10" || v.F15 != 15 {
		t.Errorf("large struct defaults not set correctly: %+v", v)
	}
}
