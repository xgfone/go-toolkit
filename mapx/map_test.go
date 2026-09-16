// Copyright 2024~2026 xgfone
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

import "fmt"

func _convert(k string, v int) (string, int64) { return k, int64(v) }

func ExampleTo() {
	type Maps map[string]int

	var nilmap1 Maps
	nilmap2 := To(nilmap1, _convert)
	if nilmap2 == nil {
		fmt.Println("nil")
	} else {
		fmt.Printf("%v\n", nilmap2)
	}

	int64map1 := To(Maps{"a": 1, "b": 2}, _convert)
	int64map2 := To(map[string]int{"a": 3, "b": 4}, _convert)

	fmt.Printf("%T\n", int64map1)
	fmt.Printf("%T\n", int64map2)
	fmt.Printf("%s=%v\n", "a", int64map1["a"])
	fmt.Printf("%s=%v\n", "b", int64map1["b"])
	fmt.Printf("%s=%v\n", "a", int64map2["a"])
	fmt.Printf("%s=%v\n", "b", int64map2["b"])

	// Output:
	// nil
	// map[string]int64
	// map[string]int64
	// a=1
	// b=2
	// a=3
	// b=4
}
