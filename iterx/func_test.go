// Copyright 2025 xgfone
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

package iterx

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

func ExampleCountFunc() {
	// Slice
	{
		ints := []int{1, 2, 3, 4, 5}
		seq := slices.Values(ints)

		c0 := CountFunc(seq, func(v int) bool { return v > 0 })
		c1 := CountFunc(seq, func(v int) bool { return v > 2 })
		c2 := CountFunc(seq, func(v int) bool { return v > 5 })

		fmt.Println(c0)
		fmt.Println(c1)
		fmt.Println(c2)
	}

	// Map
	{
		intm := map[int]int{1: 1, 2: 2, 3: 3}
		seq := maps.Values(intm)

		c0 := CountFunc(seq, func(v int) bool { return v > 0 })
		c1 := CountFunc(seq, func(v int) bool { return v > 1 })
		c2 := CountFunc(seq, func(v int) bool { return v > 3 })

		fmt.Println(c0)
		fmt.Println(c1)
		fmt.Println(c2)
	}

	// Output:
	// 5
	// 3
	// 0
	// 3
	// 2
	// 0
}

func ExampleSumFunc() {
	ints1 := []int{1, 2, 3, 4}
	sum1 := SumFunc(slices.Values(ints1), func(v int) int { return v })
	fmt.Println(sum1)

	ints2 := []int64{1, 2, 3, 4}
	sum2 := SumFunc(slices.Values(ints2), func(v int64) int { return int(v) })
	fmt.Println(sum2)

	// Output:
	// 10
	// 10
}

func ExampleFilter() {
	ints := []int64{1, 2, 3, 4}
	iter := Filter(slices.Values(ints), func(v int64) bool { return v%2 == 0 })
	ints = slices.Collect(iter)
	fmt.Println(ints)

	var values []int64
	iter(func(v int64) bool {
		values = append(values, v)
		return false
	})
	fmt.Println(values)

	// Output:
	// [2 4]
	// [2]
}

func ExampleFilter2() {
	ints := []int64{1, 2, 3, 4}
	iter := Filter2(slices.All(ints), func(_ int, v int64) bool { return v%2 == 0 })
	ints = slices.Collect(Values(iter))
	fmt.Println(ints)

	var values []int64
	iter(func(_ int, v int64) bool {
		values = append(values, v)
		return false
	})
	fmt.Println(values)

	// Output:
	// [2 4]
	// [2]
}

func ExampleTo() {
	ints := []int64{1, 2, 3}
	iter := To(slices.Values(ints), func(v int64) string {
		return strconv.FormatInt(v*v, 10)
	})
	strs := slices.Collect(iter)
	fmt.Println(strs)

	var values []string
	iter(func(v string) bool {
		values = append(values, v)
		return false
	})
	fmt.Println(values)

	// Output:
	// [1 4 9]
	// [1]
}
