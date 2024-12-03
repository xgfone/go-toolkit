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

func TestDesensitizer(t *testing.T) {
	dephoner := NewDesensitizer(3, 4).WithChars("****")

	if s := dephoner.Desensitize("123"); s != "****" {
		t.Errorf("expect '%s', but got '%s'", "****", s)
	}

	if s := dephoner.Desensitize("1234567"); s != "****" {
		t.Errorf("expect '%s', but got '%s'", "****", s)
	}

	if s := dephoner.Desensitize("12345678"); s != "123****5678" {
		t.Errorf("expect '%s', but got '%s'", "123****5678", s)
	}

	if s := dephoner.Desensitize("1234567890"); s != "123****7890" {
		t.Errorf("expect '%s', but got '%s'", "123****7890", s)
	}

	denamer1 := NewDesensitizer(1, 0).WithChars("**")
	if s := denamer1.Desensitize("谢1谢2"); s != "谢**" {
		t.Errorf("expect '%s', but got '%s'", "谢**", s)
	}

	denamer2 := NewDesensitizer(0, 1).WithChars("**")
	if s := denamer2.Desensitize("1谢2谢"); s != "**谢" {
		t.Errorf("expect '%s', but got '%s'", "**谢", s)
	}
}
