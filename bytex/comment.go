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

package bytex

import "bytes"

// Pre-define some comment characters.
var (
	CommentHash    = []byte("#")
	CommentSlashes = []byte("//")
)

// RemoveLineComments is a simple function to remove the whole line comment
// that the first non-white character starts with comments,
// and the similar line tail comment.
func RemoveLineComments(data, comments []byte) []byte {
	result := make([]byte, 0, len(data))
	for len(data) > 0 {
		var line []byte
		if index := bytes.IndexByte(data, '\n'); index == -1 {
			line = data
			data = nil
		} else {
			line = data[:index]
			data = data[index+1:]
		}

		orig := line
		line = bytes.TrimLeft(line, " \t")
		if len(line) == 0 || bytes.HasPrefix(line, comments) {
			continue
		}

		// Line Suffix Comment
		if index := lineCommentIndex(orig, comments); index == -1 {
			result = append(result, orig...)
		} else {
			result = append(result, bytes.TrimRight(orig[:index], " \t")...)
		}
		result = append(result, '\n')
	}
	return result
}

func lineCommentIndex(line, comments []byte) int {
	quoted := false
	for i := 0; i < len(line); i++ {
		if quoted {
			switch line[i] {
			case '\\':
				i++ // Skip the escaped byte, including an escaped quote.

			case '"':
				quoted = false
			}
		} else if bytes.HasPrefix(line[i:], comments) {
			return i
		} else if line[i] == '"' {
			quoted = true
		}
	}
	return -1
}
