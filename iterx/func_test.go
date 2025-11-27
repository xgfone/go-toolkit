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

//go:build go1.23

package iterx

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
)

func ExampleAll() {
	{
		ints := []int{1, 2, 3}
		seq := slices.Values(ints)

		ok0 := All(seq, func(v int) bool { return v > 0 })
		ok1 := All(seq, func(v int) bool { return v > 1 })
		ok2 := All(seq, func(v int) bool { return v > 2 })
		ok3 := All(seq, func(v int) bool { return v > 3 })

		fmt.Println(ok0)
		fmt.Println(ok1)
		fmt.Println(ok2)
		fmt.Println(ok3)
	}

	{
		intm := map[int]int{1: 1, 2: 2, 3: 3}
		seq := maps.Values(intm)

		ok0 := All(seq, func(v int) bool { return v > 0 })
		ok1 := All(seq, func(v int) bool { return v > 1 })
		ok2 := All(seq, func(v int) bool { return v > 2 })
		ok3 := All(seq, func(v int) bool { return v > 3 })

		fmt.Println(ok0)
		fmt.Println(ok1)
		fmt.Println(ok2)
		fmt.Println(ok3)
	}

	// Output:
	// true
	// false
	// false
	// false
	// true
	// false
	// false
	// false
}

func ExampleAny() {
	// Slice
	{
		ints := []int{1, 2, 3}
		seq := slices.Values(ints)

		ok0 := Any(seq, func(v int) bool { return v > 0 })
		ok1 := Any(seq, func(v int) bool { return v > 1 })
		ok2 := Any(seq, func(v int) bool { return v > 2 })
		ok3 := Any(seq, func(v int) bool { return v > 3 })

		fmt.Println(ok0)
		fmt.Println(ok1)
		fmt.Println(ok2)
		fmt.Println(ok3)
	}

	// Map
	{
		intm := map[int]int{1: 1, 2: 2, 3: 3}
		seq := maps.Values(intm)

		ok0 := Any(seq, func(v int) bool { return v > 0 })
		ok1 := Any(seq, func(v int) bool { return v > 1 })
		ok2 := Any(seq, func(v int) bool { return v > 2 })
		ok3 := Any(seq, func(v int) bool { return v > 3 })

		fmt.Println(ok0)
		fmt.Println(ok1)
		fmt.Println(ok2)
		fmt.Println(ok3)
	}

	// Output:
	// true
	// true
	// true
	// false
	// true
	// true
	// true
	// false
}

func ExampleSum() {
	ints1 := []int{1, 2, 3, 4}
	sum1 := Sum(slices.Values(ints1), func(v int) int { return v })
	fmt.Println(sum1)

	ints2 := []int64{1, 2, 3, 4}
	sum2 := Sum(slices.Values(ints2), func(v int64) int { return int(v) })
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
	ints = slices.Collect(Seq(iter, func(_ int, v int64) int64 { return v }))
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

func ExampleMap() {
	ints := []int64{1, 2, 3}
	iter := Map(slices.Values(ints), func(v int64) string { return strconv.FormatInt(v*v, 10) })
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

func ExampleSeq() {
	ints := []int64{1, 2, 3}
	iters := Seq(slices.All(ints), func(_ int, v int64) string { return strconv.FormatInt(v*v, 10) })
	strs := slices.Collect(iters)

	intm := map[string]int64{"a": 1, "b": 2, "c": 3}
	iterm := Seq(maps.All(intm), func(_ string, v int64) string { return strconv.FormatInt(v*2, 10) })
	strm := slices.Collect(iterm)
	slices.Sort(strm)

	fmt.Println(strs)
	fmt.Println(strm)

	var values []string
	iters(func(v string) bool {
		values = append(values, v)
		return false
	})
	fmt.Println(values)

	// Output:
	// [1 4 9]
	// [2 4 6]
	// [1]
}

func ExampleSeq2() {
	ints := []int64{1, 2, 3}
	iters := Seq2(slices.Values(ints), func(v int64) (int64, string) { return v, strconv.FormatInt(v*v, 10) })
	strs := maps.Collect(iters)

	intm := map[string]int64{"a": 1, "b": 2, "c": 3}
	iterm := Seq2(maps.Values(intm), func(v int64) (int64, string) { return v, strconv.FormatInt(v*2, 10) })
	strm := maps.Collect(iterm)

	fmt.Println(strs)
	fmt.Println(strm)

	var values []string
	iters(func(_ int64, v string) bool {
		values = append(values, v)
		return false
	})
	fmt.Println(values)

	// Output:
	// map[1:1 2:4 3:9]
	// map[1:2 2:4 3:6]
	// [1]
}
