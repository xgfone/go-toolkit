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

package priority

import "testing"

type _NoPriority int

type _HasPriority int

func (p _HasPriority) Priority() int {
	return int(p)
}

func TestPriority(t *testing.T) {
	var v1 _NoPriority
	if p := Get(v1); p != 1 {
		t.Errorf("expect get priority %d, but got %d", 1, p)
	}

	var v2 _HasPriority = 2
	if p := Get(v2); p != 2 {
		t.Errorf("expect get priority %d, but got %d", 2, p)
	}
}
