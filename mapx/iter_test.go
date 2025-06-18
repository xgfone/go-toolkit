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

package mapx

import (
	"fmt"
	"slices"
)

func ExampleAll() {
	intm := map[int]int{1: 1, 2: 2, 3: 3}
	pairs := slices.Collect(All(intm))
	slices.SortFunc(pairs, func(a, b Pair[int, int]) int { return a.Key - b.Key })

	for _, p := range pairs {
		fmt.Printf("%d -> %d\n", p.Key, p.Value)
	}

	var keys []int
	All(intm)(func(p Pair[int, int]) bool {
		keys = append(keys, p.Key)
		return false // Only append one key randomly
	})
	fmt.Println("KeyNum:", len(keys))

	// Output:
	// 1 -> 1
	// 2 -> 2
	// 3 -> 3
	// KeyNum: 1
}
