// Copyright 2024~2025 xgfone
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

// Some variables.
var (
	// Default: time.RFC3339Nano
	Format = time.RFC3339Nano

	// Defaults: []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"}
	Formats = []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02"}

	// Default: time.UTC
	Location = time.UTC
)

// Now is used to customize the now time.
//
// Default: time.Now
var Now = time.Now

// Today returns the today local time at 00:00:00.
func Today() time.Time {
	return ToToday(Now())
}

// ToToday converts the any time.Time to today at 00:00:00.
func ToToday(any time.Time) (today time.Time) {
	return time.Date(any.Year(), any.Month(), any.Day(), 0, 0, 0, 0, any.Location())
}
