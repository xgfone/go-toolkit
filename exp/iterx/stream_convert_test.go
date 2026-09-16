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
	"strconv"
	"testing"

	"github.com/xgfone/go-toolkit/exp/iterx"
)

func ExampleStream_To2_users() {
	type User struct {
		ID   int
		Name string
	}
	users := []User{{1, "Alice"}, {2, "Bob"}}
	stream := iterx.FromSeq(slices.Values(users)).
		To2(func(u User) (int, User) { return u.ID, u }).
		Filter(func(id int, _ User) bool { return id > 1 })

	fmt.Println(maps.Collect(stream.Seq2()))
	// Output: map[2:{2 Bob}]
}

func ExampleStream_To2() {
	stream := iterx.FromSeq(slices.Values([]string{"go", "rust"})).
		To2(func(s string) (string, int) { return s, len(s) })

	fmt.Println(maps.Collect(stream.Seq2()))
	// Output: map[go:2 rust:4]
}

func TestStreamTo2(t *testing.T) {
	s := new(visits)
	calls := 0
	stream := iterx.FromSeq(tracked(s)).To2(func(v int) ([]int, string) {
		calls++
		return []int{v}, strconv.Itoa(v)
	})
	if *s != (visits{}) || calls != 0 {
		t.Fatalf("eager evaluation: visits = %+v, callbacks = %d", s, calls)
	}

	for k, v := range stream.Seq2() {
		if !slices.Equal(k, []int{1}) || v != "1" {
			t.Fatalf("first pair = (%v, %s)", k, v)
		}
		break
	}
	if *s != (visits{started: 1, read: 1, closed: 1}) || calls != 1 {
		t.Fatalf("early stop: visits = %+v, callbacks = %d", s, calls)
	}

	for range 2 {
		*s = visits{}
		calls = 0
		var got []string
		for k, v := range stream.Seq2() {
			got = append(got, fmt.Sprintf("%v=%s", k, v))
		}
		if !slices.Equal(got, []string{"[1]=1", "[2]=2", "[3]=3", "[4]=4"}) {
			t.Fatalf("pairs = %v", got)
		}
		if *s != (visits{started: 1, read: 4, closed: 1}) || calls != 4 {
			t.Fatalf("repeated iteration: visits = %+v, callbacks = %d", s, calls)
		}
	}

	*s = visits{}
	calls = 0
	got := slices.Collect(stream.Values().Take(1).Seq())
	if !slices.Equal(got, []string{"1"}) {
		t.Fatalf("projected values = %v", got)
	}
	if *s != (visits{started: 1, read: 1, closed: 1}) || calls != 1 {
		t.Fatalf("projection did not stop upstream: visits = %+v, callbacks = %d", s, calls)
	}
}

func TestStreamTo2Empty(t *testing.T) {
	stream := iterx.FromSeq(slices.Values([]int(nil))).To2(func(int) (int, int) {
		t.Fatal("callback called on empty input")
		return 0, 0
	})
	for range stream.Seq2() {
		t.Fatal("empty input yielded a pair")
	}
}

func ExampleStream2_To() {
	stream := iterx.FromSeq2(slices.All([]string{"go", "rust"})).
		Map(func(i int, s string) (int, string) { return i + 1, s }).
		To(func(i int, s string) string { return strconv.Itoa(i) + ": " + s })

	fmt.Println(slices.Collect(stream.Seq()))
	// Output: [1: go 2: rust]
}

func TestStream2To(t *testing.T) {
	s := new(visits)
	calls := 0
	stream := iterx.FromSeq2(pairs(tracked(s))).
		Map(func(k, v int) ([]int, string) { return []int{k}, strconv.Itoa(v) }).
		To(func(k []int, v string) string {
			calls++
			return fmt.Sprintf("%v=%s", k, v)
		})
	if *s != (visits{}) || calls != 0 {
		t.Fatalf("eager evaluation: visits = %+v, callbacks = %d", s, calls)
	}

	got := slices.Collect(stream.Take(1).Seq())
	if !slices.Equal(got, []string{"[10]=1"}) {
		t.Fatalf("first value = %v", got)
	}
	if *s != (visits{started: 1, read: 1, closed: 1}) || calls != 1 {
		t.Fatalf("early stop: visits = %+v, callbacks = %d", s, calls)
	}

	for range 2 {
		*s = visits{}
		calls = 0
		got := slices.Collect(stream.Seq())
		if !slices.Equal(got, []string{"[10]=1", "[20]=2", "[30]=3", "[40]=4"}) {
			t.Fatalf("values = %v", got)
		}
		if *s != (visits{started: 1, read: 4, closed: 1}) || calls != 4 {
			t.Fatalf("repeated iteration: visits = %+v, callbacks = %d", s, calls)
		}
	}
}

func TestStream2ToEmpty(t *testing.T) {
	stream := iterx.FromSeq2(slices.All([]int(nil))).To(func(int, int) string {
		t.Fatal("callback called on empty input")
		return ""
	})
	for range stream.Seq() {
		t.Fatal("empty input yielded a value")
	}
}
