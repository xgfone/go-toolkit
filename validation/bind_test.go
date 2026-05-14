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
	"strings"
	"testing"
)

type testValidator struct {
	Name  string `json:"name"`
	Valid bool
}

func (t *testValidator) Validate() error {
	if !t.Valid {
		return errors.New("invalid")
	}
	return nil
}

func TestBind(t *testing.T) {
	validJSON := `{"name":"test"}`
	invalidJSON := `{invalid json}`

	tests := []struct {
		name         string
		validInput   func(any) error
		invalidInput func(any) error
	}{
		{"Bytes", func(out any) error { return BindJSONBytes([]byte(validJSON), out) },
			func(out any) error { return BindJSONBytes([]byte(invalidJSON), out) }},
		{"String", func(out any) error { return BindJSONString(validJSON, out) },
			func(out any) error { return BindJSONString(invalidJSON, out) }},
		{"Reader", func(out any) error { return BindJSONReader(strings.NewReader(validJSON), out) },
			func(out any) error { return BindJSONReader(strings.NewReader(invalidJSON), out) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Valid object + valid input → success
			if err := tt.validInput(&testValidator{Valid: true}); err != nil {
				t.Errorf("expected no error, got %v", err)
			}

			// Invalid object + valid input → validation error
			if err := tt.validInput(&testValidator{Valid: false}); err == nil {
				t.Error("expected validation error, got nil")
			}

			// Invalid input → unmarshal error
			if err := tt.invalidInput(&testValidator{}); err == nil {
				t.Error("expected unmarshal error, got nil")
			}
		})
	}
}
