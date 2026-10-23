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
)

// requestGroup closes admission before joining. A handler racing with shutdown
// is either counted or rejected, so Wait never races with a new uncounted user.
type requestGroup struct {
	mu sync.Mutex

	active  int
	closing bool

	done chan struct{}
}

func (g *requestGroup) enter() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closing {
		return false
	}

	g.active++
	return true
}

func (g *requestGroup) leave() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.active--
	if g.closing && g.active == 0 {
		close(g.done)
	}
}

func (g *requestGroup) stop() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closing {
		return false
	}

	g.closing = true
	if g.active == 0 {
		close(g.done)
	}

	return true
}

func (g *requestGroup) wait(ctx context.Context) error {
	return waitHTTPShutdown(ctx, g.done)
}

// Completion takes precedence when cancellation and completion race. Waiting
// only observes the operation; the context passed to Stop controls shutdown.
func waitHTTPShutdown(ctx context.Context, done <-chan struct{}) error {
	select {
	case <-done:
		return nil

	default:
	}

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		select {
		case <-done:
			return nil

		default:
			return ctx.Err()
		}
	}
}
