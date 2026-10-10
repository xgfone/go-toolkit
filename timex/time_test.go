// Copyright 2024～2026 xgfone
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

func TestSetNowFunc(t *testing.T) {
	// Save original now function
	originalNow := _now
	defer func() {
		_now = originalNow
	}()

	// Test setting valid function
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("unexpected panic: %v", r)
			}
		}()
		SetNowFunc(time.Now)
	}()

	// Test setting nil function should panic
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, but got none")
			}
		}()
		SetNowFunc(nil)
	}()
}

func TestNow(t *testing.T) {
	// Save the original local time zone and clock.
	originalLocal := time.Local
	originalNow := _now
	defer func() {
		time.Local = originalLocal
		_now = originalNow
	}()

	// The default clock follows the application's configured local time zone.
	time.Local = time.FixedZone("TestZone", 3600*8)
	before := time.Now()
	now := Now()
	after := time.Now()
	if now.Before(before) || now.After(after) {
		t.Errorf("expected time between %v and %v, got %v", before, after, now)
	}
	if now.Location() != time.Local {
		t.Errorf("expected location %v, got %v", time.Local, now.Location())
	}

	// A custom clock returns a predictable time without converting its location.
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	SetNowFunc(func() time.Time {
		return fixedTime
	})

	for range 2 {
		if got := Now(); got != fixedTime {
			t.Errorf("expected %v, got %v", fixedTime, got)
		}
	}
}

func TestToToday(t *testing.T) {
	testTime := time.Date(2024, 12, 25, 15, 30, 45, 123456789, time.UTC)
	expected := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)

	result := ToToday(testTime)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}

	// Check time components
	if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 || result.Nanosecond() != 0 {
		t.Errorf("expected 00:00:00.000000000, got %02d:%02d:%02d.%09d",
			result.Hour(), result.Minute(), result.Second(), result.Nanosecond())
	}
}

func TestToday(t *testing.T) {
	originalNow := _now
	defer func() {
		_now = originalNow
	}()

	loc := time.FixedZone("TestZone", 3600*8)
	// This is still the previous year in UTC; Today must use the clock's location.
	now := time.Date(2025, 1, 1, 0, 30, 45, 123456789, loc)
	SetNowFunc(func() time.Time { return now })

	want := time.Date(2025, 1, 1, 0, 0, 0, 0, loc)
	if got := Today(); got != want {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestMeasure(t *testing.T) {
	f := func() {
		time.Sleep(time.Millisecond * 500)
	}

	cost := Measure(f)
	if cost < time.Millisecond*500 || cost > time.Millisecond*600 {
		t.Errorf("expect cost %dms, but got %dms", 500, cost.Milliseconds())
	}

	// Test with function that panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic, but got none")
			}
		}()

		Measure(func() {
			panic("test panic")
		})
	}()
}
