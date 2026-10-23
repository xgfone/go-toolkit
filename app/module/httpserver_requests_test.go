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

package module

import (
	"context"
	"sync"
	"testing"
)

type completionContext struct {
	context.Context
	complete func()
}

func (c completionContext) Done() <-chan struct{} {
	c.complete()
	return c.Context.Done()
}

func TestRequestCompletionWinsCanceledWait(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	for range 64 {
		g := &requestGroup{done: make(chan struct{})}
		g.enter()
		g.stop()
		ctx := completionContext{canceled, sync.OnceFunc(g.leave)}
		if err := g.wait(ctx); err != nil {
			t.Fatalf("completed request lost to canceled wait: %v", err)
		}
	}
}
