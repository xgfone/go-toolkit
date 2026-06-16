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
	"slices"
	"testing"
)

func TestValidateAnyFrontend(t *testing.T) {
	validate := func(reflect.Value, string) error {
		t.Fatal("validator should not be called")
		return nil
	}

	var nilStruct *struct{ Name string }
	tests := []struct {
		name string
		in   any
		err  string
	}{
		{name: "nil", in: nil},
		{name: "nil struct pointer", in: nilStruct},
		{name: "struct value", in: struct{ Name string }{}},
		{name: "non struct", in: 1, err: "Validate: not a struct or pointer to struct"},
		{name: "pointer to non struct", in: new(int), err: "Validate: not a pointer to struct"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAny(tt.in, validate)
			if tt.err == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || err.Error() != tt.err {
				t.Fatalf("got error %v, want %q", err, tt.err)
			}
		})
	}
}

func TestValidateAnyPanicsWithoutValidator(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = ValidateAny(struct{}{}, nil)
}

func TestValidateAnyFields(t *testing.T) {
	type address struct {
		City string `json:"city" validate:"city-rule"`
	}
	type profile struct {
		Age int
	}
	type user struct {
		Name     string `json:"name,omitempty" validate:"name-rule"`
		Address  address
		Optional *address
		Profile  *profile `json:"profile" validate:"profile-rule"`
		Ignored  string
	}

	u := user{Name: "alice", Address: address{City: "hangzhou"}}
	var rules []string
	err := ValidateAny(&u, func(value reflect.Value, rule string) error {
		rules = append(rules, rule)
		if rule == "profile-rule" && (!value.IsNil() || value.Type() != reflect.TypeFor[*profile]()) {
			t.Fatalf("profile value = %#v, want nil *profile", value)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"name-rule", "city-rule", "profile-rule"}
	if !slices.Equal(rules, want) {
		t.Fatalf("rules = %v, want %v", rules, want)
	}
	if !reflect.ValueOf(u.Optional).IsNil() {
		t.Fatal("nil nested pointer should not be allocated")
	}
}

func TestValidateAnyPassesReflectValue(t *testing.T) {
	type namedInt int
	type values struct {
		Name  string   `validate:"name"`
		Count namedInt `validate:"count"`
	}

	got := make(map[string]reflect.Type)
	err := ValidateAny(values{Name: "x", Count: 12}, func(value reflect.Value, rule string) error {
		got[rule] = value.Type()
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := map[string]reflect.Type{
		"name":  reflect.TypeFor[string](),
		"count": reflect.TypeFor[namedInt](),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestValidateAnyFieldError(t *testing.T) {
	errBad := errors.New("bad")
	err := ValidateAny(struct {
		Name string `json:"name,omitempty" validate:"name-rule"`
	}{}, func(reflect.Value, string) error {
		return errBad
	})

	if !errors.Is(err, errBad) {
		t.Fatalf("got error %v, want wrapping %v", err, errBad)
	}
	if err.Error() != `"name": bad` {
		t.Fatalf("got error %q", err)
	}
}
