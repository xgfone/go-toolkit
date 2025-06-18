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

package stringx

import "testing"

func TestTruncate(t *testing.T) {
	if s := Truncate("中国", 0); s != "" {
		t.Errorf("expect empty string, but got '%s'", s)
	}

	if s := Truncate("中国", 1); s != "中" {
		t.Errorf("expect '%s', but got '%s'", "中", s)
	}

	if s := Truncate("中国", 3); s != "中国" {
		t.Errorf("expect '%s', but got '%s'", "中国", s)
	}

	if s := Truncate("中国", 10); s != "中国" {
		t.Errorf("expect '%s', but got '%s'", "中国", s)
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expect a panic, but got not")
			}
		}()
		Truncate("abc", -1)
	}()
}
