// Copyright 2024 xgfone
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

package runtimex

import (
	"fmt"
	"strings"
	"testing"
)

func ExampleTrimPkgFile() {
	srcfile := TrimPkgFile("/path/to/src/github.com/xgfone/go-toolkit/srcfile.go")
	modfile := TrimPkgFile("/path/to/pkg/mod/github.com/xgfone/go-toolkit/modfile.go")
	origfile := TrimPkgFile("/path/to/github.com/xgfone/go-toolkit/modfile.go")

	fmt.Println(srcfile)
	fmt.Println(modfile)
	fmt.Println(origfile)

	// Output:
	// github.com/xgfone/go-toolkit/srcfile.go
	// github.com/xgfone/go-toolkit/modfile.go
	// /path/to/github.com/xgfone/go-toolkit/modfile.go
}

func TestCaller(t *testing.T) {
	caller := Caller(0)
	expect := "github.com/xgfone/go-toolkit/runtimex/stack_test.go:TestCaller:39"
	if caller.String() != expect {
		t.Errorf("expect '%s', but got '%s'", expect, caller.String())
	}
}

func TestStacks(t *testing.T) {
	stacks := Stacks(0)
	for i, stack := range stacks {
		if strings.HasPrefix(stack.File, "testing/") {
			stacks = stacks[:i]
			break
		}
	}

	expects := []string{
		"github.com/xgfone/go-toolkit/runtimex/stack_test.go:TestStacks:47",
	}

	if len(expects) != len(stacks) {
		t.Fatalf("expect %d line, but got %d: %v", len(expects), len(stacks), stacks)
	}

	for i, line := range expects {
		if line != stacks[i].String() {
			t.Errorf("%d: expect '%s', but got '%s'", i, line, stacks[i].String())
		}
	}
}
