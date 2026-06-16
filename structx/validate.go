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
	"strings"

	"github.com/xgfone/go-toolkit/internal/structs"
)

// ValidateAny validates the exported fields of v tagged with "validate".
//
// v must be a struct or a pointer to struct. A nil value or nil struct pointer
// is ignored. Nested struct fields are expanded unless the struct field itself
// has a "validate" tag, in which case the whole field is validated. Nil
// intermediate struct pointers are not allocated and their child fields are
// skipped.
//
// validateField is called with each field value and its validate tag value.
// The field value is passed as reflect.Value so callers can choose whether
// and when to convert it to an interface value.
//
// ValidateAny stops at the first validation error and wraps it with the field name,
// preferring json, form, yaml, query, then header tag names before the Go field name.
func ValidateAny(v any, validateField func(fieldValue reflect.Value, rule string) error) (err error) {
	if v == nil {
		return nil
	}

	if validateField == nil {
		panic("field validate function is nil")
	}

	var root reflect.Value
	rtype := reflect.TypeOf(v)
	switch rtype.Kind() {
	case reflect.Pointer:
		root = reflect.ValueOf(v)
		if root.IsNil() {
			return nil
		}

		rtype = rtype.Elem()
		if rtype.Kind() != reflect.Struct {
			return errors.New("Validate: not a pointer to struct")
		}

		root = root.Elem()

	case reflect.Struct:
		root = reflect.ValueOf(v)

	default:
		return errors.New("Validate: not a struct or pointer to struct")
	}

	return _FieldRuleValidateFunc(validateField).Validate(rtype, root)
}

type _FieldRuleValidateFunc func(fieldValue reflect.Value, rule string) error

func (v _FieldRuleValidateFunc) Validate(rtype reflect.Type, root reflect.Value) error {
	for _, f := range validateParser.Parse(rtype, "").Fields {
		if f.Data.Rule == "" {
			continue
		}

		rvalue := structs.GetFieldByIndex(root, f.Indexes, false)
		if !rvalue.IsValid() { // The field is nil pointer and not allocated.
			continue
		}

		if err := v(rvalue, f.Data.Rule); err != nil {
			return fmt.Errorf("%q: %w", f.Data.Name, err)
		}
	}
	return nil
}

var validateParser = structs.NewParser(validatorCompileField, validatorIsOpaque)

type _ValidateData struct {
	Rule string
	Name string
}

func validatorCompileField(sf reflect.StructField) _ValidateData {
	return _ValidateData{
		Rule: sf.Tag.Get("validate"),
		Name: getValidateFieldName(sf),
	}
}

func validatorIsOpaque(sf reflect.StructField) bool {
	return sf.Tag.Get("validate") != ""
}

var validateFieldNameTags = []string{"json", "form", "yaml", "query", "header"}

func getValidateFieldName(field reflect.StructField) string {
	for _, tag := range validateFieldNameTags {
		if name := extractFieldName(field.Tag.Get(tag)); name != "" {
			return name
		}
	}
	return field.Name
}

func extractFieldName(name string) string {
	if name == "" {
		return ""
	}

	name, _, _ = strings.Cut(name, ",")
	return strings.TrimSpace(name)
}
