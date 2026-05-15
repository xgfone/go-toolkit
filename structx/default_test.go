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
	"reflect"
	"strings"
	"testing"
)

func withSetDefaultBackend(t *testing.T, fn func(reflect.Type, reflect.Value) error) {
	t.Helper()
	old := _setdefault
	_setdefault = fn
	t.Cleanup(func() { _setdefault = old })
}

func TestSetDefault_Frontend(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		var v *struct{ A int }
		err := SetDefault(v)
		if !errors.Is(err, errDefaultNilPointer) {
			t.Fatalf("expect errDefaultNilPointer, got %v", err)
		}
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		v := 1
		err := SetDefault(&v)
		if !errors.Is(err, errDefaultNotStruct) {
			t.Fatalf("expect errDefaultNotStruct, got %v", err)
		}
	})

	t.Run("valid struct pointer delegates to backend", func(t *testing.T) {
		var called int
		withSetDefaultBackend(t, func(rt reflect.Type, rv reflect.Value) error {
			called++
			if rt.Kind() != reflect.Struct {
				t.Fatalf("expect struct type, got %s", rt.Kind())
			}
			if rv.Kind() != reflect.Struct {
				t.Fatalf("expect struct value, got %s", rv.Kind())
			}
			if !rv.CanAddr() {
				t.Fatal("expect addressable struct value")
			}
			return nil
		})

		err := SetDefault(&struct{ A int }{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called != 1 {
			t.Fatalf("expect backend called once, got %d", called)
		}
	})
}

func TestSetDefaultAny_Frontend(t *testing.T) {
	tests := []struct {
		name string
		in   any
		err  error
	}{
		{name: "nil interface", in: nil, err: errDefaultNilPointer},
		{name: "non-pointer", in: 1, err: errDefaultNotStruct},
		{name: "pointer to non-struct", in: new(int), err: errDefaultNotStruct},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := 0
			withSetDefaultBackend(t, func(reflect.Type, reflect.Value) error {
				called++
				return nil
			})

			err := SetDefaultAny(tc.in)
			if !errors.Is(err, tc.err) {
				t.Fatalf("expect %v, got %v", tc.err, err)
			}
			if called != 0 {
				t.Fatalf("backend should not be called, got %d", called)
			}
		})
	}

	t.Run("typed nil struct pointer", func(t *testing.T) {
		var v *struct{ A int }
		err := SetDefaultAny(v)
		if !errors.Is(err, errDefaultNilPointer) {
			t.Fatalf("expect errDefaultNilPointer, got %v", err)
		}
	})

	t.Run("valid struct pointer delegates to backend", func(t *testing.T) {
		var called int
		withSetDefaultBackend(t, func(rt reflect.Type, rv reflect.Value) error {
			called++
			if rt.Kind() != reflect.Struct {
				t.Fatalf("expect struct type, got %s", rt.Kind())
			}
			if rv.Kind() != reflect.Struct {
				t.Fatalf("expect struct value, got %s", rv.Kind())
			}
			return nil
		})

		err := SetDefaultAny(&struct{ A int }{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called != 1 {
			t.Fatalf("expect backend called once, got %d", called)
		}
	})
}

type backendInner struct {
	Value int `default:"7"`
}

type backendSample struct {
	Name    string `default:"kit"`
	Age     int    `default:"18"`
	Keep    string `default:"ignored"`
	Enabled bool   `default:"true"`
	Score   *int   `default:"9"`
	Inner   backendInner
	InnerP  *backendInner
	NoTag   int
}

func TestSetDefault_Backend_AppliesDefaults(t *testing.T) {
	v := backendSample{Keep: "custom", Inner: backendInner{Value: 88}}
	err := setDefault(reflect.TypeFor[backendSample](), reflect.ValueOf(&v).Elem())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v.Name != "kit" || v.Age != 18 || !v.Enabled {
		t.Fatalf("scalar defaults not applied correctly: %+v", v)
	}
	if v.Keep != "custom" {
		t.Fatalf("non-zero field should not be overwritten, got %q", v.Keep)
	}
	if v.Score == nil || *v.Score != 9 {
		t.Fatalf("pointer default not applied: %+v", v.Score)
	}
	if v.Inner.Value != 88 {
		t.Fatalf("nested non-zero field should not be overwritten, got %d", v.Inner.Value)
	}
	if v.InnerP == nil || v.InnerP.Value != 7 {
		t.Fatalf("nested pointer struct default not applied: %+v", v.InnerP)
	}
	if v.NoTag != 0 {
		t.Fatalf("untagged field should stay zero, got %d", v.NoTag)
	}
}

func TestSetDefault_Backend_FieldError(t *testing.T) {
	type bad struct {
		Age int8 `default:"999"`
	}

	v := bad{}
	err := setDefault(reflect.TypeFor[bad](), reflect.ValueOf(&v).Elem())
	if err == nil {
		t.Fatal("expect error")
	}
	if !strings.Contains(err.Error(), `"Age":`) {
		t.Fatalf("expect field name wrapped in error, got %v", err)
	}
}
