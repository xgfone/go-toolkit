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

package app

import (
	"context"
	"fmt"
	"sync"
)

// TaskGroup owns a module's business tasks. Its context retains Run's values,
// but not its cancellation or deadline: dependencies keep working while their
// users drain. The app stops and joins it in the owning module's shutdown batch.
// Modules may use it for servers, consumers, refreshers, and other business
// goroutines. Resource-only modules may leave it unused. Module.Wait joins
// any additional business activity outside the group; both joins must succeed.
// Internal logger writers must be owned and joined by the finalizer instead.
// Obtain a group using App.TaskGroup; its zero value must not be used.
type TaskGroup struct {
	mu sync.Mutex

	app  *App
	name string

	ctx    context.Context
	cancel context.CancelFunc

	stopping bool
	active   int
	done     chan struct{}
}

func newTaskGroup(a *App, name string, base context.Context) *TaskGroup {
	ctx, cancel := context.WithCancel(context.WithoutCancel(base))
	return &TaskGroup{
		app:    a,
		name:   name,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// TaskGroup returns a named module's group after Run validates the module plan.
// The group remains available after Run returns, including for live users of
// retained providers on ErrShutdownIncomplete. Stopped groups still reject Go.
// Calls before plan validation and unknown names panic.
func (a *App) TaskGroup(module string) *TaskGroup {
	a.mu.Lock()
	defer a.mu.Unlock()
	g := a.groups[module]
	if g == nil {
		panic(fmt.Sprintf("app: task group for module %q is unavailable", module))
	}
	return g
}

// Context is canceled when this group's shutdown batch begins, after Module.Stop.
func (g *TaskGroup) Context() context.Context { return g.ctx }

// Done closes when admission is sealed and all admitted tasks have returned.
func (g *TaskGroup) Done() <-chan struct{} { return g.done }

// Go starts owned work. It may be called while upstream modules drain, until
// this group is stopped. Nil callbacks and registration after Stop panic.
// Errors and panics trigger App.Stop and are included in Run's final result.
func (g *TaskGroup) Go(name string, callback func(context.Context) error) {
	if callback == nil {
		panic("app: nil background task func")
	}

	g.mu.Lock()
	if g.stopping {
		g.mu.Unlock()
		panic("app: task group is stopping")
	}
	g.active++
	g.mu.Unlock()

	go func() {
		defer g.finish()
		if err := saferun(g.ctx, callback); err != nil {
			g.app.reportTaskError(g.ctx, g.name, name, err)
		}
	}()
}

// Stop seals admission and cancels this group's work context. It is idempotent.
// Normally the app calls it after the owning Module.Stop initiates shutdown.
// Module.Stop can focus on the business stop protocol.
func (g *TaskGroup) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.stopping {
		return
	}

	g.stopping = true
	g.cancel()
	if g.active == 0 {
		close(g.done)
	}
}

// Wait joins the sealed task set, or returns ctx.Err. It neither stops tasks
// nor consumes their errors; Run reports task errors separately. Completion is
// preferred over an already canceled wait context. Multiple waiters are safe.
// The app performs this join independently of Module.Wait; modules do not need
// to repeat it when implementing Module.Wait.
func (g *TaskGroup) Wait(ctx context.Context) error {
	select {
	case <-g.done:
		return nil
	default:
	}

	select {
	case <-g.done:
		return nil

	case <-ctx.Done():
		select {
		case <-g.done:
			return nil

		default:
			return ctx.Err()
		}
	}
}

func (g *TaskGroup) finish() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.active--
	if g.stopping && g.active == 0 {
		close(g.done)
	}
}
