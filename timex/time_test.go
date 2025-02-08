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

package timex

import (
	"testing"
	"time"
)

func TestToday(t *testing.T) {
	Location = time.Local

	now := time.Now()
	nowdate := now.Format(time.DateOnly)

	today := Today()
	if date := today.Format(time.DateOnly); date != nowdate {
		t.Errorf("expect date '%s', but got '%s'", nowdate, date)
	}
	if time := today.Format(time.TimeOnly); time != "00:00:00" {
		t.Errorf("expect time '%s', but got '%s'", "00:00:00", time)
	}
	if nsec := today.Nanosecond(); nsec != 0 {
		t.Errorf("expect nanosecond %d, but got %d", 0, nsec)
	}
}

func TestUnix(t *testing.T) {
	Location = time.UTC

	expected := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	gottime := Unix(1735689600, 0)
	if expected != gottime {
		t.Errorf("expect time '%s', but got '%s'", expected.Format(time.RFC3339), gottime.Format(time.RFC3339))
	}
}
