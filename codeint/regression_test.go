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

package codeint

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestRegressionErrorIsWithUncomparableData(t *testing.T) {
	for _, tc := range []struct {
		name string
		data any
	}{
		{"slice", []int{1}},
		{"map", map[string]int{"value": 1}},
		{"nil slice", []int(nil)},
		{"nil map", map[string]int(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := NewError(400).WithData(tc.data)
			sameCode := err.WithReason("different reason")
			otherCode := err.WithCode(401)

			for _, match := range []struct {
				name   string
				err    error
				target error
				want   bool
			}{
				{"self", err, err, true},
				{"preset", err, NewError(400), true},
				{"same code", err, sameCode, true},
				{"different code", err, otherCode, false},
				{"value and pointer", err, &sameCode, true},
				{"pointer and value", &err, sameCode, true},
				{"pointers", &err, &sameCode, true},
				{"wrapped", fmt.Errorf("wrapped: %w", err), sameCode, true},
				{"joined", errors.Join(errors.New("other"), err), sameCode, true},
				{"unrelated", err, errors.New("other"), false},
				{"nil target", err, nil, false},
				{"nil error", nil, err, false},
			} {
				t.Run(match.name, func(t *testing.T) {
					if got := errors.Is(match.err, match.target); got != match.want {
						t.Fatalf("errors.Is(%T, %T) = %v, want %v",
							match.err, match.target, got, match.want)
					}
				})
			}

			sentinel := errors.New("wrapped cause")
			wrapped := err.WithError(sentinel)
			if !errors.Is(wrapped, sentinel) {
				t.Fatal("errors.Is did not find the wrapped cause")
			}
		})
	}
}

func TestRegressionErrorIsPointerTarget(t *testing.T) {
	var nilTarget *Error
	value := NewError(400)
	matching := NewError(400)
	different := NewError(401)

	for _, tc := range []struct {
		name   string
		value  Error
		target *Error
		want   bool
	}{
		{"nil target", value, nilTarget, false},
		{"nil target and zero code", Error{}, nilTarget, false},
		{"same code", value, &matching, true},
		{"different code", value, &different, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.value.Is(tc.target); got != tc.want {
				t.Fatalf("Error.Is = %v, want %v", got, tc.want)
			}

			wrapped := fmt.Errorf("wrapped: %w", tc.value)
			for _, err := range []error{tc.value, &tc.value, wrapped} {
				if got := errors.Is(err, tc.target); got != tc.want {
					t.Errorf("errors.Is(%T, %T) = %v, want %v",
						err, tc.target, got, tc.want)
				}
			}
		})
	}
}

func TestRegressionErrorJSONWithUncomparableData(t *testing.T) {
	for _, tc := range []struct {
		name string
		data any
		want string
	}{
		{"slice", []int{1}, `{"Data":[1],"Code":400,"Message":"Bad Request"}`},
		{"map", map[string]int{"value": 1}, `{"Data":{"value":1},"Code":400,"Message":"Bad Request"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := NewError(400).WithData(tc.data)
			data, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if string(data) != tc.want {
				t.Fatalf("JSON = %s, want %s", data, tc.want)
			}

			var decoded Error
			if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
				t.Fatal(unmarshalErr)
			}
			if !errors.Is(decoded, err) {
				t.Fatal("decoded error did not match the original code")
			}
		})
	}
}

func TestRegressionLowBusinessCode(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewError(42).ServeHTTP panics: %v", r)
		}
	}()

	e := NewError(42)
	if s := e.StatusCode(); s != 500 {
		t.Errorf("expected safe HTTP status 500 for business code 42, got %d", s)
	}
	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
}
