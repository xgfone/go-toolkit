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

// Package anysetter provides a field setter compiler for any.
package anysetter

import (
	"encoding"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/xgfone/go-toolkit/reflectx"
	"github.com/xgfone/go-toolkit/unsafex"
)

var binderType = reflect.TypeFor[_Binder]()
var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

type _Binder interface {
	Bind(any) error
}

func bind(v reflect.Value, s any) error {
	return v.Interface().(_Binder).Bind(s)
}

type SetterFunc func(t reflect.Type, dst reflect.Value, src any) error

func Compile(t reflect.Type) SetterFunc {
	if t.Kind() == reflect.Pointer {
		return compilePointer(t)
	}

	if reflectx.Implements(reflect.PointerTo(t), binderType) {
		return setValueInterface
	}
	return compileSetter(t)
}

func compilePointer(t reflect.Type) SetterFunc {
	if reflectx.Implements(t, binderType) {
		return setPointerInterface
	}

	elemSetter := compileSetter(t.Elem())
	return func(t reflect.Type, v reflect.Value, s any) (err error) {
		if s == nil {
			return unsupportedValueType(t, s)
		}

		if setAssignable(t, v, s) {
			return nil
		}

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

func setPointerInterface(t reflect.Type, v reflect.Value, s any) (err error) {
	if !v.IsNil() {
		return bind(v, s)
	}

	tmp := reflect.New(t.Elem())
	if err = bind(tmp, s); err == nil {
		v.Set(tmp)
	}
	return
}

func setValueInterface(_ reflect.Type, v reflect.Value, s any) error {
	return bind(v.Addr(), s)
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

	case reflect.Slice:
		return compileSlice(t)

	case reflect.Map:
		return compileMap(t)
	}
	return setAny
}

func setAny(t reflect.Type, dst reflect.Value, src any) error {
	if src == nil {
		return unsupportedValueType(t, src)
	}

	if setAssignable(t, dst, src) {
		return nil
	}

	if t.Kind() == reflect.Struct && reflectx.Implements(reflect.PointerTo(t), textUnmarshalerType) {
		if b, ok := textBytes(src); ok {
			return dst.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText(b)
		}
	}

	return unsupportedValueType(t, src)
}

func setAssignable(t reflect.Type, dst reflect.Value, src any) (ok bool) {
	sv := reflect.ValueOf(src)
	if ok = sv.Type().AssignableTo(t); ok {
		dst.Set(sv)
	}
	return
}

func compileSlice(t reflect.Type) SetterFunc {
	elemType := t.Elem()
	elemSetter := Compile(elemType)
	return func(t reflect.Type, dst reflect.Value, src any) error {
		return setSlice(t, elemType, elemSetter, dst, src)
	}
}

func compileMap(t reflect.Type) SetterFunc {
	keyType := t.Key()
	elemType := t.Elem()
	keySetter := Compile(keyType)
	elemSetter := Compile(elemType)
	return func(t reflect.Type, dst reflect.Value, src any) error {
		return setMap(t, keyType, elemType, keySetter, elemSetter, dst, src)
	}
}

func textBytes(src any) ([]byte, bool) {
	if src == nil {
		return nil, false
	}

	switch v := src.(type) {
	case string:
		return unsafex.Bytes(v), true

	case []byte:
		return v, true
	}

	switch v := reflect.ValueOf(src); v.Kind() {
	case reflect.String:
		return unsafex.Bytes(v.String()), true

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return v.Bytes(), true
		}
	}

	return nil, false
}

func setString(t reflect.Type, dst reflect.Value, src any) (err error) {
	switch v := src.(type) {
	case bool:
		if v {
			dst.SetString("true")
		} else {
			dst.SetString("false")
		}

	case int:
		dst.SetString(strconv.FormatInt(int64(v), 10))

	case int8:
		dst.SetString(strconv.FormatInt(int64(v), 10))

	case int16:
		dst.SetString(strconv.FormatInt(int64(v), 10))

	case int32:
		dst.SetString(strconv.FormatInt(int64(v), 10))

	case int64:
		dst.SetString(strconv.FormatInt(v, 10))

	case uint:
		dst.SetString(strconv.FormatUint(uint64(v), 10))

	case uint8:
		dst.SetString(strconv.FormatUint(uint64(v), 10))

	case uint16:
		dst.SetString(strconv.FormatUint(uint64(v), 10))

	case uint32:
		dst.SetString(strconv.FormatUint(uint64(v), 10))

	case uint64:
		dst.SetString(strconv.FormatUint(v, 10))

	case float32:
		dst.SetString(strconv.FormatFloat(float64(v), 'f', -1, 32))

	case float64:
		dst.SetString(strconv.FormatFloat(v, 'f', -1, 64))

	case string:
		dst.SetString(v)

	case time.Time:
		dst.SetString(v.Format(time.RFC3339))

	case fmt.Stringer:
		dst.SetString(v.String())

	case error:
		dst.SetString(v.Error())

	default:
		if err = setStringReflect(dst, src); err != nil {
			err = unsupportedValue(dst, src)
		}
	}
	return
}

func setStringReflect(dst reflect.Value, src any) error {
	if src == nil {
		return unsupportedValue(dst, src)
	}

	switch v := reflect.ValueOf(src); v.Kind() {
	case reflect.Bool:
		dst.SetString(strconv.FormatBool(v.Bool()))

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst.SetString(strconv.FormatInt(v.Int(), 10))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		dst.SetString(strconv.FormatUint(v.Uint(), 10))

	case reflect.Float32:
		dst.SetString(strconv.FormatFloat(v.Float(), 'f', -1, 32))

	case reflect.Float64:
		dst.SetString(strconv.FormatFloat(v.Float(), 'f', -1, 64))

	case reflect.String:
		dst.SetString(v.String())

	default:
		return unsupportedValue(dst, src)
	}
	return nil
}

func setBool(t reflect.Type, dst reflect.Value, src any) (err error) {
	b, err := boolValue(t, src)
	if err == nil {
		dst.SetBool(b)
	}
	return
}

func setInt(t reflect.Type, dst reflect.Value, src any) (err error) {
	n, err := intValue(t, src)
	if err == nil {
		dst.SetInt(n)
	}
	return
}

func setUint(t reflect.Type, dst reflect.Value, src any) (err error) {
	n, err := uintValue(t, src)
	if err == nil {
		dst.SetUint(n)
	}
	return
}

func setFloat(t reflect.Type, dst reflect.Value, src any) (err error) {
	f, err := floatValue(t, src)
	if err == nil {
		dst.SetFloat(f)
	}
	return
}

func setSlice(t, elemType reflect.Type, elemSetter SetterFunc, dst reflect.Value, src any) error {
	if src == nil {
		return unsupportedValue(dst, src)
	}

	sv := reflect.ValueOf(src)
	if sv.Type().AssignableTo(t) {
		dst.Set(sv)
		return nil
	}

	switch sv.Kind() {
	case reflect.Array:
	case reflect.Slice:
		if sv.IsNil() {
			dst.Set(reflect.Zero(t))
			return nil
		}
	default:
		return unsupportedValue(dst, src)
	}

	tmp := reflect.MakeSlice(t, sv.Len(), sv.Len())
	for i := range sv.Len() {
		if err := elemSetter(elemType, tmp.Index(i), sv.Index(i).Interface()); err != nil {
			return err
		}
	}

	dst.Set(tmp)
	return nil
}

func setMap(
	t, keyType, elemType reflect.Type,
	keySetter, elemSetter SetterFunc,
	dst reflect.Value, src any,
) error {
	if src == nil {
		return unsupportedValue(dst, src)
	}

	sv := reflect.ValueOf(src)
	if sv.Type().AssignableTo(t) {
		dst.Set(sv)
		return nil
	}
	if sv.Kind() != reflect.Map {
		return unsupportedValue(dst, src)
	}
	if sv.IsNil() {
		dst.Set(reflect.Zero(t))
		return nil
	}

	tmp := reflect.MakeMapWithSize(t, sv.Len())
	iter := sv.MapRange()
	for iter.Next() {
		key := reflect.New(keyType).Elem()
		if err := keySetter(keyType, key, iter.Key().Interface()); err != nil {
			return err
		}

		elem := reflect.New(elemType).Elem()
		if err := elemSetter(elemType, elem, iter.Value().Interface()); err != nil {
			return err
		}

		if tmp.MapIndex(key).IsValid() {
			return duplicateMapKey(t, key.Interface())
		}
		tmp.SetMapIndex(key, elem)
	}

	dst.Set(tmp)
	return nil
}

func boolValue(t reflect.Type, src any) (bool, error) {
	switch v := src.(type) {
	case bool:
		return v, nil

	case int:
		return v != 0, nil

	case int8:
		return v != 0, nil

	case int16:
		return v != 0, nil

	case int32:
		return v != 0, nil

	case int64:
		return v != 0, nil

	case uint:
		return v != 0, nil

	case uint8:
		return v != 0, nil

	case uint16:
		return v != 0, nil

	case uint32:
		return v != 0, nil

	case uint64:
		return v != 0, nil

	case uintptr:
		return v != 0, nil

	case float32:
		return v != 0, nil

	case float64:
		return v != 0, nil

	case string:
		return strconv.ParseBool(v)

	default:
		return boolValueReflect(t, src)
	}
}

func boolValueReflect(t reflect.Type, src any) (bool, error) {
	if src == nil {
		return false, unsupportedValueType(t, src)
	}

	switch v := reflect.ValueOf(src); v.Kind() {
	case reflect.Bool:
		return v.Bool(), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0, nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() != 0, nil

	case reflect.Float32, reflect.Float64:
		return v.Float() != 0, nil

	case reflect.String:
		return strconv.ParseBool(v.String())
	}
	return false, unsupportedValueType(t, src)
}

func intValue(t reflect.Type, src any) (int64, error) {
	switch v := src.(type) {
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil

	case int:
		return intValueFromInt64(t, int64(v), src)

	case int8:
		return intValueFromInt64(t, int64(v), src)

	case int16:
		return intValueFromInt64(t, int64(v), src)

	case int32:
		return intValueFromInt64(t, int64(v), src)

	case int64:
		return intValueFromInt64(t, v, src)

	case uint:
		return intValueFromUint64(t, uint64(v), src)

	case uint8:
		return intValueFromUint64(t, uint64(v), src)

	case uint16:
		return intValueFromUint64(t, uint64(v), src)

	case uint32:
		return intValueFromUint64(t, uint64(v), src)

	case uint64:
		return intValueFromUint64(t, v, src)

	case uintptr:
		return intValueFromUint64(t, uint64(v), src)

	case float32:
		return floatToInt(t, float64(v), src)

	case float64:
		return floatToInt(t, v, src)

	case string:
		return strconv.ParseInt(v, 10, t.Bits())

	default:
		return intValueReflect(t, src)
	}
}

func intValueReflect(t reflect.Type, src any) (int64, error) {
	if src == nil {
		return 0, unsupportedValueType(t, src)
	}

	switch v := reflect.ValueOf(src); v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return 1, nil
		}
		return 0, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intValueFromInt64(t, v.Int(), src)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return intValueFromUint64(t, v.Uint(), src)

	case reflect.Float32, reflect.Float64:
		return floatToInt(t, v.Float(), src)

	case reflect.String:
		return strconv.ParseInt(v.String(), 10, t.Bits())
	}
	return 0, unsupportedValueType(t, src)
}

func intValueFromInt64(t reflect.Type, n int64, src any) (int64, error) {
	if dstOverflowsInt(t, n) {
		return 0, overflowValue(t, src)
	}
	return n, nil
}

func intValueFromUint64(t reflect.Type, n uint64, src any) (int64, error) {
	if uintOverflowsInt(t, n) {
		return 0, overflowValue(t, src)
	}
	return int64(n), nil
}

func uintValue(t reflect.Type, src any) (uint64, error) {
	switch v := src.(type) {
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil

	case int:
		return uintValueFromInt64(t, int64(v), src)

	case int8:
		return uintValueFromInt64(t, int64(v), src)

	case int16:
		return uintValueFromInt64(t, int64(v), src)

	case int32:
		return uintValueFromInt64(t, int64(v), src)

	case int64:
		return uintValueFromInt64(t, v, src)

	case uint:
		return uintValueFromUint64(t, uint64(v), src)

	case uint8:
		return uintValueFromUint64(t, uint64(v), src)

	case uint16:
		return uintValueFromUint64(t, uint64(v), src)

	case uint32:
		return uintValueFromUint64(t, uint64(v), src)

	case uint64:
		return uintValueFromUint64(t, v, src)

	case uintptr:
		return uintValueFromUint64(t, uint64(v), src)

	case float32:
		return floatToUint(t, float64(v), src)

	case float64:
		return floatToUint(t, v, src)

	case string:
		return strconv.ParseUint(v, 10, t.Bits())

	default:
		return uintValueReflect(t, src)
	}
}

func uintValueReflect(t reflect.Type, src any) (uint64, error) {
	if src == nil {
		return 0, unsupportedValueType(t, src)
	}

	switch v := reflect.ValueOf(src); v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			return 1, nil
		}
		return 0, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uintValueFromInt64(t, v.Int(), src)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintValueFromUint64(t, v.Uint(), src)

	case reflect.Float32, reflect.Float64:
		return floatToUint(t, v.Float(), src)

	case reflect.String:
		return strconv.ParseUint(v.String(), 10, t.Bits())
	}
	return 0, unsupportedValueType(t, src)
}

func uintValueFromInt64(t reflect.Type, n int64, src any) (uint64, error) {
	if n < 0 {
		return 0, overflowValue(t, src)
	}
	return uintValueFromUint64(t, uint64(n), src)
}

func uintValueFromUint64(t reflect.Type, n uint64, src any) (uint64, error) {
	if dstOverflowsUint(t, n) {
		return 0, overflowValue(t, src)
	}
	return n, nil
}

func floatValue(t reflect.Type, src any) (float64, error) {
	var f float64
	switch v := src.(type) {
	case bool:
		if v {
			f = 1
		}

	case int:
		f = float64(v)

	case int8:
		f = float64(v)

	case int16:
		f = float64(v)

	case int32:
		f = float64(v)

	case int64:
		f = float64(v)

	case uint:
		f = float64(v)

	case uint8:
		f = float64(v)

	case uint16:
		f = float64(v)

	case uint32:
		f = float64(v)

	case uint64:
		f = float64(v)

	case uintptr:
		f = float64(v)

	case float32:
		f = float64(v)

	case float64:
		f = v

	case string:
		return strconv.ParseFloat(v, t.Bits())

	default:
		return floatValueReflect(t, src)
	}
	return checkedFloatValue(t, f, src)
}

func floatValueReflect(t reflect.Type, src any) (float64, error) {
	if src == nil {
		return 0, unsupportedValueType(t, src)
	}

	var f float64
	switch v := reflect.ValueOf(src); v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			f = 1
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f = float64(v.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		f = float64(v.Uint())

	case reflect.Float32, reflect.Float64:
		f = v.Float()

	case reflect.String:
		return strconv.ParseFloat(v.String(), t.Bits())

	default:
		return 0, unsupportedValueType(t, src)
	}
	return checkedFloatValue(t, f, src)
}

func checkedFloatValue(t reflect.Type, f float64, src any) (float64, error) {
	if dstOverflowsFloat(t, f) {
		return 0, overflowValue(t, src)
	}
	return f, nil
}

func floatToInt(t reflect.Type, f float64, src any) (int64, error) {
	bits := t.Bits()
	if !validIntegerFloat(f) || f < float64(minInt(bits)) {
		return 0, overflowValue(t, src)
	}

	if bits == 64 {
		if f >= math.Ldexp(1, 63) {
			return 0, overflowValue(t, src)
		}
	} else if f > float64(maxInt(bits)) {
		return 0, overflowValue(t, src)
	}

	return int64(f), nil
}

func floatToUint(t reflect.Type, f float64, src any) (uint64, error) {
	bits := t.Bits()
	if !validIntegerFloat(f) || f < 0 {
		return 0, overflowValue(t, src)
	}

	if bits == 64 {
		if f >= math.Ldexp(1, 64) {
			return 0, overflowValue(t, src)
		}
	} else if f > float64(maxUint(bits)) {
		return 0, overflowValue(t, src)
	}

	return uint64(f), nil
}

func validIntegerFloat(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0) && math.Trunc(f) == f
}

func dstOverflowsInt(t reflect.Type, n int64) bool {
	return n < minInt(t.Bits()) || n > maxInt(t.Bits())
}

func uintOverflowsInt(t reflect.Type, n uint64) bool {
	return n > uint64(maxInt(t.Bits()))
}

func dstOverflowsUint(t reflect.Type, n uint64) bool {
	return n > maxUint(t.Bits())
}

func dstOverflowsFloat(t reflect.Type, f float64) bool {
	return t.Kind() == reflect.Float32 && !math.IsInf(f, 0) && !math.IsNaN(f) && math.Abs(f) > math.MaxFloat32
}

func minInt(bits int) int64 {
	return -1 << (bits - 1)
}

func maxInt(bits int) int64 {
	return 1<<(bits-1) - 1
}

func maxUint(bits int) uint64 {
	if bits == 64 {
		return math.MaxUint64
	}
	return 1<<bits - 1
}

func overflowValue(t reflect.Type, src any) error {
	return fmt.Errorf("value %v overflows field type %s", src, t)
}

func unsupportedValue(dst reflect.Value, src any) error {
	return unsupportedValueType(dst.Type(), src)
}

func unsupportedValueType(t reflect.Type, src any) error {
	return fmt.Errorf("not support to set %T to field type %s", src, t)
}

func duplicateMapKey(t reflect.Type, key any) error {
	return fmt.Errorf("duplicate converted map key %v for field type %s", key, t)
}
