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

// Go starts a lifecycle-managed background task on [Default]. See [App.Go].
func Go(name string, callback func(ctx context.Context) error) {
	defaultApp.Go(name, callback)
}

// Go starts a lifecycle-managed background task. The name may be empty but is
// useful for error messages.
//
// It can only be called after [App.Run] starts, usually inside [Module.Start]
// or callbacks. If callback returns a non-nil error while [App] is still running,
// [App] will start shutdown. The task must observe context cancellation so
// shutdown can finish. The app waits for managed tasks after [Module.Stop] and
// before [StageCleanup], subject to its shutdown timeout. Goroutines started
// directly with go are not managed or waited for.
func (a *App) Go(name string, callback func(ctx context.Context) error) {
	if callback == nil {
		panic("app: nil background task func")
	}

	a.mu.Lock()

	if a.state != stateRunning {
		a.mu.Unlock()
		panic("app: Go can only be called while app is running")
	}

	runCtx := a.runCtx
	errCh := a.errCh
	cancelRun := a.cancelRun

	// Add must be protected by the same state lock.
	// This avoids racing with shutdown waiting on the WaitGroup.
	a.wg.Add(1)

	a.mu.Unlock()

	go func() {
		defer a.wg.Done()

		if err := saferun(runCtx, callback); err != nil {
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
			cancelRun()
		}
	}()
}

func saferun(ctx context.Context, callback func(ctx context.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = fmt.Errorf("panic: %w", e)
			} else {
				err = fmt.Errorf("panic: %v", r)
			}
		}
	}()
	return callback(ctx)
}
