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

package anysetter

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"
)

var errBadBind = errors.New("bad bind")

type (
	namedBool     bool
	namedInt      int64
	namedUint     uint64
	namedFloat64  float64
	namedFloat32  float32
	namedString   string
	namedBytes    []byte
	stringerValue string
	bindValue     string
	unsupported   complex64
)

func (s stringerValue) String() string { return "stringer:" + string(s) }

func (b *bindValue) Bind(v any) error {
	if v == "bad" {
		return errBadBind
	}

	*b = bindValue("bind:" + fmt.Sprint(v))
	return nil
}

func setValue[T any](src any) (dst T, err error) {
	t := reflect.TypeFor[T]()
	err = Compile(t)(t, reflect.ValueOf(&dst).Elem(), src)
	return
}

func setExisting[T any](dst *T, src any) error {
	t := reflect.TypeFor[T]()
	return Compile(t)(t, reflect.ValueOf(dst).Elem(), src)
}

func TestSetScalarValues(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"bool from bool", expectValue[bool](true, true)},
		{"bool from zero number", expectValue[bool](int8(0), false)},
		{"bool from non-zero named float", expectValue[bool](namedFloat64(-0.5), true)},
		{"bool from string", expectValue[bool]("true", true)},

		{"int from bool", expectValue[int](true, 1)},
		{"int8 from max string", expectValue[int8]("127", 127)},
		{"int16 from integral float", expectValue[int16](float64(12), 12)},
		{"int64 from named uint", expectValue[int64](namedUint(23), 23)},

		{"uint from bool", expectValue[uint](true, 1)},
		{"uint8 from max int", expectValue[uint8](int16(255), 255)},
		{"uint64 from named int", expectValue[uint64](namedInt(9), 9)},
		{"uintptr from string", expectValue[uintptr]("42", 42)},

		{"float32 from string", expectValue[float32]("1.5", 1.5)},
		{"float64 from bool", expectValue[float64](true, 1)},
		{"float64 from named int", expectValue[float64](namedInt(-7), -7)},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestScalarConversionMatrix(t *testing.T) {
	t.Run("to bool", func(t *testing.T) {
		for _, tt := range []struct {
			src  any
			want bool
		}{
			{true, true}, {int(-1), true}, {int8(0), false}, {int16(1), true},
			{int32(1), true}, {int64(1), true}, {uint(0), false}, {uint8(1), true},
			{uint16(1), true}, {uint32(1), true}, {uint64(1), true}, {uintptr(1), true},
			{float32(0), false}, {float64(-0.5), true}, {"false", false},
			{namedBool(true), true}, {namedInt(0), false}, {namedUint(1), true},
			{namedFloat64(1), true}, {namedString("true"), true},
		} {
			t.Run(typeName(tt.src), expectValue[bool](tt.src, tt.want))
		}
	})

	t.Run("to int64", func(t *testing.T) {
		for _, tt := range []struct {
			src  any
			want int64
		}{
			{true, 1}, {false, 0}, {int(-1), -1}, {int8(2), 2}, {int16(3), 3},
			{int32(4), 4}, {int64(5), 5}, {uint(6), 6}, {uint8(7), 7},
			{uint16(8), 8}, {uint32(9), 9}, {uint64(10), 10}, {uintptr(11), 11},
			{float32(12), 12}, {float64(13), 13}, {"14", 14},
			{namedBool(true), 1}, {namedBool(false), 0},
			{namedInt(15), 15}, {namedUint(16), 16},
			{namedFloat64(17), 17}, {namedString("18"), 18},
		} {
			t.Run(typeName(tt.src), expectValue[int64](tt.src, tt.want))
		}
	})

	t.Run("to uint64", func(t *testing.T) {
		for _, tt := range []struct {
			src  any
			want uint64
		}{
			{true, 1}, {false, 0}, {int(1), 1}, {int8(2), 2}, {int16(3), 3},
			{int32(4), 4}, {int64(5), 5}, {uint(6), 6}, {uint8(7), 7},
			{uint16(8), 8}, {uint32(9), 9}, {uint64(10), 10}, {uintptr(11), 11},
			{float32(12), 12}, {float64(13), 13}, {"14", 14},
			{namedBool(true), 1}, {namedBool(false), 0},
			{namedInt(15), 15}, {namedUint(16), 16},
			{namedFloat64(17), 17}, {namedString("18"), 18},
		} {
			t.Run(typeName(tt.src), expectValue[uint64](tt.src, tt.want))
		}
	})

	t.Run("to float64", func(t *testing.T) {
		for _, tt := range []struct {
			src  any
			want float64
		}{
			{true, 1}, {false, 0}, {int(-1), -1}, {int8(2), 2}, {int16(3), 3},
			{int32(4), 4}, {int64(5), 5}, {uint(6), 6}, {uint8(7), 7},
			{uint16(8), 8}, {uint32(9), 9}, {uint64(10), 10}, {uintptr(11), 11},
			{float32(12.5), 12.5}, {float64(13.5), 13.5}, {"14.5", 14.5},
			{namedBool(true), 1}, {namedInt(15), 15}, {namedUint(16), 16},
			{namedFloat64(17.5), 17.5}, {namedString("18.5"), 18.5},
		} {
			t.Run(typeName(tt.src), expectValue[float64](tt.src, tt.want))
		}
	})
}

func TestSetStringValues(t *testing.T) {
	ts := time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)
	tests := []struct {
		name string
		src  any
		want string
	}{
		{"bool", true, "true"},
		{"bool", false, "false"},
		{"int", int(-7), "-7"},
		{"int8", int8(-8), "-8"},
		{"int16", int16(-16), "-16"},
		{"int32", int32(-32), "-32"},
		{"int64", int64(-64), "-64"},
		{"uint", uint(7), "7"},
		{"uint8", uint8(8), "8"},
		{"uint16", uint16(16), "16"},
		{"uint32", uint32(32), "32"},
		{"uint64", uint64(64), "64"},
		{"float32", float32(1.25), "1.25"},
		{"float64", 1.5, "1.5"},
		{"string", "abc", "abc"},
		{"time", ts, ts.Format(time.RFC3339)},
		{"stringer", stringerValue("ok"), "stringer:ok"},
		{"error", errors.New("boom"), "boom"},
		{"named true", namedBool(true), "true"},
		{"named false", namedBool(false), "false"},
		{"named int", namedInt(23), "23"},
		{"named uint", namedUint(24), "24"},
		{"named float32", namedFloat32(2.5), "2.5"},
		{"named float64", namedFloat64(2.5), "2.5"},
		{"named string", namedString("named"), "named"},
	}

	for _, tt := range tests {
		t.Run(tt.name, expectValue[string](tt.src, tt.want))
	}
}

func TestSetPointerValues(t *testing.T) {
	var pi *int
	if err := setExisting(&pi, "11"); err != nil || pi == nil || *pi != 11 {
		t.Fatalf("nil *int set: got (%v, %#v), want (*11, nil)", err, pi)
	}

	old := pi
	if err := setExisting(&pi, "12"); err != nil || pi != old || *pi != 12 {
		t.Fatalf("existing *int set: got (%v, %#v), want same pointer to 12", err, pi)
	}

	if err := setExisting(&pi, "bad"); err == nil || pi != old || *pi != 12 {
		t.Fatalf("failed existing *int set: got (%v, %#v), want same pointer to 12 and error", err, pi)
	}

	var ps *string
	if err := setExisting(&ps, unsupported(1)); err == nil || ps != nil {
		t.Fatalf("failed nil *string set: got (%v, %#v), want nil pointer and error", err, ps)
	}
}

func TestSetBinderValues(t *testing.T) {
	var v bindValue
	if err := setExisting(&v, 10); err != nil || v != "bind:10" {
		t.Fatalf("value binder: got (%v, %q), want bind:10", err, v)
	}

	if err := setExisting(&v, "bad"); !errors.Is(err, errBadBind) || v != "bind:10" {
		t.Fatalf("failed value binder: got (%v, %q), want previous value and errBadBind", err, v)
	}

	if err := setExisting(&v, nil); err != nil || v != "bind:<nil>" {
		t.Fatalf("nil value binder: got (%v, %q), want bind:<nil>", err, v)
	}

	var p *bindValue
	if err := setExisting(&p, "ok"); err != nil || p == nil || *p != "bind:ok" {
		t.Fatalf("nil pointer binder: got (%v, %#v), want allocated bind:ok", err, p)
	}

	old := p
	if err := setExisting(&p, "next"); err != nil || p != old || *p != "bind:next" {
		t.Fatalf("existing pointer binder: got (%v, %#v), want same pointer to bind:next", err, p)
	}

	if err := setExisting(&p, "bad"); !errors.Is(err, errBadBind) || p != old || *p != "bind:next" {
		t.Fatalf("failed pointer binder: got (%v, %#v), want same previous pointer and errBadBind", err, p)
	}

	if err := setExisting(&p, nil); err != nil || p != old || *p != "bind:<nil>" {
		t.Fatalf("nil pointer binder: got (%v, %#v), want same pointer to bind:<nil>", err, p)
	}
}

func TestSetTextUnmarshalerValues(t *testing.T) {
	ts := time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)
	text := ts.Format(time.RFC3339)

	got, err := setValue[time.Time](ts)
	if err != nil || !got.Equal(ts) {
		t.Fatalf("time from time: got (%v, %v), want (%v, nil)", got, err, ts)
	}

	got, err = setValue[time.Time](text)
	if err != nil || !got.Equal(ts) {
		t.Fatalf("time from string: got (%v, %v), want (%v, nil)", got, err, ts)
	}

	got, err = setValue[time.Time]([]byte(text))
	if err != nil || !got.Equal(ts) {
		t.Fatalf("time from []byte: got (%v, %v), want (%v, nil)", got, err, ts)
	}

	got, err = setValue[time.Time](namedString(text))
	if err != nil || !got.Equal(ts) {
		t.Fatalf("time from named string: got (%v, %v), want (%v, nil)", got, err, ts)
	}

	got, err = setValue[time.Time](namedBytes(text))
	if err != nil || !got.Equal(ts) {
		t.Fatalf("time from named bytes: got (%v, %v), want (%v, nil)", got, err, ts)
	}

	var ptr *time.Time
	if err := setExisting(&ptr, text); err != nil || ptr == nil || !ptr.Equal(ts) {
		t.Fatalf("*time from string: got (%v, %#v), want (%v, nil)", err, ptr, ts)
	}

	next := ts.Add(time.Hour)
	if err := setExisting(&ptr, next); err != nil || ptr == nil || !ptr.Equal(next) {
		t.Fatalf("*time from time: got (%v, %#v), want (%v, nil)", err, ptr, next)
	}
}

func TestSetAssignableFallback(t *testing.T) {
	src := unsupported(7)
	got, err := setValue[unsupported](src)
	if err != nil || got != src {
		t.Fatalf("got (%v, %v), want (%v, nil)", got, err, src)
	}

	ptr := &src
	var gotPtr *unsupported
	if err := setExisting(&gotPtr, ptr); err != nil || gotPtr != ptr {
		t.Fatalf("got (%v, %#v), want (%#v, nil)", err, gotPtr, ptr)
	}

	if err := setExisting(&gotPtr, (*unsupported)(nil)); err != nil || gotPtr != nil {
		t.Fatalf("got (%v, %#v), want nil pointer", err, gotPtr)
	}
}

func TestSetSliceValues(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"assignable", func(t *testing.T) {
			src := []int{1, 2}
			got, err := setValue[[]int](src)
			if err != nil || !reflect.DeepEqual(got, src) {
				t.Fatalf("got (%v, %v), want (%v, nil)", got, err, src)
			}
		}},
		{"convert elements", func(t *testing.T) {
			got, err := setValue[[]int]([]string{"1", "2"})
			if err != nil || !reflect.DeepEqual(got, []int{1, 2}) {
				t.Fatalf("got (%v, %v), want ([1 2], nil)", got, err)
			}
		}},
		{"array source", func(t *testing.T) {
			got, err := setValue[[]string]([2]int{1, 2})
			if err != nil || !reflect.DeepEqual(got, []string{"1", "2"}) {
				t.Fatalf("got (%v, %v), want ([1 2], nil)", got, err)
			}
		}},
		{"typed nil", func(t *testing.T) {
			var src []namedString
			got := []string{"keep"}
			if err := setExisting(&got, src); err != nil || got != nil {
				t.Fatalf("got (%v, %v), want (nil, nil)", got, err)
			}
		}},
		{"element error preserves destination", func(t *testing.T) {
			got := []int{9}
			err := setExisting(&got, []string{"1", "bad"})
			if err == nil || !reflect.DeepEqual(got, []int{9}) {
				t.Fatalf("got (%v, %v), want preserved [9] and error", got, err)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestSetMapValues(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"assignable", func(t *testing.T) {
			src := map[string]int{"a": 1}
			got, err := setValue[map[string]int](src)
			if err != nil || !reflect.DeepEqual(got, src) {
				t.Fatalf("got (%v, %v), want (%v, nil)", got, err, src)
			}
		}},
		{"convert keys and elements", func(t *testing.T) {
			got, err := setValue[map[int]string](map[string]int{"1": 10, "2": 20})
			want := map[int]string{1: "10", 2: "20"}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("got (%v, %v), want (%v, nil)", got, err, want)
			}
		}},
		{"typed nil", func(t *testing.T) {
			var src map[namedString]int
			got := map[string]int{"keep": 1}
			if err := setExisting(&got, src); err != nil || got != nil {
				t.Fatalf("got (%v, %v), want (nil, nil)", got, err)
			}
		}},
		{"key error preserves destination", func(t *testing.T) {
			got := map[int]string{9: "keep"}
			err := setExisting(&got, map[string]int{"bad": 1})
			if err == nil || !reflect.DeepEqual(got, map[int]string{9: "keep"}) {
				t.Fatalf("got (%v, %v), want preserved map and error", got, err)
			}
		}},
		{"element error preserves destination", func(t *testing.T) {
			got := map[string]int{"keep": 9}
			err := setExisting(&got, map[string]string{"a": "bad"})
			if err == nil || !reflect.DeepEqual(got, map[string]int{"keep": 9}) {
				t.Fatalf("got (%v, %v), want preserved map and error", got, err)
			}
		}},
		{"duplicate converted key preserves destination", func(t *testing.T) {
			got := map[int]string{9: "keep"}
			err := setExisting(&got, map[string]string{"1": "a", "01": "b"})
			if err == nil || !reflect.DeepEqual(got, map[int]string{9: "keep"}) {
				t.Fatalf("got (%v, %v), want preserved map and error", got, err)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestSetScalarValueErrorsPreserveDestination(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{"bool parse", func() error {
			v := true
			err := setExisting(&v, "not-bool")
			assertEqual(t, v, true)
			return err
		}},
		{"int parse", func() error {
			v := 7
			err := setExisting(&v, "bad")
			assertEqual(t, v, 7)
			return err
		}},
		{"int overflow", func() error {
			v := int8(7)
			err := setExisting(&v, 128)
			assertEqual(t, v, int8(7))
			return err
		}},
		{"int fractional float", func() error {
			v := 7
			err := setExisting(&v, 12.5)
			assertEqual(t, v, 7)
			return err
		}},
		{"int infinite float", func() error {
			v := 7
			err := setExisting(&v, math.Inf(1))
			assertEqual(t, v, 7)
			return err
		}},
		{"uint negative int", func() error {
			v := uint(7)
			err := setExisting(&v, -1)
			assertEqual(t, v, uint(7))
			return err
		}},
		{"uint overflow", func() error {
			v := uint8(7)
			err := setExisting(&v, 256)
			assertEqual(t, v, uint8(7))
			return err
		}},
		{"uint NaN", func() error {
			v := uint(7)
			err := setExisting(&v, math.NaN())
			assertEqual(t, v, uint(7))
			return err
		}},
		{"float parse", func() error {
			v := 7.0
			err := setExisting(&v, "bad")
			assertEqual(t, v, 7.0)
			return err
		}},
		{"float32 overflow", func() error {
			v := float32(7)
			err := setExisting(&v, math.MaxFloat64)
			assertEqual(t, v, float32(7))
			return err
		}},
		{"unsupported string value", func() error {
			v := "keep"
			err := setExisting(&v, unsupported(1))
			assertEqual(t, v, "keep")
			return err
		}},
		{"nil value", func() error {
			v := 7
			err := setExisting(&v, nil)
			assertEqual(t, v, 7)
			return err
		}},
		{"unsupported type", func() error {
			v := unsupported(1)
			err := setExisting(&v, 2)
			assertEqual(t, v, unsupported(1))
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestSetNilSource(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{"setString", func() error {
			v := "keep"
			err := setExisting(&v, nil)
			assertEqual(t, v, "keep")
			return err
		}},
		{"setBool", func() error {
			v := true
			err := setExisting(&v, nil)
			assertEqual(t, v, true)
			return err
		}},
		{"setInt", func() error {
			v := 7
			err := setExisting(&v, nil)
			assertEqual(t, v, 7)
			return err
		}},
		{"setUint", func() error {
			v := uint(7)
			err := setExisting(&v, nil)
			assertEqual(t, v, uint(7))
			return err
		}},
		{"setFloat", func() error {
			v := 7.5
			err := setExisting(&v, nil)
			assertEqual(t, v, 7.5)
			return err
		}},
		{"setSlice", func() error {
			v := []int{7}
			err := setExisting(&v, nil)
			if !reflect.DeepEqual(v, []int{7}) {
				t.Fatalf("destination changed: got %v, want [7]", v)
			}
			return err
		}},
		{"setMap", func() error {
			v := map[string]int{"keep": 7}
			err := setExisting(&v, nil)
			if !reflect.DeepEqual(v, map[string]int{"keep": 7}) {
				t.Fatalf("destination changed: got %v, want map[keep:7]", v)
			}
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestNumericBoundaries(t *testing.T) {
	expectOK := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"int64 max float excluded", func(t *testing.T) {
			f := math.Nextafter(math.Ldexp(1, 63), 0)
			want := int64(f)
			v := int64(1)
			if err := setExisting(&v, f); err != nil || v != want {
				t.Fatalf("got (%v, %d), want %d", err, v, want)
			}
		}},
		{"uint64 max float excluded", func(t *testing.T) {
			f := math.Nextafter(math.Ldexp(1, 64), 0)
			want := uint64(f)
			v := uint64(1)
			if err := setExisting(&v, f); err != nil || v != want {
				t.Fatalf("got (%v, %d), want %d", err, v, want)
			}
		}},
		{"float32 infinities and NaN allowed", func(t *testing.T) {
			for _, f := range []float64{math.Inf(-1), math.Inf(1), math.NaN()} {
				var v float32
				if err := setExisting(&v, f); err != nil {
					t.Fatalf("set float32 from %v: %v", f, err)
				}
			}
		}},
	}

	for _, tt := range expectOK {
		t.Run(tt.name, tt.run)
	}

	for _, tt := range []struct {
		name string
		run  func() error
	}{
		{"int64 max float overflows", func() error {
			var v int64
			return setExisting(&v, math.Ldexp(1, 63))
		}},
		{"uint64 max float overflows", func() error {
			var v uint64
			return setExisting(&v, math.Ldexp(1, 64))
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func expectValue[T comparable](src any, want T) func(*testing.T) {
	return func(t *testing.T) {
		got, err := setValue[T](src)
		if err != nil || got != want {
			t.Fatalf("set %s from %s: got (%v, %v), want (%v, nil)",
				reflect.TypeFor[T](), reflect.TypeOf(src), got, err, want)
		}
	}
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("destination changed: got %v, want %v", got, want)
	}
}

func typeName(v any) string {
	return reflect.TypeOf(v).String()
}

func TestStringFormatAgreesWithStrconv(t *testing.T) {
	for _, f := range []float64{0.1, math.Pi, math.SmallestNonzeroFloat64} {
		t.Run(strconv.FormatFloat(f, 'g', -1, 64), expectValue[string](f, strconv.FormatFloat(f, 'f', -1, 64)))
	}
}
