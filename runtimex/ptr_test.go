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

package runtimex

import "fmt"

func ExampleIndiect() {
	var v *int
	fmt.Println(Indirect(v))

	v = new(int)
	*v = 123
	fmt.Println(Indirect(v))

	var s *string
	fmt.Println(Indirect(s))

	s = new(string)
	*s = "hello"
	fmt.Println(Indirect(s))

	// Output:
	// 0
	// 123
	//
	// hello
}
