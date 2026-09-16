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

package mapx_test

import (
	"fmt"
	"testing"

	"github.com/xgfone/go-toolkit/mapx"
)

func TestTo(t *testing.T) {
	type values map[string]int
	for _, input := range []values{nil, {}, {"a": 1, "b": 2}} {
		calls := 0
		got := mapx.To(input, func(k string, v int) (string, string) {
			calls++
			return k, fmt.Sprint(v)
		})
		if calls != len(input) || len(got) != len(input) || (got == nil) != (input == nil) {
			t.Fatalf("To = %v, calls=%d", got, calls)
		}

		for k, v := range input {
			if got[k] != fmt.Sprint(v) {
				t.Fatal(got)
			}
		}
	}

	last := 0
	input := map[int]int{1: 10, 2: 20, 3: 30}
	got := mapx.To(input, func(_ int, v int) (int, int) {
		last = v
		return 0, v
	})
	if len(got) != 1 || got[0] != last {
		t.Fatalf("collision = %v, last=%d", got, last)
	}
}
