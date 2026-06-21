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

import "strings"

// IsInteger reports whether s represents a valid integer.
//
// Rules:
//   - An optional leading '+' or '-' sign is allowed.
//   - The numeric part must not be empty.
//   - Leading zeros are forbidden unless the number is exactly "0".
//   - No decimal points, exponents, or any non‑digit characters.
//   - No leading or trailing whitespace.
//
// Valid examples: "0", "123", "+456", "-789"
// Invalid examples: "", "01", "0.1", " 123", "12a", "+"
func IsInteger(s string) bool {
	if s == "" {
		return false
	}

	if s[0] == '+' || s[0] == '-' {
		s = s[1:]
	}

	switch s {
	case "":
		return false

	case "0":
		return true
	}

	if s[0] == '0' {
		return false
	}

	return isASCIIDigits(s)
}

// IsFloat reports whether s represents a valid decimal floating‑point number
// (no exponent part, sign allowed).
//
// Rules:
//   - An optional leading '+' or '-' sign is allowed.
//   - At least one digit must appear either before or after the decimal point.
//   - The integer part follows the same rules as IsInteger (no leading zero unless it is exactly "0").
//   - The fractional part, if present, must consist only of digits; it may be empty (e.g. "123.").
//   - No leading/trailing whitespace, no exponent (e.g. "1e5").
//
// Valid examples: "0", "123", ".456", "123.", "-0.5", "+12.34"
// Invalid examples: "", ".", "01.2", "12..3", " 1.2", "1e3"
func IsFloat(s string) bool {
	if s == "" {
		return false
	}

	if s[0] == '+' || s[0] == '-' {
		s = s[1:]
	}

	if s == "." {
		return false
	}

	decimal, fractional, ok := strings.Cut(s, ".")
	if !ok {
		return IsInteger(decimal)
	}

	if decimal != "" && decimal != "0" && !IsInteger(decimal) {
		return false
	}

	return isASCIIDigits(fractional)
}

// IsASCIIDigits reports whether s is non-empty and contains only ASCII digits.
func IsASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	return isASCIIDigits(s)
}

func isASCIIDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
