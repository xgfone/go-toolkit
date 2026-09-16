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
	"iter"
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/xgfone/go-toolkit/exp/iterx"
	baseiterx "github.com/xgfone/go-toolkit/iterx"
)

func ExampleStream() {
	stream := iterx.FromSeq(slices.Values([]string{"1", "bad", "2", "3", "4"})).
		FilterMap(func(s string) (int, bool) {
			n, err := strconv.Atoi(s)
			return n, err == nil
		}).
		Filter(func(n int) bool { return n%2 == 0 }).
		Map(strconv.Itoa).
		Take(2)
	fmt.Println(slices.Collect(stream.Seq()))
	// Output: [2 4]
}

func ExampleStream2() {
	stream := iterx.FromSeq2(slices.All([]string{"go", "rust", "js"})).
		Filter(func(_ int, s string) bool { return len(s) > 2 }).
		Map2(func(i int, s string) (string, int) { return s, i })
	fmt.Println(maps.Collect(stream.Seq()))
	// Output: map[rust:1]
}

func ExampleStream2_Values() {
	stream := iterx.FromSeq2(slices.All([]string{"one", "two", "three"})).
		Filter(func(i int, _ string) bool { return i > 0 }).
		Values().
		Map(func(s string) int { return len(s) }).
		Take(1)
	fmt.Println(baseiterx.Sum(stream.Seq()))
	// Output: 3
}

func TestStreamChain(t *testing.T) {
	s := new(visits)
	stream := iterx.FromSeq(tracked(s)).
		Skip(1).
		Filter(func(v int) bool { return v%2 == 0 }).
		Map(strconv.Itoa).
		FilterMap(func(v string) (int, bool) {
			n, err := strconv.Atoi(v)
			return n * 10, err == nil
		}).
		Take(2)
	if *s != (visits{}) {
		t.Fatalf("chain started its source: %+v", s)
	}
	if got := slices.Collect(stream.Take(0).Seq()); len(got) != 0 || *s != (visits{}) {
		t.Fatalf("Take(0) = %v, visits = %+v", got, s)
	}
	for v := range stream.Seq() {
		if v != 20 {
			t.Fatalf("first value = %d, want 20", v)
		}
		break
	}
	if *s != (visits{started: 1, read: 2, closed: 1}) {
		t.Fatalf("early stop: %+v", s)
	}
	for range 2 {
		*s = visits{}
		if got := slices.Collect(stream.Seq()); !slices.Equal(got, []int{20, 40}) {
			t.Fatalf("values = %v", got)
		}
		if *s != (visits{started: 1, read: 4, closed: 1}) {
			t.Fatalf("repeated iteration: %+v", s)
		}
	}
}

func TestStream2Chain(t *testing.T) {
	s := new(visits)
	stream := iterx.FromSeq2(pairs(tracked(s))).
		Filter(func(k, v int) bool { return k == v*10 && v > 1 }).
		Map2(func(k, v int) ([]int, string) { return []int{k}, strconv.Itoa(v) }).
		FilterMap2(func(k []int, v string) ([]int, int, bool) {
			n, err := strconv.Atoi(v)
			return k, n, err == nil && n%2 == 0
		})
	if *s != (visits{}) {
		t.Fatalf("chain started its source: %+v", s)
	}

	// Two-value iteration accepts non-comparable first values and stops upstream.
	for k, v := range stream.Seq() {
		if !slices.Equal(k, []int{20}) || v != 2 {
			t.Fatalf("first pair = (%v, %d)", k, v)
		}
		break
	}
	if *s != (visits{started: 1, read: 2, closed: 1}) {
		t.Fatalf("early stop: %+v", s)
	}

	*s = visits{}
	keys := stream.Keys().Map(func(k []int) int { return k[0] }).Take(1)
	if got := slices.Collect(keys.Seq()); !slices.Equal(got, []int{20}) {
		t.Fatalf("keys = %v", got)
	}
	if *s != (visits{started: 1, read: 2, closed: 1}) {
		t.Fatalf("projection did not stop upstream: %+v", s)
	}

	for range 2 {
		*s = visits{}
		if got := slices.Collect(stream.Values().Seq()); !slices.Equal(got, []int{2, 4}) {
			t.Fatalf("values = %v", got)
		}
		if *s != (visits{started: 1, read: 4, closed: 1}) {
			t.Fatalf("repeated iteration: %+v", s)
		}
	}
}

func TestStreamsPreserveSingleUseSource(t *testing.T) {
	for _, pair := range []bool{false, true} {
		t.Run(fmt.Sprintf("pairs=%t", pair), func(t *testing.T) {
			next, stop := iter.Pull(slices.Values([]int{1, 2, 3}))
			defer stop()
			seq := func(yield func(int) bool) {
				for {
					v, ok := next()
					if !ok || !yield(v) {
						return
					}
				}
			}
			stream := iterx.FromSeq(seq)
			if pair {
				stream = iterx.FromSeq2(pairs(seq)).Values()
			}
			first := stream.Take(1)
			for _, want := range []int{1, 2, 3} {
				if got := slices.Collect(first.Seq()); !slices.Equal(got, []int{want}) {
					t.Fatalf("single-use input = %v, want [%d]", got, want)
				}
			}
			if got := slices.Collect(first.Seq()); len(got) != 0 {
				t.Fatalf("exhausted input = %v", got)
			}
		})
	}
}

type visits struct{ started, read, closed int }

func tracked(s *visits) iter.Seq[int] {
	return func(yield func(int) bool) {
		s.started++
		defer func() { s.closed++ }()
		for i := 1; i <= 4; i++ {
			s.read++
			if !yield(i) {
				return
			}
		}
	}
}

func pairs(seq iter.Seq[int]) iter.Seq2[int, int] {
	return func(yield func(int, int) bool) {
		for v := range seq {
			if !yield(v*10, v) {
				return
			}
		}
	}
}

func ExampleStream_To() {
	// Aliases can be mixed with the primary method names.
	stream := iterx.FromSeq(slices.Values([]string{"skip", "bad", "2", "3"})).
		Drop(1).
		FilterTo(func(s string) (int, bool) {
			n, err := strconv.Atoi(s)
			return n, err == nil
		}).
		To(strconv.Itoa).
		Take(1)
	fmt.Println(slices.Collect(stream.Seq()))
	// Output: [2]
}

func ExampleStream2_To2() {
	stream := iterx.FromSeq2(slices.All([]string{"bad", "2", "3"})).
		FilterTo2(func(i int, s string) (int, int, bool) {
			n, err := strconv.Atoi(s)
			return i, n, err == nil
		}).
		To2(func(i, n int) (string, int) { return strconv.Itoa(i), n * 2 })
	fmt.Println(maps.Collect(stream.Seq()))
	// Output: map[1:4 2:6]
}
