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

package slicex

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"testing"
)

func ExampleGroupBy() {
	groups := GroupBy([]string{"one", "two", "four", "six"}, func(s string) int {
		return len(s)
	})

	fmt.Println(groups[3])
	fmt.Println(groups[4])
	// Output:
	// [one two six]
	// [four]
}

func ExampleHasDuplicates() {
	fmt.Println(HasDuplicates([]int{3, 1, 3}))
	fmt.Println(HasDuplicates([]int{3, 1, 2}))
	// Output:
	// true
	// false
}

func ExampleUnique() {
	fmt.Println(Unique([]string{"b", "a", "b", "c", "a"}))
	// Output: [b a c]
}

func TestGroupBy(t *testing.T) {
	type item struct {
		Team string
		ID   int
	}
	type items []item

	input := items{{"a", 3}, {"b", 2}, {"a", 1}, {"b", 4}}
	original := slices.Clone(input)

	var calls []int
	got := GroupBy(input, func(v item) string {
		calls = append(calls, v.ID)
		return v.Team
	})

	want := map[string][]item{
		"a": {{"a", 3}, {"a", 1}},
		"b": {{"b", 2}, {"b", 4}},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if !slices.Equal(calls, []int{3, 2, 1, 4}) {
		t.Fatalf("key calls: %v", calls)
	}

	got["a"][0].ID = 99
	if !slices.Equal(input, original) {
		t.Fatalf("input was modified: %v", input)
	}
}

func TestGroupByEmpty(t *testing.T) {
	for _, input := range [][]int{nil, {}} {
		got := GroupBy(input, func(v int) int {
			t.Fatal("key called for empty input")
			return v
		})
		if got == nil || len(got) != 0 {
			t.Fatalf("got %#v, want non-nil empty map", got)
		}
	}
}

func TestHasDuplicates(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		want  bool
	}{
		{"nil", nil, false},
		{"empty", []int{}, false},
		{"single", []int{3}, false},
		{"distinct", []int{3, 1, 2}, false},
		{"adjacent", []int{3, 3, 1}, true},
		{"non-adjacent", []int{3, 1, 3}, true},
		{"zero", []int{0, 1, 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := slices.Clone(tt.input)
			if got := HasDuplicates(tt.input); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if !slices.Equal(tt.input, original) {
				t.Errorf("input changed: %v", tt.input)
			}
		})
	}

	type key struct {
		Name string
		ID   int
	}
	type keys []key

	if !HasDuplicates(keys{{"a", 1}, {"b", 1}, {"a", 1}}) {
		t.Error("missed duplicate struct")
	}
	if HasDuplicates(keys{{"a", 1}, {"a", 2}}) {
		t.Error("distinct structs were duplicates")
	}
	if HasDuplicates([]float64{math.NaN(), math.NaN()}) {
		t.Error("NaNs must follow == semantics")
	}
}

func TestUnique(t *testing.T) {
	type ints []int
	tests := []struct {
		name        string
		input, want ints
	}{
		{"nil", nil, nil},
		{"empty", ints{}, ints{}},
		{"single", ints{3}, ints{3}},
		{"distinct", ints{3, 1, 2}, ints{3, 1, 2}},
		{"non-adjacent", ints{3, 1, 3, 2, 1}, ints{3, 1, 2}},
		{"all equal", ints{0, 0, 0}, ints{0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := slices.Clone(tt.input)
			got := Unique(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
			if len(got) > 0 {
				got[0] = 99
			}
			if !slices.Equal(tt.input, original) {
				t.Fatalf("output aliases input: %v", tt.input)
			}
		})
	}

	type key struct {
		Name string
		ID   int
	}

	input := []key{{"a", 1}, {"a", 2}, {"a", 1}}
	if got := Unique(input); !slices.Equal(got, input[:2]) {
		t.Errorf("structs: %v", got)
	}

	got := Unique([]float64{math.NaN(), 1, math.NaN(), 1})
	if len(got) != 3 || !math.IsNaN(got[0]) || got[1] != 1 || !math.IsNaN(got[2]) {
		t.Errorf("unexpected NaN handling: %v", got)
	}
}
