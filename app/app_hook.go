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
)

// On registers a lifecycle hook for the default app.
//
// It must be called before Run.
func On(stage Stage, hook func(context.Context, *App) error) {
	DefaultApp.On(stage, hook)
}

// OnNamed registers a named lifecycle hook for the default app.
//
// It must be called before Run.
func OnNamed(stage Stage, name string, hook func(context.Context, *App) error) {
	DefaultApp.OnNamed(stage, name, hook)
}

// On registers a lifecycle hook.
//
// It must be called before Run.
func (a *App) On(stage Stage, hook func(context.Context, *App) error) {
	a.OnNamed(stage, "", hook)
}

// OnNamed registers a named lifecycle hook.
//
// Name is optional but useful for error messages.
//
// It must be called before Run.
func (a *App) OnNamed(stage Stage, name string, hook func(context.Context, *App) error) {
	if hook == nil {
		panic("app: nil hook")
	}

	if !validStage(stage) {
		panic(fmt.Sprintf("app: invalid stage %q", stage))
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("On")
	a.hooks[stage] = append(a.hooks[stage], namedCtxAppFunc{name: name, fn: hook})
}

func (a *App) runHooks(ctx context.Context, stage Stage) error {
	a.mu.Lock()
	hooks := append([]namedCtxAppFunc(nil), a.hooks[stage]...)
	a.mu.Unlock()

	var errs []error

	for i, hook := range hooks {
		if err := hook.fn(ctx, a); err != nil {
			wrapped := fmt.Errorf("app: hook %s: %w", hookLabel(stage, hook.name, i), err)

			// During shutdown stages, continue executing remaining hooks.
			if stage == StageStopping || stage == StageExited {
				errs = append(errs, wrapped)
				continue
			}

			return wrapped
		}
	}

	return errors.Join(errs...)
}
