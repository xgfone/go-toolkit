// Copyright 2025~2026 xgfone
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
	"reflect"
	"slices"
	"testing"

	"github.com/xgfone/go-toolkit/iterx"
)

type visits struct{ started, read, closed, callbacks int }

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

func pairSums(seq iter.Seq2[int, int]) iter.Seq[int] {
	return func(yield func(int) bool) {
		for k, v := range seq {
			if !yield(k + v) {
				return
			}
		}
	}
}

func TestAdapters(t *testing.T) {
	tests := []struct {
		name  string
		build func(iter.Seq[int], *visits) iter.Seq[int]
		want  []int

		firstRead, firstCallbacks, fullRead, fullCallbacks int
	}{
		{
			"To", func(seq iter.Seq[int], s *visits) iter.Seq[int] {
				return iterx.To(seq, func(v int) int {
					s.callbacks++
					return v * 2
				})
			},
			[]int{2, 4, 6, 8}, 1, 1, 4, 4,
		},
		{
			"To2", func(seq iter.Seq[int], s *visits) iter.Seq[int] {
				return pairSums(iterx.To2(pairs(seq), func(k, v int) (int, int) {
					s.callbacks++
					return k + 1, v * 2
				}))
			},
			[]int{13, 25, 37, 49}, 1, 1, 4, 4,
		},
		{
			"Keys", func(seq iter.Seq[int], _ *visits) iter.Seq[int] {
				return iterx.Keys(pairs(seq))
			},
			[]int{10, 20, 30, 40}, 1, 0, 4, 0,
		},
		{
			"Values", func(seq iter.Seq[int], _ *visits) iter.Seq[int] {
				return iterx.Values(pairs(seq))
			},
			[]int{1, 2, 3, 4}, 1, 0, 4, 0,
		},
		{
			"Filter", func(seq iter.Seq[int], s *visits) iter.Seq[int] {
				return iterx.Filter(seq, func(v int) bool {
					s.callbacks++
					return v%2 == 0
				})
			},
			[]int{2, 4}, 2, 2, 4, 4,
		},
		{
			"Filter2", func(seq iter.Seq[int], s *visits) iter.Seq[int] {
				return pairSums(iterx.Filter2(pairs(seq), func(k, v int) bool {
					s.callbacks++
					return k == 10*v && v%2 == 0
				}))
			},
			[]int{22, 44}, 2, 2, 4, 4,
		},
		{
			"FilterTo", func(seq iter.Seq[int], s *visits) iter.Seq[int] {
				return iterx.FilterTo(seq, func(v int) (int, bool) {
					s.callbacks++
					return v * 10, v%2 == 0
				})
			},
			[]int{20, 40}, 2, 2, 4, 4,
		},
		{
			"FilterTo2", func(seq iter.Seq[int], s *visits) iter.Seq[int] {
				return pairSums(iterx.FilterTo2(pairs(seq), func(k, v int) (int, int, bool) {
					s.callbacks++
					return k + 1, v * 2, v%2 == 0
				}))
			},
			[]int{25, 49}, 2, 2, 4, 4,
		},
		{
			"Take", func(seq iter.Seq[int], _ *visits) iter.Seq[int] {
				return iterx.Take(seq, 2)
			},
			[]int{1, 2}, 1, 0, 2, 0,
		},
		{
			"Drop", func(seq iter.Seq[int], _ *visits) iter.Seq[int] {
				return iterx.Drop(seq, 2)
			},
			[]int{3, 4}, 3, 0, 4, 0,
		},
		{
			"Concat", func(seq iter.Seq[int], _ *visits) iter.Seq[int] {
				return iterx.Concat(seq, slices.Values([]int{5}))
			},
			[]int{1, 2, 3, 4, 5}, 1, 0, 4, 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := new(visits)
			seq := tt.build(tracked(s), s)
			if *s != (visits{}) {
				t.Fatalf("eager evaluation: %+v", s)
			}

			for v := range seq {
				if v != tt.want[0] {
					t.Fatalf("first value = %v", v)
				}
				break
			}

			if *s != (visits{1, tt.firstRead, 1, tt.firstCallbacks}) {
				t.Fatalf("early stop: %+v", s)
			}

			for range 2 {
				*s = visits{}
				if got := slices.Collect(seq); !slices.Equal(got, tt.want) {
					t.Fatalf("values = %v, want %v", got, tt.want)
				}
				if *s != (visits{1, tt.fullRead, 1, tt.fullCallbacks}) {
					t.Fatalf("full consumption: %+v", s)
				}
			}

			*s = visits{}
			defer func() {
				if got := recover(); got != "consumer panic" {
					t.Errorf("panic = %v", got)
				}
				if s.closed != 1 || s.read != tt.firstRead {
					t.Errorf("panic cleanup: %+v", s)
				}
			}()
			for range seq {
				panic("consumer panic")
			}
		})

		t.Run(tt.name+"/empty", func(t *testing.T) {
			s := new(visits)
			got := slices.Collect(tt.build(slices.Values([]int(nil)), s))
			want := []int(nil)
			if tt.name == "Concat" {
				want = []int{5}
			}
			if !slices.Equal(got, want) || s.callbacks != 0 {
				t.Fatalf("empty values = %v, visits = %+v", got, s)
			}
		})
	}
}

func TestTerminalOperations(t *testing.T) {
	empty := slices.Values([]int(nil))
	unused := func(int) bool { t.Fatal("callback on empty input"); return false }
	if iterx.Count(empty) != 0 || iterx.CountFunc(empty, unused) != 0 || iterx.Sum(empty) != 0 {
		t.Fatal("empty totals")
	}

	gotint := iterx.SumFunc(empty, func(int) int {
		t.Fatal("sum callback on empty input")
		return 1
	})
	if gotint != 0 {
		t.Fatal(gotint)
	}

	if v, ok := iterx.Find(empty, unused); v != 0 || ok {
		t.Fatalf("empty Find = %v, %v", v, ok)
	}

	gotstr := iterx.Reduce(empty, "initial", func(string, int) string {
		t.Fatal("reduce on empty input")
		return ""
	})
	if gotstr != "initial" {
		t.Fatal(gotstr)
	}

	type Amount int64
	if got := iterx.Sum(slices.Values([]Amount{1, 2, 3})); got != Amount(6) {
		t.Fatal(got)
	}
	if got := iterx.Sum(slices.Values([]float64{1.25, 2.5})); got != 3.75 {
		t.Fatal(got)
	}

	s := new(visits)
	total := iterx.SumFunc(tracked(s), func(v int) Amount {
		s.callbacks++
		return Amount(v * 2)
	})
	if total != 20 || *s != (visits{1, 4, 1, 4}) {
		t.Fatalf("sum = %v, visits = %+v", total, s)
	}

	*s = visits{}
	if count := iterx.Count(tracked(s)); count != 4 || *s != (visits{1, 4, 1, 0}) {
		t.Fatalf("count = %v, visits = %+v", count, s)
	}

	*s = visits{}
	count := iterx.CountFunc(tracked(s), func(v int) bool {
		s.callbacks++
		return v%2 == 0
	})
	if count != 2 || *s != (visits{1, 4, 1, 4}) {
		t.Fatalf("count = %v, visits = %+v", count, s)
	}

	*s = visits{}
	result := iterx.Reduce(tracked(s), "start", func(r string, v int) string {
		s.callbacks++
		return fmt.Sprintf("%s/%d", r, v)
	})
	if result != "start/1/2/3/4" || *s != (visits{1, 4, 1, 4}) {
		t.Fatalf("reduce = %v, visits = %+v", result, s)
	}
}

func TestFind(t *testing.T) {
	for _, target := range []int{2, 5} {
		s := new(visits)
		got, ok := iterx.Find(tracked(s), func(v int) bool {
			s.callbacks++
			return v == target
		})

		want, read := target, target
		if target == 5 {
			want, read = 0, 4
		}
		if got != want || ok != (target == 2) || *s != (visits{1, read, 1, read}) {
			t.Fatalf("Find(%d) = %d, %v; %+v", target, got, ok, s)
		}
	}

	got, ok := iterx.Find(slices.Values([]int{0}), func(int) bool { return true })
	if got != 0 || !ok {
		t.Fatal("zero value match lost")
	}
}

func TestTakeDropLimits(t *testing.T) {
	for _, n := range []int{0, 1, 4, 5} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s := new(visits)
			got := slices.Collect(iterx.Take(tracked(s), n))
			want := []int{1, 2, 3, 4}[:min(n, 4)]
			starts := 1
			if n == 0 {
				starts = 0
			}
			if !slices.Equal(got, want) || *s != (visits{starts, min(n, 4), starts, 0}) {
				t.Fatalf("Take = %v; %+v", got, s)
			}

			got = slices.Collect(iterx.Drop(slices.Values([]int{1, 2, 3, 4}), n))
			if !slices.Equal(got, []int{1, 2, 3, 4}[min(n, 4):]) {
				t.Fatalf("Drop = %v", got)
			}
		})
	}

	for name, fn := range map[string]func(iter.Seq[int], int) iter.Seq[int]{
		"Take": iterx.Take[int],
		"Drop": iterx.Drop[int],
	} {
		t.Run(name+"/negative", func(t *testing.T) {
			s := new(visits)
			defer func() {
				if recover() == nil {
					t.Error("negative count must panic at construction")
				}
				if *s != (visits{}) {
					t.Error("negative count started source")
				}
			}()
			fn(tracked(s), -1)
		})
	}
}

func TestConcatStopsBeforeNextInput(t *testing.T) {
	first, second := new(visits), new(visits)
	got := slices.Collect(iterx.Take(iterx.Concat(tracked(first), tracked(second)), 4))
	if !slices.Equal(got, []int{1, 2, 3, 4}) {
		t.Fatal(got)
	}
	if *first != (visits{1, 4, 1, 0}) || *second != (visits{}) {
		t.Fatalf("first = %+v, second = %+v", first, second)
	}
	if got := slices.Collect(iterx.Concat[int]()); len(got) != 0 {
		t.Fatal(got)
	}
}

func TestSingleUse(t *testing.T) {
	position := 0
	source := iter.Seq[int](func(yield func(int) bool) {
		for position < 5 {
			position++
			if !yield(position) {
				return
			}
		}
	})

	seq := iterx.Take(iterx.To(source, func(v int) int { return v * 10 }), 2)
	for i, want := range [][]int{{10, 20}, {30, 40}, {50}, nil} {
		if got := slices.Collect(seq); !slices.Equal(got, want) {
			t.Fatalf("iteration %d = %v, want %v", i, got, want)
		}
	}
}

func TestNonComparablePairs(t *testing.T) {
	seq := iter.Seq2[[]int, int](func(yield func([]int, int) bool) {
		for _, v := range []int{1, 2} {
			if !yield([]int{v}, v) {
				return
			}
		}
	})

	keys := slices.Collect(iterx.Keys(seq))
	if !reflect.DeepEqual(keys, [][]int{{1}, {2}}) {
		t.Fatal(keys)
	}

	converted := iterx.To2(seq, func(k []int, v int) ([]int, []int) {
		return k, []int{v * 2}
	})
	selected := iterx.FilterTo2(converted, func(k, v []int) ([]int, int, bool) {
		return k, v[0], k[0] == 2
	})
	got := slices.Collect(iterx.Values(selected))
	if !slices.Equal(got, []int{4}) {
		t.Fatal(got)
	}
}

func TestCallbackPanicCleansUp(t *testing.T) {
	s := new(visits)
	defer func() {
		if recover() != "callback panic" || s.closed != 1 {
			t.Fatalf("cleanup = %+v", s)
		}
	}()

	panicf := func(int) int { panic("callback panic") }
	for range iterx.To(tracked(s), panicf) {
	}
}
