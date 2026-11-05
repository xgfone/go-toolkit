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

package timex

import "time"

// AddMonths returns t shifted by months calendar months. If t's day does not
// exist in the target month, the result uses that month's last day. Negative
// months move backward; zero returns t unchanged.
//
// AddMonths preserves t's location, clock time and nanoseconds, subject to
// [time.Date]'s handling of daylight saving transitions.
//
// Clamping is not reversible and does not retain an end-of-month preference:
// January 31 plus one month is February 28 in a non-leap year, and adding
// another month yields March 28. For recurring dates, use AddMonths(anchor, n)
// with the original anchor instead of repeatedly adding to the previous result.
func AddMonths(t time.Time, months int) time.Time {
	if months == 0 {
		return t
	}

	year, month, day := t.Date()
	year += months / 12
	month += time.Month(months % 12)

	// Month length must not depend on local midnight transitions.
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}

	return time.Date(year, month, day, t.Hour(), t.Minute(), t.Second(),
		t.Nanosecond(), t.Location())
}
