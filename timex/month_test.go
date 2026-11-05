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

package timex_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/xgfone/go-toolkit/timex"
)

func TestAddMonths(t *testing.T) {
	clock := 12*time.Hour + 34*time.Minute + 56*time.Second + 123456789*time.Nanosecond
	for _, tt := range []struct {
		name   string
		from   string
		months int
		want   string
	}{
		{"ordinary forward", "2026-01-15", 1, "2026-02-15"},
		{"ordinary backward", "2026-03-15", -1, "2026-02-15"},
		{"short month forward", "2026-01-31", 1, "2026-02-28"},
		{"short month backward", "2026-03-31", -1, "2026-02-28"},
		{"leap month forward", "2028-01-31", 1, "2028-02-29"},
		{"leap month backward", "2028-03-31", -1, "2028-02-29"},
		{"thirty days forward", "2026-03-31", 1, "2026-04-30"},
		{"thirty days backward", "2026-10-31", -1, "2026-09-30"},
		{"year rollover forward", "2026-12-31", 1, "2027-01-31"},
		{"year rollover backward", "2026-01-31", -1, "2025-12-31"},
		{"multiple months", "2026-01-31", 2, "2026-03-31"},
		{"multiple months backward", "2026-03-31", -2, "2026-01-31"},
		{"multiple years forward", "2026-01-31", 25, "2028-02-29"},
		{"multiple years backward", "2026-03-31", -25, "2024-02-29"},
		{"leap day forward", "2028-02-29", 12, "2029-02-28"},
		{"leap day backward", "2028-02-29", -12, "2027-02-28"},
		{"century without leap day", "1900-01-31", 1, "1900-02-28"},
		{"century with leap day", "2000-01-31", 1, "2000-02-29"},
		{"future century", "2100-03-31", -1, "2100-02-28"},
		{"no month-end preference", "2026-02-28", 1, "2026-03-28"},
		{"zero months", "2026-01-31", 0, "2026-01-31"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			start, err := time.Parse(time.DateOnly, tt.from)
			if err != nil {
				t.Fatal(err)
			}

			want, err := time.Parse(time.DateOnly, tt.want)
			if err != nil {
				t.Fatal(err)
			}

			start = start.Add(clock)
			want = want.Add(clock)
			if got := timex.AddMonths(start, tt.months); got != want {
				t.Fatalf("AddMonths(%v, %d) = %v; want %v", start, tt.months, got, want)
			}
		})
	}
}

func TestAddMonthsZero(t *testing.T) {
	zone := time.FixedZone("custom", 5*3600+45*60)
	for _, start := range []time.Time{
		{},
		time.Now(), // Includes the monotonic reading, which zero must retain.
		time.Date(2026, time.January, 31, 0, 0, 0, 123456789, zone),
	} {
		if got := timex.AddMonths(start, 0); got != start {
			t.Errorf("AddMonths(%v, 0) changed the input: %v", start, got)
		}
	}
}

func TestAddMonthsZeroTime(t *testing.T) {
	for _, tt := range []struct {
		months int
		want   time.Time
	}{
		{1, time.Date(1, time.February, 1, 0, 0, 0, 0, time.UTC)},
		{-1, time.Date(0, time.December, 1, 0, 0, 0, 0, time.UTC)},
	} {
		if got := timex.AddMonths(time.Time{}, tt.months); got != tt.want {
			t.Errorf("AddMonths(zero, %d) = %v; want %v", tt.months, got, tt.want)
		}
	}
}

func TestAddMonthsLocations(t *testing.T) {
	for _, tt := range []struct {
		name   string
		zone   string
		from   string
		months int
		want   string
	}{
		{"Shanghai", "Asia/Shanghai", "2026-01-31", 1, "2026-02-28"},
		{"spring DST", "America/New_York", "2026-02-28", 1, "2026-03-28"},
		{"autumn DST", "America/New_York", "2026-10-31", 1, "2026-11-30"},
		{"midnight transition at month end", "Africa/Cairo", "2004-03-31", 1, "2004-04-30"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			zone, err := time.LoadLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}

			const layout = "2006-01-02 15:04:05.999999999"
			start, err := time.ParseInLocation(layout, tt.from+" 12:34:56.123456789", zone)
			if err != nil {
				t.Fatal(err)
			}

			want, err := time.ParseInLocation(layout, tt.want+" 12:34:56.123456789", zone)
			if err != nil {
				t.Fatal(err)
			}

			got := timex.AddMonths(start, tt.months)
			if !got.Equal(want) || got.Location() != zone {
				t.Fatalf("AddMonths(%v, %d) = %v; want %v in %v", start, tt.months, got, want, zone)
			}
		})
	}
}

func TestAddMonthsNonexistentLocalTime(t *testing.T) {
	for _, tt := range []struct {
		zone  string
		year  int
		month time.Month
		day   int
		hour  int
	}{
		{"America/New_York", 2026, time.February, 8, 2},
		{"Pacific/Apia", 2011, time.November, 30, 12},
	} {
		t.Run(tt.zone, func(t *testing.T) {
			zone, err := time.LoadLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}

			start := time.Date(tt.year, tt.month, tt.day, tt.hour, 30, 0, 123456789, zone)
			// Missing local times follow time.Date's transition resolution.
			want := time.Date(tt.year, tt.month+1, tt.day, tt.hour, 30, 0, 123456789, zone)
			got := timex.AddMonths(start, 1)
			if !got.Equal(want) || got.Location() != zone {
				t.Fatalf("AddMonths(%v, 1) = %v; want time.Date result %v", start, got, want)
			}
		})
	}
}

func ExampleAddMonths() {
	anchor := time.Date(2026, time.January, 31, 12, 0, 0, 0, time.UTC)
	next := timex.AddMonths(anchor, 1)

	fmt.Println(next.Format(time.DateOnly))
	fmt.Println(timex.AddMonths(next, 1).Format(time.DateOnly))
	fmt.Println(timex.AddMonths(anchor, 2).Format(time.DateOnly))

	// Output:
	// 2026-02-28
	// 2026-03-28
	// 2026-03-31
}
