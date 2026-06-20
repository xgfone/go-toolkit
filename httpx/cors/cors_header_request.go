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

import "strings"

func parseRequestHeaderValues(values []string) (headers []string, hasNonWildcardRequestHeader bool, ok bool) {
	if capacity := estimateRequestHeaderCount(values); capacity > 0 {
		headers = make([]string, 0, capacity)
	}

	for _, value := range values {
		for {
			part := value
			if i := strings.IndexByte(value, ','); i >= 0 {
				part = value[:i]
				value = value[i+1:]
			} else {
				value = ""
			}

			header := strings.TrimSpace(part)
			if header != "" {
				if !validToken(header) {
					return nil, false, false
				}
				if isCORSNonWildcardRequestHeaderName(header) {
					hasNonWildcardRequestHeader = true
				}
				headers = append(headers, header)
			}

			if value == "" {
				break
			}
		}
	}
	return headers, hasNonWildcardRequestHeader, true
}

func estimateRequestHeaderCount(values []string) int {
	const maxRequestHeadersPrealloc = 4

	count := 0
	for _, value := range values {
		if value == "" {
			continue
		}

		count++
		count += strings.Count(value, ",")

		if count >= maxRequestHeadersPrealloc {
			return maxRequestHeadersPrealloc
		}
	}
	return count
}

func isCORSNonWildcardRequestHeaderName(header string) bool {
	return strings.EqualFold(header, "Authorization")
}
