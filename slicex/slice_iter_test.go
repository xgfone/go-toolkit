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
	"slices"
)

func ExampleValuesFunc() {
	seq := ValuesFunc([]int{1, 2, 3}, func(i int) string {
		return fmt.Sprintf("v%d", i)
	})

	fmt.Println(slices.Collect(seq))

	for v := range seq {
		fmt.Println(v)
		break
	}

	// Output:
	// [v1 v2 v3]
	// v1
}
