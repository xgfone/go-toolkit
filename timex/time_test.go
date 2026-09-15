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

func TestSetLocation(t *testing.T) {
	// Test normal case
	original := Location()
	defer SetLocation(original)

	newLoc := time.FixedZone("TestZone", 3600*8) // UTC+8
	SetLocation(newLoc)
	if Location() != newLoc {
		t.Errorf("expected location %v, got %v", newLoc, Location())
	}

	// Test panic with nil location
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil location, but got none")
		}
	}()
	SetLocation(nil)
}

func TestLocation(t *testing.T) {
	original := Location()
	defer SetLocation(original)

	// Default should be UTC
	if Location() != time.UTC {
		t.Errorf("expected default location UTC, got %v", Location())
	}

	// Change location and verify
	newLoc := time.Local
	SetLocation(newLoc)
	if Location() != newLoc {
		t.Errorf("expected location %v, got %v", newLoc, Location())
	}
}

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
	// Save original location and now function
	originalLoc := Location()
	originalNow := _now
	defer func() {
		SetLocation(originalLoc)
		_now = originalNow
	}()

	// Test with default now function
	SetLocation(time.UTC)
	SetNowFunc(nowloc)

	now1 := Now()
	time.Sleep(time.Millisecond * 10)
	now2 := Now()

	if now2.Before(now1) {
		t.Errorf("now2 should be after now1, got now1=%v, now2=%v", now1, now2)
	}

	// Test with custom now function
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	SetNowFunc(func() time.Time {
		return fixedTime
	})

	now3 := Now()
	if !now3.Equal(fixedTime) {
		t.Errorf("expected %v, got %v", fixedTime, now3)
	}
}

func TestToToday(t *testing.T) {
	// Save original location
	originalLoc := Location()
	defer SetLocation(originalLoc)

	SetLocation(time.UTC)

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
	// Save original location and now function
	originalLoc := Location()
	originalNow := _now
	defer func() {
		SetLocation(originalLoc)
		_now = originalNow
	}()

	SetLocation(time.Local)
	SetNowFunc(nowloc)

	now := Now()
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
	// Save original location
	originalLoc := Location()
	defer SetLocation(originalLoc)

	SetLocation(time.UTC)

	expected := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	gottime := Unix(1735689600, 0)
	if expected != gottime {
		t.Errorf("expect time '%s', but got '%s'", expected.Format(time.RFC3339), gottime.Format(time.RFC3339))
	}

	// Test with different location
	SetLocation(time.FixedZone("TestZone", 3600*8)) // UTC+8
	gottime2 := Unix(1735689600, 0)
	if gottime2.Location() != Location() {
		t.Errorf("expected location %v, got %v", Location(), gottime2.Location())
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
