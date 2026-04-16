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

package stringx

import "testing"

func TestIsInteger(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		// valid integers
		{"zero", "0", true},
		{"positive", "123", true},
		{"negative", "-456", true},
		{"explicit plus", "+789", true},
		{"large number", "999999999999", true},

		// invalid integers
		{"empty string", "", false},
		{"only plus", "+", false},
		{"only minus", "-", false},
		{"leading zero", "01", false},
		{"negative leading zero", "-01", false},
		{"contains dot", "12.3", false},
		{"contains letter", "12a", false},
		{"leading space", " 123", false},
		{"trailing space", "123 ", false},
		{"sign in middle", "12-3", false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInteger(tt.s); got != tt.want {
				t.Errorf("%d: IsInteger(%q) = %v, want %v", i, tt.s, got, tt.want)
			}
		})
	}
}

func TestIsFloat(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		// valid floats
		{"integer form", "123", true},
		{"negative integer", "-456", true},
		{"explicit plus integer", "+789", true},
		{"empty fractional part", "123.", true},
		{"empty integer part", ".456", true},
		{"negative zero", "-0.0", true},
		{"normal decimal", "12.34", true},
		{"negative decimal", "-0.5", true},
		{"plus decimal", "+12.34", true},
		{"zero variants", "0.0", true},
		{"only integer zero", "0", true},

		// invalid floats
		{"empty string", "", false},
		{"bare dot", ".", false},
		{"sign only", "+", false},
		{"sign plus dot", "+.", false},
		{"multiple dots", "1.2.3", false},
		{"integer part leading zero", "01.2", false},
		{"fractional part non-digit", "12.3a", false},
		{"exponent notation", "1e5", false},
		{"leading space", " 1.2", false},
		{"space after sign", "- 1.2", false},
		{"fractional slash", "12/3", false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFloat(tt.s); got != tt.want {
				t.Errorf("%d: IsFloat(%q) = %v, want %v", i, tt.s, got, tt.want)
			}
		})
	}
}
