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

// BindStringMap converts map[string]string to a value getter
// and uses BindValues to bind map[string]string into dst.
func BindStringMap[Struct any, Map ~map[string]string](dst *Struct, src Map, tag string) error {
	return BindValues(dst, mapx.SMap[string](src), tag)
}

// BindValues binds flat string values from src into dst based on the field tag name.
//
// BindValues walks all exported fields of the destination struct type T. For
// each field, it resolves the src name from the given tag. If the tag value
// is empty, the field name itself is used. A tag value of "-" skips the field.
// Struct-typed fields are recursively parsed when they have exported
// sub-fields and are either anonymous embedded fields or exported named fields.
// Add the "opaque" tag option to stop recursive parsing and bind the struct
// field as a single field instead, for example `q:"field,opaque"`.
//
// When src.Get(name) returns a non-empty string, BindValues converts that
// string and assigns it to the field. If src.Get(name) returns an empty
// string and the field declares a non-empty `default:"..."` tag value, the
// default value is used instead. Empty strings are otherwise ignored.
//
// A field whose type is T or *T may customize string binding by making *T
// implement encoding.TextUnmarshaler. Pointer-to-pointer fields are not
// supported.
//
// BindValues treats a missing src value and an explicitly provided empty
// string identically, because src exposes only Get(string) string.
func BindValues[Struct any, Getter interface{ Get(string) string }](dst *Struct, src Getter, tag string) error {
	if dst == nil {
		return errors.New("dst is nil")
	}

	rtype := reflect.TypeFor[Struct]()
	if rtype.Kind() != reflect.Struct {
		return errors.New("dst is not a pointer to struct")
	}

	root := reflect.ValueOf(dst).Elem()
	for _, f := range structs.StringParser.Parse(rtype, tag).Fields {
		s := src.Get(f.Name)
		if s == "" && f.Default != "" {
			s = f.Default
		}
		if s == "" {
			continue
		}

		if err := f.SetValue(root, s); err != nil {
			return fmt.Errorf("%q: %w", f.Name, err)
		}
	}

	return nil
}

// BindMap binds values from src into dst based on the field tag name.
//
// BindMap walks all exported fields of the destination struct type T. For each
// field, it resolves the src name from the given tag. If the tag value is
// empty, the field name itself is used. A tag value of "-" skips the field.
// Struct-typed fields are recursively parsed when they have exported
// sub-fields and are either anonymous embedded fields or exported named fields.
// Add the "opaque" tag option to stop recursive parsing and bind the struct
// field as a single field instead, for example `json:"field,opaque"`.
//
// A field whose type is T or *T may customize any-value binding by making *T
// implement interface{ Bind(any) error }. Pointer-to-pointer fields are not
// supported.
//
// For nested struct fields, BindMap follows the parsed field path through
// nested map[string]any values. A nil or missing src value is ignored.
// Non-nil values are converted and assigned using the any-value setter.
func BindMap[Struct any, Map ~map[string]any](dst *Struct, src Map, tag string) error {
	binder := mapBinder[Struct, Map]{
		parser: structs.AnyParser,

		dst: dst,
		src: src,
		tag: tag,
	}
	return binder.Bind()
}

type mapBinder[Struct any, Map ~map[string]any] struct {
	parser *structs.Parser[any]

	src Map
	dst *Struct
	tag string
}

func (b *mapBinder[Struct, Map]) Bind() (err error) {
	if b.dst == nil {
		return errors.New("dst is nil")
	}

	rtype := reflect.TypeFor[Struct]()
	if rtype.Kind() != reflect.Struct {
		return errors.New("dst is not a pointer to struct")
	}

	root := reflect.ValueOf(b.dst).Elem()
	for _, f := range b.parser.Parse(rtype, b.tag).Fields {
		value := f.GetValue(b.src)
		if value == nil {
			continue
		}

		if err = f.SetValue(root, value); err != nil {
			return fmt.Errorf("%q: %w", f.Name, err)
		}
	}
	return
}
