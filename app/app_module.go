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
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/xgfone/go-toolkit/internal/priority"
)

// Module represents a lifecycle-managed component.
//
// Modules are ordered by descending Priority() int, if implemented,
// with a default priority of 1. Equal priorities retain registration order.
// [Module.Init] and [Module.Start] follow that order; [Module.Stop] reverses it.
// Only modules whose Init returned nil are stopped, even if Start was not called
// or returned an error. A failed Init must arrange to release partially acquired
// resources.
//
// Stop is called before waiting for managed background tasks. A resource needed
// by those tasks can instead be released through [App.On] at [StageCleanup]
// after successful initialization. Such a module's Stop must leave that resource open.
type Module interface {
	Name() string
	Init(ctx context.Context, app *App) error
	Start(ctx context.Context, app *App) error
	Stop(ctx context.Context, app *App) error
}

// Use registers lifecycle modules for the default app.
//
// It must be called before [Run].
func Use(mods ...Module) {
	defaultApp.Use(mods...)
}

// Use registers lifecycle modules.
//
// It must be called before [App.Run].
func (a *App) Use(mods ...Module) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("Use")

	for _, m := range mods {
		if m == nil {
			panic("app: nil module")
		}

		if m.Name() == "" {
			panic("app: empty module name")
		}

		a.modules = append(a.modules, m)
	}
}

func sortModules(mods []Module) {
	slices.SortStableFunc(mods, func(a, b Module) int {
		return cmp.Compare(priority.Get(b), priority.Get(a))
	})
}

func (a *App) registerModuleLocked(m Module) {
	initialized := false
	for _, entry := range []struct {
		phase    lifecyclePhase
		method   string
		callback func(context.Context, *App) error
	}{
		{phaseModuleInit, "init", m.Init},
		{phaseModuleStart, "start", m.Start},
		{phaseModuleStop, "stop", m.Stop},
	} {
		a.registerCallbackLocked(entry.phase, "", func(ctx context.Context) error {
			if entry.phase == phaseModuleStop && !initialized {
				return nil
			}

			err := entry.callback(ctx, a)
			if entry.phase == phaseModuleInit {
				initialized = err == nil
			}
			if err != nil {
				return fmt.Errorf("app: %s module %q: %w", entry.method, m.Name(), err)
			}
			return nil
		})
	}
}
