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
	"strings"
	"testing"
)

type testValidator struct {
	Name  string `json:"name"`
	Valid bool
}

func (t *testValidator) Validate(ctx context.Context) error {
	if !t.Valid {
		return errors.New("invalid")
	}
	return nil
}

func TestBindJSONBytes(t *testing.T) {
	ctx := context.Background()

	// Test scenario 1: Unmarshal success, validation success
	validJSON := []byte(`{"name":"test"}`)
	var validObj testValidator
	validObj.Valid = true
	err := BindJSONBytes(ctx, validJSON, &validObj)
	if err != nil {
		t.Errorf("BindJSONBytes: expected no error, got %v", err)
	}

	// Test scenario 2: Unmarshal success, validation failure
	validJSON2 := []byte(`{"name":"test"}`)
	var invalidObj testValidator
	invalidObj.Valid = false
	err = BindJSONBytes(ctx, validJSON2, &invalidObj)
	if err == nil {
		t.Error("BindJSONBytes: expected validation error, got nil")
	}

	// Test scenario 3: Unmarshal failure
	invalidJSON := []byte(`{invalid json}`)
	var obj testValidator
	err = BindJSONBytes(ctx, invalidJSON, &obj)
	if err == nil {
		t.Error("BindJSONBytes: expected unmarshal error, got nil")
	}
}

func TestBindJSONString(t *testing.T) {
	ctx := context.Background()

	// Test scenario 1: Unmarshal success, validation success
	validJSON := `{"name":"test"}`
	var validObj testValidator
	validObj.Valid = true
	err := BindJSONString(ctx, validJSON, &validObj)
	if err != nil {
		t.Errorf("BindJSONString: expected no error, got %v", err)
	}

	// Test scenario 2: Unmarshal success, validation failure
	validJSON2 := `{"name":"test"}`
	var invalidObj testValidator
	invalidObj.Valid = false
	err = BindJSONString(ctx, validJSON2, &invalidObj)
	if err == nil {
		t.Error("BindJSONString: expected validation error, got nil")
	}

	// Test scenario 3: Unmarshal failure
	invalidJSON := `{invalid json}`
	var obj testValidator
	err = BindJSONString(ctx, invalidJSON, &obj)
	if err == nil {
		t.Error("BindJSONString: expected unmarshal error, got nil")
	}
}

func TestBindJSONReader(t *testing.T) {
	ctx := context.Background()

	// Test scenario 1: Unmarshal success, validation success
	validJSON := strings.NewReader(`{"name":"test"}`)
	var validObj testValidator
	validObj.Valid = true
	err := BindJSONReader(ctx, validJSON, &validObj)
	if err != nil {
		t.Errorf("BindJSONReader: expected no error, got %v", err)
	}

	// Test scenario 2: Unmarshal success, validation failure
	validJSON2 := strings.NewReader(`{"name":"test"}`)
	var invalidObj testValidator
	invalidObj.Valid = false
	err = BindJSONReader(ctx, validJSON2, &invalidObj)
	if err == nil {
		t.Error("BindJSONReader: expected validation error, got nil")
	}

	// Test scenario 3: Unmarshal failure
	invalidJSON := strings.NewReader(`{invalid json}`)
	var obj testValidator
	err = BindJSONReader(ctx, invalidJSON, &obj)
	if err == nil {
		t.Error("BindJSONReader: expected unmarshal error, got nil")
	}
}
