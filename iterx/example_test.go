// Copyright 2025~2026 xgfone
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

package iterx_test

import (
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/xgfone/go-toolkit/iterx"
)

func Example() {
	// Parse valid integers, keep the first two, and collect the result.
	numbers := iterx.FilterTo(slices.Values([]string{"10", "bad", "20", "30"}), func(s string) (int, bool) {
		n, err := strconv.Atoi(s)
		return n, err == nil
	})
	fmt.Println(slices.Collect(iterx.Take(numbers, 2)))
	// Output: [10 20]
}

func ExampleSum() {
	fmt.Println(iterx.Sum(slices.Values([]int{1, 2, 3})))
	// Output: 6
}

func ExampleCount() {
	fmt.Println(iterx.Count(slices.Values([]int{1, 2, 3})))
	// Output: 3
}

func ExampleTo2() {
	seq := iterx.To2(slices.All([]string{"one", "two"}), func(i int, s string) (string, int) {
		return s, i + 1
	})
	fmt.Println(maps.Collect(seq))
	// Output: map[one:1 two:2]
}

func ExampleKeys() {
	seq := iterx.Filter2(slices.All([]string{"one", "two", "three"}), func(_ int, s string) bool {
		return len(s) == 3
	})
	fmt.Println(slices.Collect(iterx.Keys(seq)))
	// Output: [0 1]
}

func ExampleValues() {
	seq := iterx.Filter2(slices.All([]string{"one", "two", "three"}), func(i int, _ string) bool {
		return i > 0
	})
	fmt.Println(slices.Collect(iterx.Values(seq)))
	// Output: [two three]
}

func ExampleFilterTo() {
	seq := iterx.FilterTo(slices.Values([]string{"1", "bad", "3"}), func(s string) (int, bool) {
		n, err := strconv.Atoi(s)
		return n, err == nil
	})
	fmt.Println(slices.Collect(seq))
	// Output: [1 3]
}

func ExampleFilterTo2() {
	seq := iterx.FilterTo2(slices.All([]string{"1", "bad", "3"}), func(i int, s string) (int, int, bool) {
		n, err := strconv.Atoi(s)
		return i, n, err == nil
	})
	fmt.Println(maps.Collect(seq))
	// Output: map[0:1 2:3]
}

func ExampleFind() {
	fmt.Println(iterx.Find(slices.Values([]int{1, 2, 3}), func(v int) bool { return v%2 == 0 }))
	// Output: 2 true
}

func ExampleTake() {
	fmt.Println(slices.Collect(iterx.Take(slices.Values([]int{1, 2, 3}), 2)))
	// Output: [1 2]
}

func ExampleDrop() {
	fmt.Println(slices.Collect(iterx.Drop(slices.Values([]int{1, 2, 3}), 2)))
	// Output: [3]
}

func ExampleConcat() {
	seq := iterx.Concat(slices.Values([]int{1, 2}), slices.Values([]int{3, 4}))
	fmt.Println(slices.Collect(seq))
	// Output: [1 2 3 4]
}

func ExampleReduce() {
	fmt.Println(iterx.Reduce(slices.Values([]int{1, 2, 3}), "values:", func(result string, v int) string {
		return result + " " + strconv.Itoa(v)
	}))
	// Output: values: 1 2 3
}
