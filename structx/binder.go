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
	"reflect"

	"github.com/xgfone/go-toolkit/internal/structs"
	"github.com/xgfone/go-toolkit/mapx"
)

var (
	anySetParser = structs.NewAnySetterParser("")
	strSetParser = structs.NewStringSetterParser("")
)

// BindStringMap converts map[string]string to a value getter
// and uses BindValues to bind map[string]string into dst.
func BindStringMap[Struct any, Map ~map[string]string](dst *Struct, src Map, tag string) error {
	return BindValues(dst, mapx.SMap[string](src), tag)
}

// BindValuesAny is like BindValues, but accepts dst as a value of type any.
//
// dst must be a non-nil pointer to struct.
func BindValuesAny(dst any, src interface{ Get(string) string }, tag string) error {
	rtype, root, err := anyStructPtr(dst)
	if err != nil {
		return err
	}
	return bindValues(rtype, root, src, tag)
}

// BindValues binds flat string values from src into dst based on the field tag name.
//
// BindValues walks all exported fields of the destination struct type T. For
// each field, it resolves the src name from the given tag. If the tag value
// is empty, the field name itself is used. A tag value of "-" skips the field.
// Struct-typed fields are recursively parsed when their immediate type has
// direct exported fields and they are either anonymous embedded fields or
// exported named fields.
// Add the "opaque" tag option to stop recursive parsing and bind the struct
// field as a single field instead, for example `q:"field,opaque"`.
//
// When src.Get(name) returns a non-empty string, BindValues converts that
// string and assigns it to the field. Empty strings are ignored.
//
// A field whose type is T or *T may customize string binding by making *T
// implement encoding.TextUnmarshaler. Pointer-to-pointer fields are not
// supported. Nested struct expansion supports struct values and single
// pointers to structs only. Self-referential struct graphs are not cycle
// detected; mark recursive fields as opaque or bind them through
// encoding.TextUnmarshaler.
//
// If a struct field implements encoding.TextUnmarshaler through its pointer
// type, it is bound as a whole and is not recursively expanded.
//
// BindValues treats a missing src value and an explicitly provided empty
// string identically, because src exposes only Get(string) string.
//
// BindValues is a flat binding API. When nested struct fields are expanded,
// each expanded field reads from its leaf field name, not from the full nested
// path. If multiple expanded fields have the same leaf name, they read the same
// source key. Use BindMap when path-aware nested binding is required.
func BindValues[Struct any, Getter interface{ Get(string) string }](dst *Struct, src Getter, tag string) error {
	if dst == nil {
		return errors.New("dst is nil")
	}

	rtype := reflect.TypeFor[Struct]()
	if rtype.Kind() != reflect.Struct {
		return errors.New("dst is not a pointer to struct")
	}

	root := reflect.ValueOf(dst).Elem()
	return bindValues(rtype, root, src, tag)
}

func bindValues(rtype reflect.Type, root reflect.Value, src interface{ Get(string) string }, tag string) error {
	for _, f := range strSetParser.Parse(rtype, tag).Fields {
		s := src.Get(f.Name)
		if s == "" {
			continue
		}

		if err := f.Data.SetField(f.Type, f.GetField(root), s); err != nil {
			return fmt.Errorf("%s: %w", f.Name, err)
		}
	}

	return nil
}

// BindMapAny is like BindMap, but accepts dst as a value of type any.
//
// dst must be a non-nil pointer to struct.
func BindMapAny(dst any, src map[string]any, tag string) error {
	rtype, root, err := anyStructPtr(dst)
	if err != nil {
		return err
	}
	return bindMap(rtype, root, src, tag)
}

// BindMap binds values from src into dst based on the field tag name.
//
// BindMap walks all exported fields of the destination struct type T. For each
// field, it resolves the src name from the given tag. If the tag value is
// empty, the field name itself is used. A tag value of "-" skips the field.
// Struct-typed fields are recursively parsed when their immediate type has
// direct exported fields and they are either anonymous embedded fields or
// exported named fields.
// Add the "opaque" tag option to stop recursive parsing and bind the struct
// field as a single field instead, for example `json:"field,opaque"`.
//
// A field whose type is T or *T may customize any-value binding by making *T
// implement interface{ Bind(any) error }. Pointer-to-pointer fields are not
// supported. Nested struct expansion supports struct values and single
// pointers to structs only. Self-referential struct graphs are not cycle
// detected; mark recursive fields as opaque or bind them through
// interface{ Bind(any) error }.
//
// If a struct field implements interface{ Bind(any) error } through its
// pointer type, it is bound as a whole and is not recursively expanded.
// Non-expanded struct fields whose pointer type implements
// encoding.TextUnmarshaler may also be assigned from string, []byte,
// or named types whose underlying kind is string or []byte.
//
// For nested struct fields, BindMap follows the parsed field path through
// nested map[string]any values. Named map types are accepted only for the
// top-level src argument; nested maps must have the exact type map[string]any.
//
// A nil or missing src value is ignored. But a typed nil value,
// such as (*T)(nil), is still treated as an explicit value
// because the interface value itself is non-nil.
func BindMap[Struct any, Map ~map[string]any](dst *Struct, src Map, tag string) error {
	if dst == nil {
		return errors.New("dst is nil")
	}

	rtype := reflect.TypeFor[Struct]()
	if rtype.Kind() != reflect.Struct {
		return errors.New("dst is not a pointer to struct")
	}

	root := reflect.ValueOf(dst).Elem()
	return bindMap(rtype, root, src, tag)
}

func bindMap(rtype reflect.Type, root reflect.Value, src map[string]any, tag string) (err error) {
	for _, f := range anySetParser.Parse(rtype, tag).Fields {
		value := f.GetValue(src)
		if value == nil {
			continue
		}

		if err = f.Data.SetField(f.Type, f.GetField(root), value); err != nil {
			return fmt.Errorf("%s: %w", f.Name, err)
		}
	}
	return
}
