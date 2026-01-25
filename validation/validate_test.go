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
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestValidate_DefaultNoop(t *testing.T) {
	// Test that Validate works with the default no-op function
	if err := Validate(context.Background(), nil); err != nil {
		t.Errorf("expect nil, but got an error: %v", err)
	}

	if err := Validate(context.Background(), "test"); err != nil {
		t.Errorf("expect nil, but got an error: %v", err)
	}

	if err := Validate(context.Background(), 123); err != nil {
		t.Errorf("expect nil, but got an error: %v", err)
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

func TestSetValidateFunc_SetsFunction(t *testing.T) {
	// Save the current validation function
	originalValidate := _validate
	defer func() { _validate = originalValidate }()

	// Test setting a validation function
	called := false
	expectedValue := "test value"

	SetValidateFunc(func(ctx context.Context, value any) error {
		called = true
		if value != expectedValue {
			t.Errorf("expected value %v, got %v", expectedValue, value)
		}
		return nil
	})

	ctx := context.Background()
	err := Validate(ctx, expectedValue)

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	if !called {
		t.Error("validation function was not called")
	}
}

func TestValidate_ReturnsErrorFromFunction(t *testing.T) {
	// Save the current validation function
	originalValidate := _validate
	defer func() { _validate = originalValidate }()

	// Test that Validate returns the error from the validation function
	expectedErr := errors.New("validation failed")
	SetValidateFunc(func(ctx context.Context, value any) error {
		return expectedErr
	})

	ctx := context.Background()
	err := Validate(ctx, "test")

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
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
			SetValidateFunc(func(ctx context.Context, value any) error {
				called = true
				if !reflect.DeepEqual(value, tc.value) {
					t.Errorf("test case %s: expected value %v, got %v", tc.name, tc.value, value)
				}
				return nil
			})

			ctx := context.Background()
			if err := Validate(ctx, tc.value); err != nil {
				t.Errorf("test case %s: expected nil error, got %v", tc.name, err)
			}

			if !called {
				t.Errorf("test case %s: validation function was not called", tc.name)
			}
		})
	}
}

func TestValidate_ContextPropagation(t *testing.T) {
	// Save the current validation function
	originalValidate := _validate
	defer func() { _validate = originalValidate }()

	ctx := context.Background()
	ctxWithValue := context.WithValue(ctx, "test-key", "test-value")

	SetValidateFunc(func(ctx context.Context, value any) error {
		if ctx.Value("test-key") != "test-value" {
			t.Error("context value not propagated to validation function")
		}
		return nil
	})

	if err := Validate(ctxWithValue, "test"); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestSetValidateFunc_OverridesPreviousFunction(t *testing.T) {
	// Save the current validation function
	originalValidate := _validate
	defer func() { _validate = originalValidate }()

	// Set first function
	firstCalled := false
	SetValidateFunc(func(ctx context.Context, value any) error {
		firstCalled = true
		return nil
	})

	ctx := context.Background()
	Validate(ctx, "test")

	if !firstCalled {
		t.Error("first validation function should have been called")
	}

	// Override with second function
	secondCalled := false
	SetValidateFunc(func(ctx context.Context, value any) error {
		secondCalled = true
		return errors.New("error from second function")
	})

	// Reset firstCalled to check if it's called again
	firstCalled = false
	err := Validate(ctx, "test")

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
