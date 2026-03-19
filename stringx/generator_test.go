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
	"testing"
	"time"
)

func TestNewBuilder(t *testing.T) {
	// Test panic when generator is nil
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when generator is nil")
		}
	}()
	NewBuilder(nil)
}

func TestBuilder(t *testing.T) {
	// Test WithPrefix
	b1 := NewBuilder(DateTimeRandGenerator)
	b2 := b1.WithPrefix("prefix")
	if b2.prefix != "prefix" {
		t.Error("expected prefix to be 'prefix'")
	}

	// Test WithSuffix
	b3 := b2.WithSuffix("suffix")
	if b3.suffix != "suffix" {
		t.Error("expected suffix to be 'suffix'")
	}

	// Test WithGenerator
	b4 := b3.WithGenerator(UnixTimeRandGenerator)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when generator is nil in WithGenerator")
		}
	}()
	b4.WithGenerator(nil)
}

func TestBuilderBuild(t *testing.T) {
	b := NewBuilder(DateTimeRandGenerator)

	// Test with length <= 0
	s1 := b.Build(0)
	if len(s1) != b.MinLen() {
		t.Errorf("expected length %d, got %d", b.MinLen(), len(s1))
	}

	// Test with valid length
	s2 := b.Build(20)
	if len(s2) != 20 {
		t.Errorf("expected length 20, got %d", len(s2))
	}

	// Test panic when length is less than min length
	smallLen := b.MinLen() - 1
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when length %d is less than min length %d", smallLen, b.MinLen())
		}
	}()
	b.Build(smallLen)
}

func TestBuilderGenerate(t *testing.T) {
	b := NewBuilder(DateTimeRandGenerator)
	buf := make([]byte, 0, 20)
	newBuf := b.Generate(buf, 20)
	if len(newBuf) != 20 {
		t.Errorf("expected length 20, got %d", len(newBuf))
	}
}

func TestBuilderMinLen(t *testing.T) {
	b := NewBuilder(DateTimeRandGenerator)
	minLen := b.MinLen()
	if minLen <= 0 {
		t.Error("expected min length to be positive")
	}

	// Test with prefix and suffix
	b2 := b.WithPrefix("prefix").WithSuffix("suffix")
	minLen2 := b2.MinLen()
	if minLen2 <= minLen {
		t.Error("expected min length to increase with prefix and suffix")
	}
}

func TestNewGenerator(t *testing.T) {
	g := NewGenerator(5, func(buf []byte, length int) []byte {
		return append(buf, []byte("test")...)
	})
	if g.MinLen() != 5 {
		t.Error("expected min length to be 5")
	}
}

func TestGenerate(t *testing.T) {
	for i := 10; i < 20; i++ {
		buf := Generate(nil, i, func(b []byte, _ time.Time) []byte {
			return append(b, []byte("test")...)
		})
		if len(buf) < i {
			t.Errorf("expected length at least %d, but got %d", i, len(buf))
		}
	}
}

func TestPredefinedGenerators(t *testing.T) {
	generators := []Generator{
		DateTimeMilliRandGenerator,
		UnixTimeMilliRandGenerator,
		DateTimeRandGenerator,
		UnixTimeRandGenerator,
	}

	for _, g := range generators {
		if g.MinLen() <= 0 {
			t.Errorf("expected min length to be positive for generator")
		}

		buf := make([]byte, 0, 20)
		newBuf := g.Generate(buf, 20)
		if len(newBuf) != 20 {
			t.Errorf("expected length 20, got %d", len(newBuf))
		}
	}
}
