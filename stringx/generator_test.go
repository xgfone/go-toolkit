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

package stringx

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xgfone/go-toolkit/timex"
)

func mustPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	f()
}

func repeatGenerator(minlen int, char string) FuncGenerator {
	return NewGenerator(minlen, func(dst []byte, length int) []byte {
		return append(dst, strings.Repeat(char, length)...)
	})
}

func TestGeneratorLengthContract(t *testing.T) {
	base := repeatGenerator(2, "x")
	affix := NewAffixGenerator(base).WithPrefix("前").WithSuffix("后")
	nested := NewAffixGenerator(affix).WithPrefix("[").WithSuffix("]")
	for _, tt := range []struct {
		name           string
		generator      Generator
		minlen         int
		prefix, suffix string
	}{
		{"callback", base, 2, "", ""},
		{"affix", affix, 8, "前", "后"},
		{"nested", nested, 10, "[前", "后]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.generator.MinLen(); got != tt.minlen {
				t.Fatalf("MinLen() = %d, want %d", got, tt.minlen)
			}

			for _, n := range []int{-1, 0, tt.minlen, tt.minlen + 5} {
				wantlen := n
				if wantlen <= 0 {
					wantlen = tt.minlen
				}

				want := tt.prefix + strings.Repeat("x", wantlen-len(tt.prefix)-len(tt.suffix)) + tt.suffix
				if got := tt.generator.Generate(n); got != want {
					t.Errorf("Generate(%d) = %q, want %q", n, got, want)
				}

				// Exercise both reallocation and reuse with a non-empty buffer.
				for _, capacity := range []int{5, 100} {
					dst := append(make([]byte, 0, capacity), "keep:"...)
					if got := string(tt.generator.Append(dst, n)); got != "keep:"+want {
						t.Errorf("Append(%d) = %q, want %q", n, got, "keep:"+want)
					}
				}
			}

			mustPanic(t, func() { tt.generator.Generate(tt.minlen - 1) })
			mustPanic(t, func() { tt.generator.Append(nil, tt.minlen-1) })
		})
	}
}

func TestAffixGeneratorConfiguration(t *testing.T) {
	base := repeatGenerator(2, "x")
	original := NewAffixGenerator(base)
	changed := original.WithPrefix("pre").WithSuffix("post").WithGenerator(repeatGenerator(3, "y"))
	if original.Prefix() != "" || original.Suffix() != "" || original.Generator().Generate(0) != "xx" {
		t.Fatal("With methods modified the original")
	}
	if changed.Prefix() != "pre" || changed.Suffix() != "post" || changed.Generator().MinLen() != 3 {
		t.Fatal("configuration getters do not reflect the new configuration")
	}
	if got := changed.Generate(12); got != "preyyyyypost" {
		t.Fatalf("Generate(12) = %q", got)
	}
}

func TestGeneratorInvalidConfiguration(t *testing.T) {
	mustPanic(t, func() { NewGenerator(-1, func(dst []byte, _ int) []byte { return dst }) })
	mustPanic(t, func() { NewGenerator(1, nil) })
	mustPanic(t, func() { FuncGenerator{}.Generate(1) })
	mustPanic(t, func() { AffixGenerator{}.Generate(1) })

	original := DefaultGenerator()
	t.Cleanup(func() { SetDefaultGenerator(original) })

	var nilPointer *customGenerator
	var nilFunc nilFuncGenerator
	for _, g := range []Generator{nil, nilPointer, nilFunc} {
		mustPanic(t, func() { NewAffixGenerator(g) })
		mustPanic(t, func() { NewAffixGenerator(original).WithGenerator(g) })
		mustPanic(t, func() { SetDefaultGenerator(g) })
	}

	// Failed setters must leave the old default usable.
	if got := DefaultGenerator().Generate(0); len(got) != original.MinLen() {
		t.Fatalf("failed setter changed default: %q", got)
	}

	invalid := invalidGenerator{minlen: -1}
	mustPanic(t, func() { SetDefaultGenerator(invalid) })
	mustPanic(t, func() { NewAffixGenerator(invalid).Generate(0) })

	const maxInt = int(^uint(0) >> 1)
	large := NewAffixGenerator(invalidGenerator{minlen: maxInt})
	mustPanic(t, func() { large.WithPrefix("x").MinLen() })
	mustPanic(t, func() { large.WithSuffix("x").MinLen() })
}

func TestGeneratorRejectsIncorrectOutputLength(t *testing.T) {
	for _, size := range []int{0, 3, 5} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			g := NewGenerator(2, func(dst []byte, _ int) []byte {
				return append(dst, strings.Repeat("x", size)...)
			})
			mustPanic(t, func() { g.Generate(4) })
			mustPanic(t, func() { g.Append([]byte("keep:"), 4) })
		})
	}

	g := NewAffixGenerator(invalidGenerator{minlen: 2}).WithPrefix("[").WithSuffix("]")
	mustPanic(t, func() { g.Generate(4) })
	mustPanic(t, func() { g.Append([]byte("keep:"), 4) })
}

func TestGeneratorZeroMinimum(t *testing.T) {
	g := repeatGenerator(0, "x")
	if got := g.Generate(0); got != "" {
		t.Fatalf("Generate(0) = %q", got)
	}
	if got := string(g.Append([]byte("keep"), -1)); got != "keep" {
		t.Fatalf("Append(-1) = %q", got)
	}
	if got := NewAffixGenerator(g).WithPrefix("[").WithSuffix("]").Generate(0); got != "[]" {
		t.Fatalf("affix Generate(0) = %q", got)
	}
}

func TestGeneratedStringDoesNotAliasCallbackBuffer(t *testing.T) {
	for _, affix := range []bool{false, true} {
		t.Run(strconv.FormatBool(affix), func(t *testing.T) {
			var retained []byte
			var g Generator = NewGenerator(1, func(dst []byte, length int) []byte {
				retained = append(dst, strings.Repeat("x", length)...)
				return retained
			})

			want := "xxxx"
			if affix {
				g = NewAffixGenerator(g).WithPrefix("[").WithSuffix("]")
				want = "[xx]"
			}

			got := g.Generate(4)
			for i := range retained {
				retained[i] = 'y'
			}
			if got != want {
				t.Fatalf("callback buffer changed generated string: got %q, want %q", got, want)
			}
		})
	}
}

func TestPredefinedGenerators(t *testing.T) {
	now := time.Date(2026, 9, 6, 1, 2, 3, 5e6, time.UTC)
	timex.SetNowFunc(func() time.Time { return now })
	t.Cleanup(func() {
		timex.SetNowFunc(func() time.Time { return time.Now().In(timex.Location()) })
	})
	for _, tt := range []struct {
		name      string
		generator Generator
		minlen    int
		prefix    string
	}{
		{"datetime_millis", DateTimeMilliRandGenerator(), 18, "20260906010203005"},
		{"unix_millis", UnixTimeMilliRandGenerator(), 14, strconv.FormatInt(now.UnixMilli(), 10)},
		{"datetime", DateTimeRandGenerator(), 15, "20260906010203"},
		{"unix", UnixTimeRandGenerator(), 11, strconv.FormatInt(now.Unix(), 10)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.generator.MinLen(); got != tt.minlen {
				t.Fatalf("MinLen() = %d, want %d", got, tt.minlen)
			}

			for _, length := range []int{-1, 0, tt.minlen, 20, 50} {
				wantlen := length
				if wantlen <= 0 {
					wantlen = tt.minlen
				}

				for _, got := range []string{
					tt.generator.Generate(length),
					strings.TrimPrefix(string(tt.generator.Append([]byte("keep:"), length)), "keep:"),
				} {
					if len(got) != wantlen || !strings.HasPrefix(got, tt.prefix) || !IsASCIIDigits(got) {
						t.Errorf("length %d: unexpected generated string %q", length, got)
					}
				}
			}
			mustPanic(t, func() { tt.generator.Generate(tt.minlen - 1) })
			mustPanic(t, func() { tt.generator.Append(nil, tt.minlen-1) })
		})
	}
}

func TestAppendTimeRand(t *testing.T) {
	const prefix = "keep:time:"
	for digits := 1; digits <= 9; digits++ {
		got := string(appendTimeRand([]byte("keep:"), len("time:")+digits, func(dst []byte, _ time.Time) []byte {
			return append(dst, "time:"...)
		}))
		if len(got) != len(prefix)+digits || !strings.HasPrefix(got, prefix) || !IsASCIIDigits(got[len(prefix):]) {
			t.Errorf("%d random digits: unexpected generated string %q", digits, got)
		}
	}
}

func TestDefaultGenerator(t *testing.T) {
	original := DefaultGenerator()
	t.Cleanup(func() { SetDefaultGenerator(original) })

	a := repeatGenerator(2, "x")
	b := &customGenerator{char: "y"}
	SetDefaultGenerator(a)

	snapshot := DefaultGenerator()
	SetDefaultGenerator(b)
	if got := snapshot.Generate(0); got != "xx" {
		t.Fatalf("replacement changed snapshot: %q", got)
	}
	if got := DefaultGenerator().Generate(0); got != "yy" {
		t.Fatalf("custom default Generate(0) = %q", got)
	}

	// Different concrete implementations must be replaceable concurrently.
	generators := []Generator{a, b, NewAffixGenerator(a).WithPrefix("[").WithSuffix("]")}
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Go(func() {
			for i := 0; i < 100; i++ {
				SetDefaultGenerator(generators[i%len(generators)])
				g := DefaultGenerator()
				if got := g.Generate(0); got != "xx" && got != "yy" && got != "[xx]" {
					t.Errorf("unexpected default output %q", got)
				}
			}
		})
	}
	wg.Wait()
}

// customGenerator deliberately implements [Generator] without the function adapter.
type customGenerator struct{ char string }

func (g *customGenerator) MinLen() int { return 2 }
func (g *customGenerator) Generate(n int) string {
	if n <= 0 {
		n = g.MinLen()
	}
	if n < g.MinLen() {
		panic("length below minimum")
	}
	return strings.Repeat(g.char, n)
}
func (g *customGenerator) Append(dst []byte, n int) []byte {
	return append(dst, g.Generate(n)...)
}

type nilFuncGenerator func()

func (nilFuncGenerator) MinLen() int                     { return 0 }
func (nilFuncGenerator) Generate(int) string             { return "" }
func (nilFuncGenerator) Append(dst []byte, _ int) []byte { return dst }

type invalidGenerator struct{ minlen int }

func (g invalidGenerator) MinLen() int                   { return g.minlen }
func (invalidGenerator) Generate(int) string             { return "" }
func (invalidGenerator) Append(dst []byte, _ int) []byte { return dst }

func ExampleNewAffixGenerator() {
	base := NewGenerator(2, func(dst []byte, length int) []byte {
		return append(dst, strings.Repeat("x", length)...)
	})

	g := NewAffixGenerator(base).WithPrefix("order_").WithSuffix("!")
	fmt.Println(g.MinLen())
	fmt.Println(g.Generate(10))
	fmt.Println(string(g.Append([]byte("id="), 0)))
	// Output:
	// 9
	// order_xxx!
	// id=order_xx!
}
