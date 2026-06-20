// Copyright 2024 xgfone
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
	"strings"
	"testing"
)

func _convert2(_, v int) int64    { return int64(v) }
func _convert(v int) int64        { return int64(v) }
func _filter(v int) (int64, bool) { return int64(v), v%2 == 0 }

func ExampleEmpty() {
	var vs []int
	fmt.Println(vs == nil, len(vs), cap(vs))

	vs = Empty(vs)
	fmt.Println(vs == nil, len(vs), cap(vs))

	vs = Empty([]int{1, 2})
	fmt.Println(vs)

	// Output:
	// true 0 0
	// false 0 0
	// [1 2]
}

func ExampleConvert() {
	type Ints []int

	fmt.Println(Convert([]int{1, 2, 3}, _convert))
	fmt.Println(Convert(Ints{4, 5, 6}, _convert))

	if s := Convert(Ints(nil), _convert); s == nil {
		fmt.Println(nil)
	} else {
		fmt.Println(s)
	}

	// Output:
	// [1 2 3]
	// [4 5 6]
	// <nil>
}

func ExampleTo() {
	type Ints []int

	fmt.Println(To([]int{1, 2, 3}, _convert))
	fmt.Println(To(Ints{4, 5, 6}, _convert))

	if s := Convert(Ints(nil), _convert); s == nil {
		fmt.Println(nil)
	} else {
		fmt.Println(s)
	}

	// Output:
	// [1 2 3]
	// [4 5 6]
	// <nil>
}

func ExampleTo2() {
	type Ints []int

	fmt.Println(To2([]int{1, 2, 3}, _convert2))
	fmt.Println(To2(Ints{4, 5, 6}, _convert2))

	if s := To2(Ints(nil), _convert2); s == nil {
		fmt.Println(nil)
	} else {
		fmt.Println(s)
	}

	// Output:
	// [1 2 3]
	// [4 5 6]
	// <nil>
}

func ExampleFilter() {
	type Ints []int

	fmt.Println(Filter([]int{1, 2, 3}, _filter))
	fmt.Println(Filter(Ints{4, 5, 6}, _filter))

	if s := Filter(Ints(nil), _filter); s == nil {
		fmt.Println(nil)
	} else {
		fmt.Println(s)
	}

	// Output:
	// [2]
	// [4 6]
	// <nil>
}

func ExampleMap() {
	slices := []int{1, 2}
	maps := Map(slices, func(v int) (int, string) { return v, fmt.Sprintf("%c", v+96) })
	fmt.Println(maps)

	// Output:
	// map[1:a 2:b]
}

func ExampleMerge() {
	type Ints []int

	// Test with no arguments
	fmt.Println(Merge[Ints]() == nil)

	// Test with single empty slice
	fmt.Println(Merge(Ints{}))

	// Test with multiple empty slices
	fmt.Println(Merge(Ints{}, Ints{}, Ints{}))

	// Test with single non-empty slice
	fmt.Println(Merge(Ints{1, 2, 3}))

	// Test with multiple non-empty slices
	fmt.Println(Merge(Ints{1, 2}, Ints{3, 4}, Ints{5, 6}))

	// Test with mixed empty and non-empty slices
	fmt.Println(Merge(Ints{}, Ints{1, 2}, Ints{}, Ints{3, 4}, Ints{}))

	// Test with nil slices
	var nilSlice Ints
	fmt.Println(Merge(nilSlice))
	fmt.Println(Merge(nilSlice, Ints{1, 2}, nilSlice))

	// Test with different slice types
	fmt.Println(Merge([]string{"a", "b"}, []string{"c", "d"}))

	// Output:
	// true
	// []
	// []
	// [1 2 3]
	// [1 2 3 4 5 6]
	// [1 2 3 4]
	// []
	// [1 2]
	// [a b c d]
}

func ExampleContainsAll() {
	fmt.Println(ContainsAll([]int{1, 2, 3}, []int{3, 1}))
	fmt.Println(ContainsAll([]int{1, 2}, []int{2, 2}))
	fmt.Println(ContainsAll([]int{1, 2}, []int{3}))

	// Output:
	// true
	// true
	// false
}

func ExampleContainsAllFunc() {
	allowed := []string{"X-Token", "X-Trace"}
	requested := []string{"x-trace", "x-token"}
	fmt.Println(ContainsAllFunc(allowed, requested, strings.EqualFold))

	// Output:
	// true
}

func TestContainsAll(t *testing.T) {
	type ints []int

	tests := []struct {
		name     string
		superset ints
		subset   []int
		want     bool
	}{
		{
			name:     "all contained",
			superset: ints{1, 2, 3},
			subset:   []int{3, 1},
			want:     true,
		},
		{
			name:     "missing element",
			superset: ints{1, 2, 3},
			subset:   []int{4},
		},
		{
			name:     "repeated subset element is ignored",
			superset: ints{1, 2},
			subset:   []int{2, 2},
			want:     true,
		},
		{
			name:     "repeated superset element is ignored",
			superset: ints{1, 1},
			subset:   []int{1},
			want:     true,
		},
		{
			name:     "empty subset",
			superset: ints{1},
			subset:   nil,
			want:     true,
		},
		{
			name:     "non-empty subset with empty superset",
			superset: nil,
			subset:   []int{1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAll(tt.superset, tt.subset); got != tt.want {
				t.Fatalf("unexpected result: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsAllFunc(t *testing.T) {
	tests := []struct {
		name     string
		superset []string
		subset   []string
		want     bool
	}{
		{
			name:     "case-insensitive match",
			superset: []string{"X-Token", "X-Trace"},
			subset:   []string{"x-trace", "x-token"},
			want:     true,
		},
		{
			name:     "missing element",
			superset: []string{"X-Token"},
			subset:   []string{"x-token", "x-trace"},
		},
		{
			name:     "repeated subset element is ignored",
			superset: []string{"X-Token"},
			subset:   []string{"x-token", "X-TOKEN"},
			want:     true,
		},
		{
			name:   "empty subset",
			subset: nil,
			want:   true,
		},
		{
			name:   "non-empty subset with empty superset",
			subset: []string{"x-token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsAllFunc(tt.superset, tt.subset, strings.EqualFold); got != tt.want {
				t.Fatalf("unexpected result: got %v, want %v", got, tt.want)
			}
		})
	}
}
