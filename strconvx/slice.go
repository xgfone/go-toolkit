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

package strconvx

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseInt64Slice parses s as a comma-separated list of decimal int64 values.
// Leading and trailing Unicode whitespace is ignored for each element.
// Each element is parsed using strconv.ParseInt with base 10 and bit size 64.
//
// Empty or whitespace-only input returns nil, nil. An empty, invalid, or
// out-of-range element returns a nil slice and an error containing the element's
// one-based position and wrapping the underlying *strconv.NumError.
func ParseInt64Slice(s string) ([]int64, error) {
	return parseIntegerSlice(s, strconv.ParseInt)
}

// ParseUint64Slice parses s as a comma-separated list of decimal uint64 values.
// Leading and trailing Unicode whitespace is ignored for each element.
// Each element is parsed using strconv.ParseUint with base 10 and bit size 64;
// neither '+' nor '-' signs are permitted.
//
// Empty or whitespace-only input returns nil, nil. An empty, invalid, or
// out-of-range element returns a nil slice and an error containing the element's
// one-based position and wrapping the underlying *strconv.NumError.
func ParseUint64Slice(s string) ([]uint64, error) {
	return parseIntegerSlice(s, strconv.ParseUint)
}

func parseIntegerSlice[T int64 | uint64](s string, parse func(string, int, int) (T, error)) ([]T, error) {
	if s = strings.TrimSpace(s); s == "" {
		return nil, nil
	}

	values := make([]T, 0, strings.Count(s, ",")+1)
	for part := range strings.SplitSeq(s, ",") {
		value, err := parse(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("strconvx: element %d: %w", len(values)+1, err)
		}
		values = append(values, value)
	}
	return values, nil
}
