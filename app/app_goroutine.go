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

// Go starts an application-level task on Default. See App.Go.
func Go(name string, callback func(context.Context) error) {
	defaultApp.Go(name, callback)
}

// ReportError records an asynchronous failure in Run's final result and
// requests stopping. It is safe during shutdown; errors after Run are ignored.
// Prefer returning errors from managed tasks when possible.
func (a *App) ReportError(err error) {
	if err == nil {
		return
	}

	a.mu.Lock()
	if a.state == stateNew || a.state == stateExited {
		a.mu.Unlock()
		return
	}

	a.taskErrors = append(a.taskErrors, err)
	cancel := a.cancelRun
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// Go starts an application-level entry task using the startup/run context.
// It must be called during Run, before stopping. These tasks are canceled when
// stopping is requested and joined with the outermost module batch, before any
// dependency batch stops. They have no per-module dependency ordering. Use
// TaskGroup.Go for module-owned work, especially servers, queues, and workers
// that must stay available while their users drain. Unmanaged goroutines and
// detached child work cannot be joined by the app.
func (a *App) Go(name string, callback func(context.Context) error) {
	if callback == nil {
		panic("app: nil background task func")
	}

	a.mu.Lock()
	if a.state != stateRunning {
		a.mu.Unlock()
		panic("app: Go can only be called while app is running")
	}
	// Seal and admission use the same app lock. The group's context is the
	// actual run context for backwards-compatible app-level cancellation.
	a.rootTasks.Go(name, callback)
	a.mu.Unlock()
}

func (a *App) reportTaskError(ctx context.Context, group, name string, err error) {
	// Ordinary cancellation is expected. Other errors, including errors during
	// graceful shutdown, must remain visible in Run and every App.Wait result.
	if ctx.Err() != nil && onlyContextError(err, ctx.Err()) {
		return
	}

	var label string
	if group == "" {
		label = fmt.Sprintf("app: background task %q", name)
	} else {
		label = fmt.Sprintf("app: module %q background task %q", group, name)
	}

	wrapped := fmt.Errorf("%s: %w", label, err)
	a.mu.Lock()
	if a.state != stateExited {
		a.taskErrors = append(a.taskErrors, wrapped)
	}
	cancel := a.cancelRun
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// Do not discard a real failure joined with an expected cancellation error.
func onlyContextError(err, target error) bool {
	if err == target {
		return true
	}

	switch e := err.(type) {
	case panicError:
		return false

	case interface{ Unwrap() []error }:
		errs := e.Unwrap()
		if len(errs) == 0 {
			return false
		}

		for _, nested := range errs {
			if !onlyContextError(nested, target) {
				return false
			}
		}
		return true

	case interface{ Unwrap() error }:
		return onlyContextError(e.Unwrap(), target)

	default:
		return false
	}
}

// Panics remain failures even if their payload is a cancellation sentinel.
type panicError struct{ error }

func (e panicError) Unwrap() error { return e.error }

func saferun(ctx context.Context, callback func(context.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = panicError{fmt.Errorf("panic: %w", e)}
			} else {
				err = panicError{fmt.Errorf("panic: %v", r)}
			}
		}
	}()
	return callback(ctx)
}
