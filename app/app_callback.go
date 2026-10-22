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

type namedCallback struct {
	name     string
	callback func(ctx context.Context) error
}

// On registers a lifecycle callback.
//
// The name may be empty but is useful for error messages.
// Capture the registering [App] and any other dependencies in a closure.
// The context is supplied when the callback executes.
//
// Before [App.Run] starts, callbacks may be registered for any valid stage.
//
// After [App.Run] starts, callbacks may only be registered for future stages.
// The [App] tracks the lifecycle stage it has reached, and registering a
// callback for the current or a past stage will panic.
//
// Callbacks are executed in registration order for most stages. The only
// exception is [StageCleanup]: callbacks registered for [StageCleanup] are
// executed in reverse registration order.
//
// Startup callbacks receive the app's run context. Shutdown callbacks share a
// separate context with the deadline configured by [App.SetShutdownTimeout].
// Shutdown continues calling callbacks even after that context expires or a
// callback fails.
func (a *App) On(stage Stage, name string, callback func(ctx context.Context) error) {
	if callback == nil {
		panic("app: nil callback")
	}

	phase := phaseForStage(stage)
	if phase == phaseCount {
		panic(fmt.Sprintf("app: invalid stage %q", stage))
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if phase <= a.stage {
		panic(fmt.Sprintf("app: cannot register callback for stage %q after stage %q",
			stage, lifecyclePhases[a.stage].stage))
	}

	a.registerCallbackLocked(phase, name, callback)
}

// Cleanup registers a simple close function for [StageCleanup]. The function
// does not receive the shutdown context or its deadline. Use [App.On] with
// [StageCleanup] for context-aware cleanup.
func (a *App) Cleanup(fn func() error) {
	if fn == nil {
		panic("app: nil cleanup function")
	}
	a.On(StageCleanup, "", func(context.Context) error { return fn() })
}

func callbackLabel(stage Stage, name string, index int) string {
	if name != "" {
		return fmt.Sprintf("%q at stage %q", name, stage)
	}
	return fmt.Sprintf("#%d at stage %q", index, stage)
}
