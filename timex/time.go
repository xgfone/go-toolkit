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

package timex

import "time"

// Pre-define some constant durations
const (
	Day  = time.Hour * 24
	Week = Day * 7
)

var _now = time.Now

// SetNowFunc replaces the clock used by [Now] and functions that call it,
// such as [Today]. Its main purpose is to provide predictable times in tests.
// Callers that do not need a replaceable clock should use [time.Now] directly.
//
// The default clock is [time.Now]. Call SetNowFunc(time.Now) to restore it.
// SetNowFunc panics if now is nil.
//
// SetNowFunc must not be called concurrently with [Now] or functions that call
// it. The supplied clock must be safe for concurrent use if [Now] is called
// concurrently.
func SetNowFunc(now func() time.Time) {
	if now == nil {
		panic("timex.SetNowFunc: now function is nil")
	}
	_now = now
}

// Now returns the time from the clock set by [SetNowFunc], which defaults to
// [time.Now]. It returns the clock's result without converting its location.
//
// Now is intended for code that needs a replaceable clock for predictable tests.
// Otherwise, use [time.Now] directly.
func Now() time.Time {
	return _now()
}

// Today returns midnight on the date returned by [Now], in that time's location.
func Today() time.Time {
	return ToToday(Now())
}

// ToToday converts the any [time.Time] to today at 00:00:00.
func ToToday(any time.Time) (today time.Time) {
	return time.Date(any.Year(), any.Month(), any.Day(), 0, 0, 0, 0, any.Location())
}

// Measure calculates the duration of the function execution.
func Measure(f func()) (cost time.Duration) {
	start := time.Now()
	defer func() {
		cost = time.Since(start)
	}()
	f()
	return
}
