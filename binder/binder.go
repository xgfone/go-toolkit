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
	"fmt"
	"reflect"

	"github.com/xgfone/go-toolkit/internal/structs"
	"github.com/xgfone/go-toolkit/mapx"
)

// Getter is an interface for getting a string value by a string key.
//
// Return "" if the value is not found.
type Getter interface {
	Get(string) string
}

// BindSMap converts map[string]string to Getter and uses BindGetter
// to bind map[string]string into dst.
func BindSMap[M ~map[string]string, T any](src M, dst *T, tag string) error {
	return BindGetter(mapx.SMap[string](src), dst, tag)
}

// BindGetter binds values from src into dst based on the field tag name.
//
// BindGetter walks all exported fields of the destination struct type T. For
// each field, it resolves the source name from the given tag. If the tag value
// is empty, the field name itself is used. A tag value of "-" skips the field.
// For anonymous embedded struct fields, exported fields are flattened when the
// embedded field does not declare an explicit tag name.
//
// When src.Get(name) returns a non-empty string, BindGetter converts that
// string and assigns it to the field. If src.Get(name) returns an empty string
// and the field declares a non-empty `default:"..."` tag value, the default
// value is used instead. Empty strings are otherwise ignored.
//
// Supported destination field types are:
//   - string
//   - bool
//   - signed integer types
//   - unsigned integer types
//   - floating-point types
//   - pointers to the supported non-pointer types above
//   - value types whose pointer type implements encoding.TextUnmarshaler
//   - pointer types that implement encoding.TextUnmarshaler
//
// Pointer fields are allocated only when a non-empty source value or a
// non-empty default value is available. The new pointer value is assigned to
// the field only after conversion succeeds.
//
// Struct metadata is parsed once per (type, tag) pair and cached for reuse.
//
// BindGetter returns an error if dst is nil, if T is not a struct type, or if
// any field conversion fails. Conversion errors are wrapped with the field name.
//
// BindGetter does not distinguish between a missing source value and an
// explicitly provided empty string, because Getter exposes only Get(string)
// string. Multi-level pointers and unsupported field kinds are not supported.
func BindGetter[V Getter, T any](src V, dst *T, tag string) (err error) {
	if dst == nil {
		return errors.New("dst is nil")
	}

	rtype := reflect.TypeFor[T]()
	if rtype.Kind() != reflect.Struct {
		return errors.New("dst is not a pointer to struct")
	}

	root := reflect.ValueOf(dst).Elem()
	for _, f := range structs.Parse(rtype, tag).Fields {
		s := src.Get(f.Name)
		if s == "" && f.Default != "" {
			s = f.Default
		}
		if s == "" {
			continue
		}

		if err = f.SetValue(root, s); err != nil {
			return fmt.Errorf("%q: %w", f.Name, err)
		}
	}

	return
}
