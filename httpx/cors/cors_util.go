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

package cors

import (
	"slices"
	"strings"
)

func containsWildcard(values []string) bool {
	return slices.ContainsFunc(values, func(s string) bool {
		return strings.TrimSpace(s) == "*"
	})
}

func removeWildcard(values []string) []string {
	n := 0
	values = slices.Clone(values)
	for _, value := range values {
		if strings.TrimSpace(value) != "*" {
			values[n] = value
			n++
		}
	}
	return values[:n]
}

func validToken(value string) bool {
	if value == "" {
		return false
	}

	for i := range len(value) {
		if !isTChar(value[i]) {
			return false
		}
	}

	return true
}

func isTChar(c byte) bool {
	return tcharTable[c]
}

var tcharTable = func() [256]bool {
	var table [256]bool
	for c := byte('0'); c <= byte('9'); c++ {
		table[c] = true
	}
	for c := byte('A'); c <= byte('Z'); c++ {
		table[c] = true
	}
	for c := byte('a'); c <= byte('z'); c++ {
		table[c] = true
	}
	for _, c := range []byte("!#$%&'*+-.^_`|~") {
		table[c] = true
	}
	return table
}()
