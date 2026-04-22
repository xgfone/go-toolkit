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

package reflectx

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// TestInterface defines a simple interface for testing.
type TestInterface interface {
	Method()
}

// TestType implements TestInterface.
type TestType struct{}

func (TestType) Method() {}

// TestType2 does NOT implement TestInterface.
type TestType2 struct{}

// AnotherInterface defines another interface for testing.
type AnotherInterface interface {
	AnotherMethod()
}

// TestType3 implements AnotherInterface but not TestInterface.
type TestType3 struct{}

func (TestType3) AnotherMethod() {}

func TestImplements(t *testing.T) {
	// Clear cache before benchmark
	typeImplements = new(sync.Map)

	// Get reflection types
	typ1 := reflect.TypeFor[TestType]()
	typ2 := reflect.TypeFor[TestType2]()
	typ3 := reflect.TypeFor[TestType3]()

	target1 := reflect.TypeFor[TestInterface]()
	target2 := reflect.TypeFor[AnotherInterface]()

	// Test 1: Type that implements the interface
	if !Implements(typ1, target1) {
		t.Error("TestType should implement TestInterface")
	}

	// Test 2: Type that does NOT implement the interface
	if Implements(typ2, target1) {
		t.Error("TestType2 should NOT implement TestInterface")
	}

	// Test 3: Type that implements a different interface
	if Implements(typ3, target1) {
		t.Error("TestType3 should NOT implement TestInterface")
	}

	// Test 4: Type that implements AnotherInterface
	if !Implements(typ3, target2) {
		t.Error("TestType3 should implement AnotherInterface")
	}

	// Test 5: Type that does NOT implement AnotherInterface
	if Implements(typ1, target2) {
		t.Error("TestType should NOT implement AnotherInterface")
	}

	// Test 6: Cache hit.
	if Implements(typ1, target2) {
		t.Error("TestType should NOT implement AnotherInterface")
	}
}

func BenchmarkImplements(b *testing.B) {
	typ := reflect.TypeFor[TestType]()
	target := reflect.TypeFor[TestInterface]()

	// Ensure cache is populated
	Implements(typ, target)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Implements(typ, target)
	}
}

func ExampleImplements() {
	typ := reflect.TypeFor[*bytes.Buffer]()
	iface := reflect.TypeFor[fmt.Stringer]()

	ok := Implements(typ, iface)
	fmt.Println(ok)
	// Output:
	// true
}
