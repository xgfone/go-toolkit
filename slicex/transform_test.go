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

package slicex_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/xgfone/go-toolkit/slicex"
)

func TestConversions(t *testing.T) {
	type ints []int
	for _, input := range []ints{nil, {}, {1, 2, 3}} {
		calls := 0
		converted := slicex.To(input, func(v int) string {
			calls++
			return fmt.Sprint(v)
		})
		if calls != len(input) || (converted == nil) != (input == nil) {
			t.Fatalf("To: %#v, calls=%d", converted, calls)
		}
		if len(converted) != len(input) || cap(converted) != len(input) {
			t.Fatal("To must allocate final length")
		}

		indexed := slicex.To2(input, func(i, v int) int { return i + v })
		for i, v := range input {
			if converted[i] != fmt.Sprint(v) || indexed[i] != i+v {
				t.Fatal("incorrect conversion")
			}
		}

		calls = 0
		selected := slicex.FilterTo(input, func(v int) (string, bool) {
			calls++
			return fmt.Sprint(v), v%2 == 1
		})
		var want []string
		for _, v := range input {
			if v%2 == 1 {
				want = append(want, fmt.Sprint(v))
			}
		}
		if !slices.Equal(selected, want) || calls != len(input) ||
			(selected == nil) != (input == nil) || cap(selected) < len(input) {
			t.Fatalf("FilterTo: %#v, calls=%d", selected, calls)
		}
	}
}

func TestMapCollisionAndIndex(t *testing.T) {
	calls := 0
	input := []string{"a", "b", "c"}
	got := slicex.Map(input, func(v string) (int, string) {
		calls++
		return 0, v
	})
	if calls != 3 || len(got) != 1 || got[0] != "c" {
		t.Fatalf("Map = %v, calls=%d", got, calls)
	}

	indexed := slicex.Map2(input, func(i int, v string) (int, string) {
		return i, v
	})
	for i, v := range input {
		if indexed[i] != v {
			t.Fatal(indexed)
		}
	}

	empty := slicex.Map([]int(nil), func(v int) (int, int) {
		t.Fatal("callback on nil input")
		return v, v
	})
	if empty == nil || len(empty) != 0 {
		t.Fatal(empty)
	}
}
