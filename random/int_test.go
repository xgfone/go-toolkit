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

package random

import (
	"bytes"
	crand "crypto/rand"
	"math"
	"strconv"
	"sync"
	"testing"
)

func TestSeedString(t *testing.T) {
	v, err := strconv.ParseInt(SeedString(), 10, 64)
	if err != nil {
		t.Fatal(err)
	} else if v < 0 {
		t.Errorf("unexpect a negative integer %d", v)
	}
}

func TestIntN(t *testing.T) {
	for _, n := range []int{1, 2, 3, 10, math.MaxInt} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			for range 100 {
				if v := IntN(n); v < 0 || v >= n {
					t.Fatalf("IntN(%d) = %d, want a value in [0, %d)", n, v, n)
				}
			}
		})
	}
}

func TestInt64N(t *testing.T) {
	for _, n := range []int64{
		1, 2, 3, 10, 36, 255, 256, 257,
		1<<32 - 1, 1 << 32, 1<<32 + 1,
		1<<62 - 1, 1 << 62, 1<<62 + 1, math.MaxInt64,
	} {
		t.Run(strconv.FormatInt(n, 10), func(t *testing.T) {
			for range 100 {
				if v := Int64N(n); v < 0 || v >= n {
					t.Fatalf("Int64N(%d) = %d, want a value in [0, %d)", n, v, n)
				}
			}
		})
	}
}

func TestIntNInvalidBound(t *testing.T) {
	for _, n := range []int{0, -1, math.MinInt} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("IntN(%d) did not panic", n)
				}
			}()
			IntN(n)
		})
	}
}

func TestInt64NInvalidBound(t *testing.T) {
	for _, n := range []int64{0, -1, math.MinInt64} {
		t.Run(strconv.FormatInt(n, 10), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("Int64N(%d) did not panic", n)
				}
			}()
			Int64N(n)
		})
	}
}

func TestCryptoSource(t *testing.T) {
	orig := crand.Reader
	t.Cleanup(func() { crand.Reader = orig })
	crand.Reader = bytes.NewReader([]byte{0x10, 0x32, 0x54, 0x76, 0x98, 0xba, 0xdc, 0xfe})

	if got := (cryptoSource{}).Uint64(); got != 0xfedcba9876543210 {
		t.Fatalf("Uint64() = %#x, want %#x", got, uint64(0xfedcba9876543210))
	}
}

func TestInt64NOneDoesNotRead(t *testing.T) {
	orig := crand.Reader
	t.Cleanup(func() { crand.Reader = orig })
	reader := bytes.NewReader(make([]byte, 8))
	crand.Reader = reader

	if got := Int64N(1); got != 0 {
		t.Fatalf("Int64N(1) = %d, want 0", got)
	}
	if reader.Len() != 8 {
		t.Fatal("Int64N(1) consumed random bytes")
	}
}

func TestInt64NConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for _, n := range []int64{1, 2, 3, 10, 36, 1 << 32, 1<<62 + 1, math.MaxInt64} {
		wg.Go(func() {
			for range 100 {
				if got := Int64N(n); got < 0 || got >= n {
					t.Errorf("Int64N(%d) = %d, want a value in [0, %d)", n, got, n)
					return
				}
			}
		})
	}
	wg.Wait()
}

func BenchmarkInt64N(b *testing.B) {
	for _, n := range []int64{1, 36, 1<<62 + 1, math.MaxInt64} {
		b.Run(strconv.FormatInt(n, 10), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				Int64N(n)
			}
		})
	}
}
