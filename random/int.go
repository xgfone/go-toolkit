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
	crand "crypto/rand"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"strconv"
)

// cryptoSource supplies cryptographic randomness to math/rand/v2's sampling algorithms.
type cryptoSource struct{}

func (cryptoSource) Uint64() uint64 {
	var buf [8]byte
	_, _ = crand.Read(buf[:]) // Read always fills buf and never returns an error.
	return binary.LittleEndian.Uint64(buf[:])
}

// SeedString returns a random 64-bit signed integer string.
func SeedString() string { return strconv.FormatInt(Seed(), 10) }

// Seed returns a random 64-bit signed integer seed.
func Seed() int64 { return Int64N(math.MaxInt64) }

// IntN returns a uniformly distributed, cryptographically secure random integer
// in [0, n) as int. It panics if n <= 0.
func IntN(n int) int { return int(Int64N(int64(n))) }

// Int64N returns a uniformly distributed, cryptographically secure random integer
// in [0, n) as int64. It panics if n <= 0.
func Int64N(n int64) int64 {
	if n == 1 {
		return 0
	}
	return rand.New(cryptoSource{}).Int64N(n)
}
