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

package runtimex

import (
	"testing"
	"time"
)

func TestRegressionNilZeroer(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("IsZero((*time.Time)(nil)) panics: %v", r)
		}
	}()

	var p *time.Time
	if !IsZero(p) {
		t.Error("typed nil must be zero")
	}
}

func TestRegressionFinalStackFrame(t *testing.T) {
	frames := Stacks(0)
	if len(frames) == 0 || frames[len(frames)-1].Func != "goexit" {
		t.Fatalf("runtime.goexit omitted from final stack frame: %+v", frames)
	}
}
