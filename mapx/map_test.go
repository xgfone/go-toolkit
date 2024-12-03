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

package mapx

import (
	"fmt"
	"slices"
	"sort"
)

func ExampleConvert() {
	type Maps map[string]int

	intmap1 := Maps{"a": 1, "b": 2}
	intmap2 := map[string]int{"a": 3, "b": 4}

	int64map1 := Convert(intmap1, func(k string, v int) (string, int64) { return k, int64(v) })
	int64map2 := Convert(intmap2, func(k string, v int) (string, int64) { return k, int64(v) })

	fmt.Printf("%T\n", int64map1)
	fmt.Printf("%T\n", int64map2)
	fmt.Printf("%s=%v\n", "a", int64map1["a"])
	fmt.Printf("%s=%v\n", "b", int64map1["b"])
	fmt.Printf("%s=%v\n", "a", int64map2["a"])
	fmt.Printf("%s=%v\n", "b", int64map2["b"])

	// Output:
	// map[string]int64
	// map[string]int64
	// a=1
	// b=2
	// a=3
	// b=4
}

func ExampleKeys() {
	intmap := map[int]int{1: 11, 2: 22}
	ints := Keys(intmap)
	sort.Ints(ints)
	fmt.Println(ints)

	strmap := map[string]string{"a": "aa", "b": "bb"}
	strs := Keys(strmap)
	sort.Strings(strs)
	fmt.Println(strs)

	// Output:
	// [1 2]
	// [a b]
}

func ExampleValues() {
	intmap := map[int]int{1: 11, 2: 22}
	ints := Values(intmap)
	sort.Ints(ints)
	fmt.Println(ints)

	strmap := map[string]string{"a": "aa", "b": "bb"}
	strs := Values(strmap)
	sort.Strings(strs)
	fmt.Println(strs)

	// Output:
	// [11 22]
	// [aa bb]
}

func ExampleKeysFunc() {
	type Key struct {
		K string
		V int32
	}
	maps := map[Key]bool{
		{K: "a", V: 1}: true,
		{K: "b", V: 2}: true,
		{K: "c", V: 3}: true,
	}

	keys := KeysFunc(maps, func(k Key) string { return k.K })
	slices.Sort(keys)
	fmt.Println(keys)

	// Output:
	// [a b c]
}

func ExampleValuesFunc() {
	type Value struct {
		V int
	}
	maps := map[string]Value{
		"a": {V: 1},
		"b": {V: 2},
		"c": {V: 3},
	}

	values := ValuesFunc(maps, func(v Value) int { return v.V })
	slices.Sort(values)
	fmt.Println(values)

	// Output:
	// [1 2 3]
}
