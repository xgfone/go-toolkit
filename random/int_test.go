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

package random

import (
	"strconv"
	"testing"
)

func TestSeedString(t *testing.T) {
	v, err := strconv.ParseInt(SeedString(), 10, 64)
	if err != nil {
		t.Fatal(err)
	} else if v < 0 {
		t.Errorf("unexpect a negative integer %d", v)
	}
}

func TestIntN(t *testing.T) {
	for range 100 {
		v := IntN(10)
		if v < 0 || v >= 10 {
			t.Errorf("expect one of [0, 9], but got %v", v)
		}
	}
}
