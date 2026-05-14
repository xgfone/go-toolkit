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

package validation

import (
	"errors"
	"reflect"
	"testing"
)

type mockValidator struct {
	err error
}

func (m mockValidator) Validate() error {
	return m.err
}

func TestValidate_DefaultNoop(t *testing.T) {
	// Test that Validate works with the default no-op function
	if err := Validate(nil); err != nil {
		t.Errorf("expect nil, but got an error: %v", err)
	}

	if err := Validate("test"); err != nil {
		t.Errorf("expect nil, but got an error: %v", err)
	}

	if err := Validate(mockValidator{err: nil}); err != nil {
		t.Errorf("expect nil, but got an error: %v", err)
	}

	if Validate(mockValidator{err: errors.New("test")}) == nil {
		t.Errorf("expect an error, but got nil")
	}
}

func TestSetValidateFunc_PanicWhenNil(t *testing.T) {
	// Save the current validation function
	originalValidate := _validate
	defer func() { _validate = originalValidate }()

	// Test that SetValidateFunc panics when given nil function
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		SetValidateFunc(nil)
	}()

	if !panicked {
		t.Error("expected panic when calling SetValidateFunc with nil function")
	}
}

func TestSetValidateFunc_ReturnsOnValue(t *testing.T) {
	// Save the current validation function
	originalValidate := _validate
	defer func() { _validate = originalValidate }()

	tests := []struct {
		name       string
		fn         func(value any) error
		value      any
		wantErr    bool
		wantCalled bool
	}{
		{
			name: "passes_value",
			fn: func(value any) error {
				if value != "test value" {
					t.Errorf("expected 'test value', got %v", value)
				}
				return nil
			},
			value:      "test value",
			wantErr:    false,
			wantCalled: true,
		},
		{
			name: "returns_error",
			fn: func(value any) error {
				return errors.New("validation failed")
			},
			value:   "test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			wrappedFn := func(value any) error {
				called = true
				return tt.fn(value)
			}

			SetValidateFunc(wrappedFn)
			err := Validate(tt.value)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if err.Error() != "validation failed" {
					t.Errorf("expected 'validation failed', got %v", err)
				}
			} else if err != nil {
				t.Errorf("expected nil, got %v", err)
			}

			if tt.wantCalled && !called {
				t.Error("validation function was not called")
			}
		})
	}
}

func TestValidate_WithDifferentValueTypes(t *testing.T) {
	// Save the current validation function
	originalValidate := _validate
	defer func() { _validate = originalValidate }()

	testCases := []struct {
		name  string
		value any
	}{
		{"string", "test string"},
		{"int", 123},
		{"float", 3.14},
		{"bool", true},
		{"slice", []string{"a", "b", "c"}},
		{"map", map[string]int{"a": 1}},
		{"nil", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			SetValidateFunc(func(value any) error {
				called = true
				if !reflect.DeepEqual(value, tc.value) {
					t.Errorf("test case %s: expected value %v, got %v", tc.name, tc.value, value)
				}
				return nil
			})

			if err := Validate(tc.value); err != nil {
				t.Errorf("test case %s: expected nil error, got %v", tc.name, err)
			}

			if !called {
				t.Errorf("test case %s: validation function was not called", tc.name)
			}
		})
	}
}

func TestSetValidateFunc_OverridesPreviousFunction(t *testing.T) {
	// Save the current validation function
	originalValidate := _validate
	defer func() { _validate = originalValidate }()

	// Set first function
	firstCalled := false
	SetValidateFunc(func(value any) error {
		firstCalled = true
		return nil
	})

	_ = Validate("test")

	if !firstCalled {
		t.Error("first validation function should have been called")
	}

	// Override with second function
	secondCalled := false
	SetValidateFunc(func(value any) error {
		secondCalled = true
		return errors.New("error from second function")
	})

	// Reset firstCalled to check if it's called again
	firstCalled = false
	err := Validate("test")

	if firstCalled {
		t.Error("first validation function should not be called after being overridden")
	}

	if !secondCalled {
		t.Error("second validation function should have been called")
	}

	if err == nil || err.Error() != "error from second function" {
		t.Errorf("expected 'error from second function', got %v", err)
	}
}
