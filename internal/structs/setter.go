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
	"fmt"
	"reflect"
	"strconv"

	"github.com/xgfone/go-toolkit/unsafex"
)

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

func unmarshalText(v reflect.Value, s string) error {
	return v.Interface().(encoding.TextUnmarshaler).UnmarshalText(unsafex.Bytes(s))
}

type SetterFunc func(t reflect.Type, dst reflect.Value, src string) error

func CompileSetter(t reflect.Type) SetterFunc {
	if t.Kind() == reflect.Pointer {
		return compileSetterPointer(t)
	}

	if reflect.PointerTo(t).Implements(textUnmarshalerType) {
		return setValueInterface
	}
	return compileSetter(t)
}

func compileSetterPointer(t reflect.Type) SetterFunc {
	if t.Implements(textUnmarshalerType) {
		return setPointerInterface
	}

	elemSetter := compileSetter(t.Elem())
	return func(t reflect.Type, v reflect.Value, s string) (err error) {
		if !v.IsNil() {
			return elemSetter(t.Elem(), v.Elem(), s)
		}

		tmp := reflect.New(t.Elem())
		if err = elemSetter(t.Elem(), tmp.Elem(), s); err == nil {
			v.Set(tmp)
		}
		return
	}
}

func setPointerInterface(t reflect.Type, v reflect.Value, s string) (err error) {
	if !v.IsNil() {
		return unmarshalText(v, s)
	}

	tmp := reflect.New(t.Elem())
	if err = unmarshalText(tmp, s); err == nil {
		v.Set(tmp)
	}
	return
}

func setValueInterface(_ reflect.Type, v reflect.Value, s string) error {
	return unmarshalText(v.Addr(), s)
}

func compileSetter(t reflect.Type) SetterFunc {
	switch t.Kind() {
	case reflect.String:
		return setString

	case reflect.Bool:
		return setBool

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return setInt

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return setUint

	case reflect.Float32, reflect.Float64:
		return setFloat

	default:
		return unsupportedType
	}
}

func setString(_ reflect.Type, v reflect.Value, s string) error {
	v.SetString(s)
	return nil
}

func setBool(_ reflect.Type, v reflect.Value, s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}

	v.SetBool(b)
	return nil
}

func setInt(t reflect.Type, v reflect.Value, s string) error {
	n, err := strconv.ParseInt(s, 10, t.Bits())
	if err != nil {
		return err
	}

	v.SetInt(n)
	return nil
}

func setUint(t reflect.Type, v reflect.Value, s string) error {
	n, err := strconv.ParseUint(s, 10, t.Bits())
	if err != nil {
		return err
	}

	v.SetUint(n)
	return nil
}

func setFloat(t reflect.Type, v reflect.Value, s string) error {
	f, err := strconv.ParseFloat(s, t.Bits())
	if err != nil {
		return err
	}

	v.SetFloat(f)
	return nil
}

func unsupportedType(_ reflect.Type, v reflect.Value, s string) error {
	return fmt.Errorf("unsupported field type %s", v.Type())
}
