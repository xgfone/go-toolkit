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

package random

import (
	"errors"
	"io"
	"math"
	"math/big"
	"testing"
)

// TestInt64N_ErrorBranch tests the error handling branch in Int64N function.
// This test uses a technique to mock the crypto/rand.Int function by
// temporarily replacing it during the test.
func TestInt64N_ErrorBranch(t *testing.T) {
	// Save the original crypto/rand.Int function
	originalCryptoRandInt := cryptoRandInt

	// Set up a mock that returns an error
	cryptoRandInt = func(rand io.Reader, max *big.Int) (n *big.Int, err error) {
		return nil, errors.New("simulated crypto/rand.Int error for testing")
	}

	// Restore the original function when test completes
	defer func() {
		cryptoRandInt = originalCryptoRandInt
	}()

	// Test with various values (all n > 0 to avoid panic in rand.Int64N)
	testCases := []struct {
		name string
		n    int64
	}{
		{"One", 1},
		{"Small", 10},
		{"Medium", 100},
		{"Large", 1000},
		{"VeryLarge", math.MaxInt64 / 2},
		{"MaxInt64", math.MaxInt64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Int64N(tc.n)

			// When crypto/rand.Int fails, Int64N should fall back to rand.Int64N
			// The result should be in range [0, n)
			if result < 0 || result >= tc.n {
				t.Errorf("Int64N(%d) = %d, expected value in [0, %d)", tc.n, result, tc.n)
			}
		})
	}
}

// TestInt64N_ZeroWithError tests edge case with n=0
func TestInt64N_ZeroWithError(t *testing.T) {
	// Save the original crypto/rand.Int function
	originalCryptoRandInt := cryptoRandInt

	// Set up a mock that returns an error
	cryptoRandInt = func(rand io.Reader, max *big.Int) (n *big.Int, err error) {
		return nil, errors.New("simulated error for n=0")
	}

	// Restore the original function when test completes
	defer func() {
		cryptoRandInt = originalCryptoRandInt
	}()

	// When n=0, crypto/rand.Int should panic according to documentation,
	// but our mock returns an error instead. Then rand.Int64N(0) will also panic.
	// We need to handle this panic in the test.
	defer func() {
		if r := recover(); r != nil {
			// Expected panic from rand.Int64N(0)
			t.Logf("Recovered from expected panic: %v", r)
		}
	}()

	result := Int64N(0)
	// If we get here without panic (unlikely), check the result
	if result != 0 {
		t.Errorf("Int64N(0) = %d, expected 0", result)
	}
}

// TestInt64N_MultipleErrorCalls tests that the error branch works correctly
// even when called multiple times
func TestInt64N_MultipleErrorCalls(t *testing.T) {
	// Save the original crypto/rand.Int function
	originalCryptoRandInt := cryptoRandInt

	callCount := 0
	// Set up a mock that returns an error every time
	cryptoRandInt = func(rand io.Reader, max *big.Int) (n *big.Int, err error) {
		callCount++
		return nil, errors.New("simulated error")
	}

	// Restore the original function when test completes
	defer func() {
		cryptoRandInt = originalCryptoRandInt
	}()

	// Call Int64N multiple times (start from 1 to avoid n=0)
	for i := range 10 {
		n := int64(i + 1)
		result := Int64N(n)

		if result < 0 || result >= n {
			t.Errorf("Call %d: Int64N(%d) = %d, expected value in [0, %d)",
				i+1, n, result, n)
		}
	}

	// Verify the mock was called
	if callCount != 10 {
		t.Errorf("Expected cryptoRandInt to be called 10 times, got %d", callCount)
	}
}

// TestInt64N_MixedSuccessAndError tests scenario where some calls succeed and some fail
func TestInt64N_MixedSuccessAndError(t *testing.T) {
	// Save the original crypto/rand.Int function
	originalCryptoRandInt := cryptoRandInt

	callSequence := []bool{true, false, true, false} // true = success, false = error
	callIndex := 0

	// Set up a mock that alternates between success and error
	cryptoRandInt = func(rand io.Reader, max *big.Int) (n *big.Int, err error) {
		if callIndex >= len(callSequence) {
			callIndex = 0
		}

		shouldSucceed := callSequence[callIndex]
		callIndex++

		if shouldSucceed {
			// Return a valid value (e.g., 5)
			return big.NewInt(5), nil
		} else {
			return nil, errors.New("simulated error")
		}
	}

	// Restore the original function when test completes
	defer func() {
		cryptoRandInt = originalCryptoRandInt
	}()

	// Test with a fixed n value
	n := int64(10)

	for i := range 4 {
		result := Int64N(n)

		// Result should always be in range [0, n)
		if result < 0 || result >= n {
			t.Errorf("Call %d: Int64N(%d) = %d, expected value in [0, %d)",
				i+1, n, result, n)
		}
	}
}
