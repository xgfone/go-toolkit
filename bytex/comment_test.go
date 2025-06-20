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

package bytex

import "fmt"

func ExampleRemoveLineComments() {
	var singleLine = []byte(`123`)

	var slashOrig = []byte(`
// line comment 1
1
    /// line comment 2
2   ///// line tail comment 3
3
    4
	"//": "abc"
	"abc"  // the trailling comment containing "
5
`)

	var hashOrig = []byte(`
# line comment 1
1
	## line comment 2
2   #### line tail comment 3
3
    4
	"#": "abc"
	"abc"  # the trailling comment containing "
5
`)

	fmt.Println("Single Line:")
	fmt.Println(string(RemoveLineComments(singleLine, CommentHash)))

	fmt.Println("Hash Result:")
	fmt.Println(string(RemoveLineComments(hashOrig, CommentHash)))

	fmt.Println("Slash Result:")
	fmt.Println(string(RemoveLineComments(slashOrig, CommentSlashes)))

	// Output:
	// Single Line:
	// 123
	//
	// Hash Result:
	// 1
	// 2
	// 3
	//     4
	// 	"#": "abc"
	// 	"abc"
	// 5
	//
	// Slash Result:
	// 1
	// 2
	// 3
	//     4
	// 	"//": "abc"
	// 	"abc"
	// 5
}
