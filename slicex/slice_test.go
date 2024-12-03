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

func ExampleConvert() {
	type Ints []int

	ints1 := []int{1, 2, 3}
	ints2 := Ints{4, 5, 6}
	int64s1 := Convert(ints1, func(v int) int64 { return int64(v) })
	int64s2 := Convert(ints2, func(v int) int64 { return int64(v) })

	fmt.Println(int64s1)
	fmt.Println(int64s2)

	// Output:
	// [1 2 3]
	// [4 5 6]
}

func ExampleMap() {
	slices := []int{1, 2}
	maps := Map(slices, func(v int) (int, string) { return v, fmt.Sprintf("%c", v+96) })
	fmt.Println(maps)

	// Output:
	// map[1:a 2:b]
}
