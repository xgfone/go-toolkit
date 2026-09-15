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
	"strings"
	"testing"
)

func TestSetDefault_InvalidPointer(t *testing.T) {
	var v *struct{ A int }
	if err := SetDefault(v); !errors.Is(err, errDefaultNilPointer) {
		t.Fatalf("expected nil pointer error, got %v", err)
	}
	n := 1
	if err := SetDefault(&n); !errors.Is(err, errDefaultNotStruct) {
		t.Fatalf("expected non-struct error, got %v", err)
	}
}

func TestSetDefaultAny_InvalidPointer(t *testing.T) {
	tests := []struct {
		name string
		in   any
		err  error
	}{
		{"nil interface", nil, errDefaultNilPointer},
		{"non-pointer", 1, errDefaultNotStruct},
		{"pointer to non-struct", new(int), errDefaultNotStruct},
		{"typed nil struct pointer", (*struct{ A int })(nil), errDefaultNilPointer},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := SetDefaultAny(tc.in); !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
		})
	}
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

func TestSetDefault_AppliesDefaults(t *testing.T) {
	for name, setDefault := range map[string]func(*backendSample) error{
		"generic": SetDefault[backendSample],
		"any":     func(v *backendSample) error { return SetDefaultAny(v) },
	} {
		t.Run(name, func(t *testing.T) {
			v := backendSample{Keep: "custom", Inner: backendInner{Value: 88}}
			err := setDefault(&v)
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
		})
	}
}

func TestSetDefault_FieldError(t *testing.T) {
	type bad struct {
		Age int8 `default:"999"`
	}

	for name, setDefault := range map[string]func(*bad) error{
		"generic": SetDefault[bad],
		"any":     func(v *bad) error { return SetDefaultAny(v) },
	} {
		t.Run(name, func(t *testing.T) {
			v := bad{}
			err := setDefault(&v)
			if err == nil {
				t.Fatal("expect error")
			}
			if !strings.Contains(err.Error(), "Age:") {
				t.Fatalf("expect field name wrapped in error, got %v", err)
			}

		})
	}
}
