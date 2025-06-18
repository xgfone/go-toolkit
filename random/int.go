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

package random

import (
	crand "crypto/rand"
	"log/slog"
	"math"
	"math/big"
	"math/rand/v2"
	"strconv"
	"sync"
)

// SeedString returns a random 64-bit signed integer string.
func SeedString() string { return strconv.FormatInt(Seed(), 10) }

// Seed returns a random 64-bit signed integer seed.
func Seed() int64 { return Int64N(math.MaxInt64) }

// IntN returns a random integer in [0, n) as int.
func IntN(n int) int { return int(Int64N(int64(n))) }

// Int64N returns a random integer in [0, n) as int64.
func Int64N(n int64) int64 {
	var v int64
	max := getBigInt(n)
	if m, err := crand.Int(crand.Reader, max); err != nil {
		slog.Error("crypto/rand.Int failed", "n", n, "err", err)
		v = rand.Int64N(n)
	} else {
		v = m.Int64()
	}
	return v
}

var (
	_block  = new(sync.Mutex)
	_bigmap = new(sync.Map)
)

func getBigInt(n int64) *big.Int {
	if value, loaded := _bigmap.Load(n); loaded {
		return value.(*big.Int)
	}

	_block.Lock()
	defer _block.Unlock()

	if value, loaded := _bigmap.Load(n); loaded {
		return value.(*big.Int)
	}

	v := big.NewInt(n)
	_bigmap.Store(n, v)
	return v
}
