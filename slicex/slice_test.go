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

import "fmt"

func _convert(v int) int64        { return int64(v) }
func _filter(v int) (int64, bool) { return int64(v), v%2 == 0 }

func ExampleEmpty() {
	var vs []int
	fmt.Println(vs == nil, len(vs), cap(vs))

	vs = Empty(vs)
	fmt.Println(vs == nil, len(vs), cap(vs))

	vs = []int{1, 2}
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
