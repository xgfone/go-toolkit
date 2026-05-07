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
)

// Go is a convenience function that calls DefaultApp.Go.
func Go(fn func(ctx context.Context) error) {
	GoNamed("", fn)
}

// GoNamed is a convenience function that calls DefaultApp.GoNamed.
func GoNamed(name string, fn func(ctx context.Context) error) {
	DefaultApp.GoNamed(name, fn)
}

// Go is short for App.GoNamed("", fn).
func (a *App) Go(fn func(ctx context.Context) error) {
	a.GoNamed("", fn)
}

// GoNamed starts a lifecycle-managed background task with the optional name.
//
// It can only be called after Run starts, usually inside Module.Start or hooks.
// If fn returns a non-nil error while App is still running, App will start shutdown.
func (a *App) GoNamed(name string, fn func(ctx context.Context) error) {
	if fn == nil {
		panic("app: nil background task func")
	}

	a.mu.Lock()

	if a.state != stateRunning {
		a.mu.Unlock()
		panic("app: Go can only be called while app is running")
	}

	runCtx := a.runCtx
	errCh := a.errCh

	// Add must be protected by the same state lock.
	// This avoids racing with shutdown waiting on the WaitGroup.
	a.wg.Add(1)

	a.mu.Unlock()

	go func() {
		defer a.wg.Done()

		if err := fn(runCtx); err != nil {
			// If the app is already shutting down, the task error is usually
			// a consequence of cancellation and should not trigger another shutdown.
			if runCtx.Err() != nil {
				return
			}

			wrapped := fmt.Errorf("app: background task %q: %w", name, err)

			select {
			case errCh <- wrapped:
			default:
			}
		}
	}()
}
