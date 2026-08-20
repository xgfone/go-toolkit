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

type testBoth struct{}

func (testBoth) Method()        {}
func (testBoth) AnotherMethod() {}

type testWrapper struct {
	next TestInterface
}

func (testWrapper) Method()                 {}
func (w testWrapper) Unwrap() TestInterface { return w.next }

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

func TestAs(t *testing.T) {
	t.Run("concrete to concrete", func(t *testing.T) {
		if _, ok := As[TestType](TestType{}); !ok {
			t.Error("same concrete type did not match")
		}
		if _, ok := As[TestType2](TestType{}); ok {
			t.Error("different concrete types matched")
		}
	})

	t.Run("interface to interface", func(t *testing.T) {
		var source TestInterface = testBoth{}
		if _, ok := As[AnotherInterface](source); !ok {
			t.Error("dynamic value implementing both interfaces did not match")
		}

		source = TestType{}
		if _, ok := As[AnotherInterface](source); ok {
			t.Error("dynamic value not implementing target interface matched")
		}
	})

	t.Run("concrete to interface", func(t *testing.T) {
		if _, ok := As[TestInterface](TestType{}); !ok {
			t.Error("concrete type implementing target interface did not match")
		}
		if _, ok := As[AnotherInterface](TestType{}); ok {
			t.Error("concrete type not implementing target interface matched")
		}
	})

	t.Run("interface to concrete", func(t *testing.T) {
		var source TestInterface = testWrapper{next: TestType{}}
		if _, ok := As[TestType](source); !ok {
			t.Error("wrapped concrete value did not match")
		}
		if _, ok := As[TestType3](source); ok {
			t.Error("concrete type not implementing source interface matched")
		}

		source = testWrapper{}
		if _, ok := As[TestType](source); ok {
			t.Error("nil unwrap result matched")
		}
	})
}

func BenchmarkImplements(b *testing.B) {
	typ := reflect.TypeFor[TestType]()
	target := reflect.TypeFor[TestInterface]()

	// Ensure cache is populated
	Implements(typ, target)

	b.ResetTimer()
	for b.Loop() {
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

func ExampleAs() {
	var ok bool

	_, ok = As[TestType](TestType{})
	fmt.Println("concrete to concrete:", ok)

	var source TestInterface = testBoth{}
	_, ok = As[AnotherInterface](source)
	fmt.Println("interface to interface:", ok)

	_, ok = As[TestInterface](TestType{})
	fmt.Println("concrete to interface:", ok)

	source = testWrapper{next: TestType{}}
	_, ok = As[TestType](source)
	fmt.Println("interface to concrete:", ok)

	// Output:
	// concrete to concrete: true
	// interface to interface: true
	// concrete to interface: true
	// interface to concrete: true
}
