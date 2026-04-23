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

package redact

import (
	"fmt"
	"testing"
)

type testRedactor struct {
	v string
}

func (r testRedactor) Redact(level Level) any {
	return fmt.Sprintf("value:%s:%s", level.String(), r.v)
}

func TestLevelValid(t *testing.T) {
	tests := []struct {
		level Level
		want  bool
	}{
		{LevelRaw, true},
		{LevelTrusted, true},
		{LevelExternal, true},
		{LevelPublic, true},
		{Level(255), false},
	}

	for _, tt := range tests {
		if got := tt.level.Valid(); got != tt.want {
			t.Fatalf("Valid(%v) = %v, want %v", tt.level, got, tt.want)
		}
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelRaw, "Raw"},
		{LevelTrusted, "Trusted"},
		{LevelExternal, "External"},
		{LevelPublic, "Public"},
		{Level(9), "redact.Level(9)"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Fatalf("String(%d) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestRedact(t *testing.T) {
	t.Run("redactor", func(t *testing.T) {
		got := Redact(testRedactor{v: "x"}, LevelTrusted)
		want := "value:Trusted:x"
		if got != want {
			t.Fatalf("Redact(redactor) = %#v, want %#v", got, want)
		}
	})

	t.Run("passthrough", func(t *testing.T) {
		got := Redact(123, LevelPublic)
		if got != 123 {
			t.Fatalf("Redact(passthrough) = %#v, want %#v", got, 123)
		}
	})
}
