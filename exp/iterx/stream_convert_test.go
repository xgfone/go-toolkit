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

func ExampleStream_Map2() {
	type User struct {
		ID   int
		Name string
	}
	users := []User{{1, "Alice"}, {2, "Bob"}}
	stream := iterx.FromSeq(slices.Values(users)).
		Map2(func(u User) (int, User) { return u.ID, u }).
		Filter(func(id int, _ User) bool { return id > 1 })

	fmt.Println(maps.Collect(stream.Seq()))
	// Output: map[2:{2 Bob}]
}

func ExampleStream_To2() {
	stream := iterx.FromSeq(slices.Values([]string{"go", "rust"})).
		To2(func(s string) (string, int) { return s, len(s) })

	fmt.Println(maps.Collect(stream.Seq()))
	// Output: map[go:2 rust:4]
}

func TestStreamMap2(t *testing.T) {
	s := new(visits)
	calls := 0
	stream := iterx.FromSeq(tracked(s)).Map2(func(v int) ([]int, string) {
		calls++
		return []int{v}, strconv.Itoa(v)
	})
	if *s != (visits{}) || calls != 0 {
		t.Fatalf("eager evaluation: visits = %+v, callbacks = %d", s, calls)
	}

	for k, v := range stream.Seq() {
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
		for k, v := range stream.Seq() {
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

func TestStreamMap2Empty(t *testing.T) {
	stream := iterx.FromSeq(slices.Values([]int(nil))).Map2(func(int) (int, int) {
		t.Fatal("callback called on empty input")
		return 0, 0
	})
	for range stream.Seq() {
		t.Fatal("empty input yielded a pair")
	}
}
