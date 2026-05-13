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
	"errors"
	"fmt"
	"iter"
	"slices"
)

// On is short for DefaultApp.On(stage, hook).
//
// It must be called before Run.
func On(stage Stage, hook func(context.Context, *App) error) {
	DefaultApp.On(stage, hook)
}

// OnNamed is short for DefaultApp.OnNamed(stage, name, hook).
//
// It must be called before Run.
func OnNamed(stage Stage, name string, hook func(context.Context, *App) error) {
	DefaultApp.OnNamed(stage, name, hook)
}

// On is short for App.OnNamed(stage, "", hook).
//
// It must be called before Run.
func (a *App) On(stage Stage, hook func(context.Context, *App) error) {
	a.OnNamed(stage, "", hook)
}

// On registers a lifecycle hook.
//
// Name is optional but useful for error messages.
//
// Before Run starts, hooks may be registered for any valid stage.
//
// After Run starts, hooks may only be registered for future stages. The App
// tracks the lifecycle stage it has reached, and registering a hook for the
// current or a past stage will panic.
//
// Hooks are executed in registration order for most stages. The only exception
// is StageCleanup: hooks registered for StageCleanup are executed in reverse
// registration order.
func (a *App) OnNamed(stage Stage, name string, hook func(context.Context, *App) error) {
	if hook == nil {
		panic("app: nil hook")
	}

	if !validStage(stage) {
		panic(fmt.Sprintf("app: invalid stage %q", stage))
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.canRegisterHookLocked(stage) {
		panic(fmt.Sprintf("app: cannot register hook for stage %q after stage %q", stage, a.stage))
	}

	a.hooks[stage] = append(a.hooks[stage], namedCtxAppFunc{name: name, fn: hook})
}

func (a *App) runHooks(ctx context.Context, stage Stage) error {
	a.mu.Lock()
	a.stage = stage
	hooks := slices.Clone(a.hooks[stage])
	a.mu.Unlock()

	var errs []error
	var seq2 iter.Seq2[int, namedCtxAppFunc]

	if stage == StageCleanup {
		seq2 = slices.Backward(hooks)
	} else {
		seq2 = slices.All(hooks)
	}

	for i, hook := range seq2 {
		if err := hook.fn(ctx, a); err != nil {
			wrapped := fmt.Errorf("app: hook %s: %w", hookLabel(stage, hook.name, i), err)

			// During shutdown stages, continue executing remaining hooks.
			if stage == StageStopping || stage == StageCleanup || stage == StageExited {
				errs = append(errs, wrapped)
				continue
			}

			return wrapped
		}
	}

	return errors.Join(errs...)
}

func hookLabel(stage Stage, name string, index int) string {
	if name != "" {
		return fmt.Sprintf("%q at stage %q", name, stage)
	}
	return fmt.Sprintf("#%d at stage %q", index, stage)
}
