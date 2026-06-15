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
	"encoding"
	"reflect"

	"github.com/xgfone/go-toolkit/internal/structs/anysetter"
	"github.com/xgfone/go-toolkit/internal/structs/strsetter"
	"github.com/xgfone/go-toolkit/reflectx"
)

type SetterFunc[T any] func(t reflect.Type, dst reflect.Value, src T) error

type FieldSetter[T any] struct {
	SetField SetterFunc[T]
	TagValue string
}

func NewSetterFieldCompiler[T any](compile func(reflect.Type) SetterFunc[T], tag string) FieldCompiler[FieldSetter[T]] {
	return func(sf reflect.StructField) FieldSetter[T] {
		var value string
		if tag != "" {
			value = sf.Tag.Get(tag)
		}
		return FieldSetter[T]{SetField: compile(sf.Type), TagValue: value}
	}
}

/// ----------------------------------------------------------------------- ///

func NewAnySetterParser(valuetag string) *Parser[FieldSetter[any]] {
	return NewParser(NewSetterFieldCompiler(CompileAnySetter, valuetag), isAnyOpaqueField)
}

func NewStringSetterParser(valuetag string) *Parser[FieldSetter[string]] {
	return NewParser(NewSetterFieldCompiler(CompileStringSetter, valuetag), isStringOpaqueField)
}

func CompileStringSetter(t reflect.Type) SetterFunc[string] {
	return SetterFunc[string](strsetter.Compile(t))
}

func CompileAnySetter(t reflect.Type) SetterFunc[any] {
	return SetterFunc[any](anysetter.Compile(t))
}

var (
	binderType          = reflect.TypeFor[interface{ Bind(any) error }]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
)

func isAnyOpaqueField(sf reflect.StructField) bool {
	return hasImplemented(sf.Type, binderType)
}

func isStringOpaqueField(sf reflect.StructField) bool {
	return hasImplemented(sf.Type, textUnmarshalerType)
}

func hasImplemented(fieldType, ifaceType reflect.Type) bool {
	if fieldType.Kind() != reflect.Pointer {
		fieldType = reflect.PointerTo(fieldType)
	}
	return reflectx.Implements(fieldType, ifaceType)
}
