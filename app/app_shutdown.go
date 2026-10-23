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
	"time"
)

// SetCleanupTimeout sets the budget shared by Module.Close and StageCleanup.
// The default is 5 seconds. It starts after draining succeeds, independently
// of the drain budget. All resource closers are attempted even after timeout.
// It must be called before Run.
func (a *App) SetCleanupTimeout(timeout time.Duration) {
	a.setPhaseTimeout("SetCleanupTimeout", &a.cleanupTimeout, timeout)
}

// SetExitedTimeout sets the budget for StageExited callbacks. The default is
// 5 seconds. It starts after Cleanup, independently of the cleanup budget.
// It must be called before Run.
func (a *App) SetExitedTimeout(timeout time.Duration) {
	a.setPhaseTimeout("SetExitedTimeout", &a.exitedTimeout, timeout)
}

// SetFinalizerTimeout sets the budget for the single callback registered with
// Finalize. The default is 5 seconds. It starts after all StageExited callbacks,
// so a slow exit notification cannot consume the logger's flush budget.
// It must be called before Run.
func (a *App) SetFinalizerTimeout(timeout time.Duration) {
	a.setPhaseTimeout("SetFinalizerTimeout", &a.finalizerTimeout, timeout)
}

func (a *App) setPhaseTimeout(method string, target *time.Duration, timeout time.Duration) {
	if timeout <= 0 {
		panic(fmt.Sprintf("app: %s timeout must be positive", method))
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.mustBeNewLocked(method)
	*target = timeout
}

// Finalize registers one final callback, normally to flush and close the logger.
// It runs after all StageExited callbacks with a fresh, independently timed
// context. Register it when initializing logging, including from StageInit.
// A second registration, a nil callback, or registration after finalization
// starts panics. The name may be empty.
//
// Finalization is skipped on ErrShutdownIncomplete, keeping logging available
// to work that has not exited. The app does not report Run's final error through
// the logger after closing it; callers can report that error to os.Stderr.
func (a *App) Finalize(name string, callback func(context.Context) error) {
	if callback == nil {
		panic("app: nil finalizer")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state == stateExited || a.finalizing {
		panic("app: cannot register finalizer after finalization starts")
	}
	if a.finalizer != nil {
		panic("app: finalizer already registered")
	}

	a.finalizer = &namedCallback{name: name, callback: callback}
}
