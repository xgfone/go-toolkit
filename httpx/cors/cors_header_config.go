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
	"fmt"
	"slices"
	"strings"
)

func joinHeaderValues(name string, values []string, allowWildcard bool) string {
	if len(values) == 0 {
		return ""
	}

	n := 0
	values = slices.Clone(values)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if value != "*" || !allowWildcard {
			if !validToken(value) {
				panic(fmt.Errorf("cors.Config.%s: invalid value %q", name, value))
			}
		}

		if value != "" {
			values[n] = value
			n++
		}
	}

	return strings.Join(values[:n], ", ")
}

func splitHeaderValues(values []string) []string {
	var hs []string
	for _, value := range values {
		for h := range strings.SplitSeq(value, ",") {
			if h = strings.TrimSpace(h); h != "" {
				hs = append(hs, h)
			}
		}
	}
	return hs
}
