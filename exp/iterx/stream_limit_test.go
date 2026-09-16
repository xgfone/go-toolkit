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

package iterx_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/xgfone/go-toolkit/exp/iterx"
)

func ExampleStream2_Take() {
	stream := iterx.FromSeq2(slices.All([]string{"one", "two", "three", "four"})).
		Skip(1).
		Take(2)

	fmt.Println(maps.Collect(stream.Seq2()))
	// Output: map[1:two 2:three]
}

func TestStream2Limits(t *testing.T) {
	tests := []struct {
		name      string
		build     func(iterx.Stream2[int, int]) iterx.Stream2[int, int]
		want      [][2]int
		firstRead int
		fullRead  int
	}{
		{"TakeZero", func(s iterx.Stream2[int, int]) iterx.Stream2[int, int] {
			return s.Take(0)
		}, nil, 0, 0},
		{"TakeTwo", func(s iterx.Stream2[int, int]) iterx.Stream2[int, int] {
			return s.Take(2)
		}, [][2]int{{10, 1}, {20, 2}}, 1, 2},
		{"TakePastEnd", func(s iterx.Stream2[int, int]) iterx.Stream2[int, int] {
			return s.Take(6)
		}, [][2]int{{10, 1}, {20, 2}, {30, 3}, {40, 4}}, 1, 4},
		{"SkipZero", func(s iterx.Stream2[int, int]) iterx.Stream2[int, int] {
			return s.Skip(0)
		}, [][2]int{{10, 1}, {20, 2}, {30, 3}, {40, 4}}, 1, 4},
		{"SkipTwo", func(s iterx.Stream2[int, int]) iterx.Stream2[int, int] {
			return s.Skip(2)
		}, [][2]int{{30, 3}, {40, 4}}, 3, 4},
		{"SkipPastEnd", func(s iterx.Stream2[int, int]) iterx.Stream2[int, int] {
			return s.Skip(6)
		}, nil, 4, 4},
		{"Drop", func(s iterx.Stream2[int, int]) iterx.Stream2[int, int] {
			return s.Drop(3)
		}, [][2]int{{40, 4}}, 4, 4},
		{"SkipThenTake", func(s iterx.Stream2[int, int]) iterx.Stream2[int, int] {
			return s.Skip(1).Take(2)
		}, [][2]int{{20, 2}, {30, 3}}, 2, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := new(visits)
			stream := tt.build(iterx.FromSeq2(pairs(tracked(s))))
			if *s != (visits{}) {
				t.Fatalf("eager evaluation: %+v", s)
			}

			var first [][2]int
			for k, v := range stream.Seq2() {
				first = append(first, [2]int{k, v})
				break
			}
			if !slices.Equal(first, tt.want[:min(1, len(tt.want))]) {
				t.Fatalf("first pair = %v, want prefix of %v", first, tt.want)
			}

			started := min(1, tt.fullRead)
			if *s != (visits{started: started, read: tt.firstRead, closed: started}) {
				t.Fatalf("early stop: %+v", s)
			}

			for range 2 {
				*s = visits{}
				var got [][2]int
				for k, v := range stream.Seq2() {
					got = append(got, [2]int{k, v})
				}
				if !slices.Equal(got, tt.want) {
					t.Fatalf("pairs = %v, want %v", got, tt.want)
				}
				if *s != (visits{started: started, read: tt.fullRead, closed: started}) {
					t.Fatalf("repeated iteration: %+v", s)
				}
			}
		})
	}
}

func TestStream2LimitsNegative(t *testing.T) {
	for _, method := range []string{"Take", "Skip", "Drop"} {
		t.Run(method, func(t *testing.T) {
			s := new(visits)
			stream := iterx.FromSeq2(pairs(tracked(s)))
			defer func() {
				if recover() == nil {
					t.Error("negative count did not panic during construction")
				}
				if *s != (visits{}) {
					t.Errorf("input started: %+v", s)
				}
			}()

			switch method {
			case "Take":
				stream.Take(-1)
			case "Skip":
				stream.Skip(-1)
			case "Drop":
				stream.Drop(-1)
			}
		})
	}
}

func TestStream2LimitsEmpty(t *testing.T) {
	empty := iterx.FromSeq2(slices.All([]int(nil)))
	for _, stream := range []iterx.Stream2[int, int]{
		empty.Take(2),
		empty.Skip(2),
		empty.Skip(0),
	} {
		for range stream.Seq2() {
			t.Fatal("empty input yielded a pair")
		}
	}
}

func TestStream2LimitsSingleUse(t *testing.T) {
	next := 0
	stream := iterx.FromSeq2(func(yield func(int, int) bool) {
		for next < 4 {
			next++
			if !yield(next*10, next) {
				return
			}
		}
	}).Skip(1).Take(1)

	for _, want := range []int{2, 4} {
		got := slices.Collect(stream.Values().Seq())
		if !slices.Equal(got, []int{want}) || next != want {
			t.Fatalf("values = %v, read = %d, want [%d] without an extra read", got, next, want)
		}
	}
	for range stream.Seq2() {
		t.Fatal("exhausted input yielded a pair")
	}
}

func TestStream2LimitsNonComparable(t *testing.T) {
	stream := iterx.FromSeq(slices.Values([]int{1, 2, 3})).
		To2(func(v int) ([]int, []int) { return []int{v * 10}, []int{v} }).
		Skip(1).
		Take(1)

	count := 0
	for k, v := range stream.Seq2() {
		count++
		if !slices.Equal(k, []int{20}) || !slices.Equal(v, []int{2}) {
			t.Fatalf("pair = (%v, %v)", k, v)
		}
	}
	if count != 1 {
		t.Fatalf("pair count = %d, want 1", count)
	}
}
