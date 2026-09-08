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

package bytex

import (
	"bytes"
	"testing"
)

func TestRegressionQuotedMarkerThenComment(t *testing.T) {
	in := []byte(`{"url":"https://example.com"} // comment`)
	want := []byte("{\"url\":\"https://example.com\"}\n")
	if got := RemoveLineComments(in, CommentSlashes); !bytes.Equal(got, want) {
		t.Fatalf("trailing comment survived quoted marker: %s", got)
	}
}

func TestRegressionEscapedQuote(t *testing.T) {
	in := []byte(`{"s":"a\"//b"}`)
	want := append(append([]byte{}, in...), '\n')
	if got := RemoveLineComments(in, CommentSlashes); !bytes.Equal(got, want) {
		t.Fatalf("valid quoted JSON corrupted: %s", got)
	}
}
