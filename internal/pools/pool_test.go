// Copyright 2025 xgfone
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

package pools

import "testing"

func TestGetBuffer(t *testing.T) {
	// Test 256 buffer
	t.Run("256", func(t *testing.T) {
		p, b := GetBuffer(256)
		if p == nil || b == nil {
			t.Errorf("failed to get the Buffer")
		} else if b.Cap() != 256 {
			t.Errorf("expect cap %d, but got %d", 256, b.Cap())
		} else if b.Len() != 0 {
			t.Errorf("expect len 0, but got %d", b.Len())
		}
		PutBuffer(p, b)
	})

	// Test 1KB buffer
	t.Run("1KB", func(t *testing.T) {
		p, b := GetBuffer(1024)
		if p == nil || b == nil {
			t.Errorf("failed to get the Buffer")
		} else if b.Cap() != 1024 {
			t.Errorf("expect cap %d, but got %d", 1024, b.Cap())
		} else if b.Len() != 0 {
			t.Errorf("expect len 0, but got %d", b.Len())
		}
		PutBuffer(p, b)
	})

	// Test 64KB buffer
	t.Run("64KB", func(t *testing.T) {
		p, b := GetBuffer(64 * 1024)
		if p == nil || b == nil {
			t.Errorf("failed to get the Buffer")
		} else if b.Cap() != 64*1024 {
			t.Errorf("expect cap %d, but got %d", 64*1024, b.Cap())
		} else if b.Len() != 0 {
			t.Errorf("expect len 0, but got %d", b.Len())
		}
		PutBuffer(p, b)
	})

	// Test panic for unsupported capacity
	t.Run("panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expect an error, but got nil")
			}
		}()

		p, b := GetBuffer(123)
		PutBuffer(p, b)
	})
}

func TestPutBuffer(t *testing.T) {
	// Test that PutBuffer properly resets the buffer
	p, b := GetBuffer(256)

	// Write some data to the buffer
	data := []byte("test data")
	b.Write(data)

	if b.Len() != len(data) {
		t.Errorf("expect len %d after write, but got %d", len(data), b.Len())
	}

	// Put it back
	PutBuffer(p, b)

	// Get it again to verify it was reset
	p2, b2 := GetBuffer(256)
	if b2.Len() != 0 {
		t.Errorf("expect len 0 after PutBuffer and GetBuffer, but got %d", b2.Len())
	}
	PutBuffer(p2, b2)
}

func TestBufferReuse(t *testing.T) {
	// Test that buffers are properly reused from the pool
	p1, b1 := GetBuffer(256)
	ptr1 := b1

	// Put it back
	PutBuffer(p1, b1)

	// Get another buffer - should be the same one
	p2, b2 := GetBuffer(256)
	ptr2 := b2

	if ptr1 != ptr2 {
		t.Log("Note: Got different buffer pointers, which is OK if pool was empty")
	}

	PutBuffer(p2, b2)
}

func TestConcurrentAccess(t *testing.T) {
	// Test concurrent access to the pool
	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := range goroutines {
		go func(id int) {
			p, b := GetBuffer(256)
			// Write some unique data
			b.WriteString("goroutine ")
			b.WriteString(string(rune('0' + id)))
			PutBuffer(p, b)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range goroutines {
		<-done
	}

	// Verify we can still get a buffer
	p, b := GetBuffer(256)
	if b.Len() != 0 {
		t.Errorf("expect len 0, but got %d", b.Len())
	}
	PutBuffer(p, b)
}
