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

package mapx

import "testing"

func TestNewSMap(t *testing.T) {
	m := NewSMap[int](10)
	if m == nil {
		t.Error("NewSMap return nil")
	}

	m["a"] = 123
	if len(m) != 1 {
		t.Errorf("expect len(m) == 1, got %d", len(m))
	}
}

func TestSMap_Get(t *testing.T) {
	m := SMap[int]{"a": 123}
	if v := m.Get("a"); v != 123 {
		t.Errorf("expect m.Get(\"a\") == 123, got %d", v)
	}
	if v := m.Get("b"); v != 0 {
		t.Errorf("expect m.Get(\"b\") == 0, got %d", v)
	}
}
