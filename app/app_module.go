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
	"slices"

	"github.com/xgfone/go-toolkit/internal/priority"
)

// Module represents a lifecycle-managed component.
//
// Init and Start are executed in registration order.
// Stop is executed in reverse registration order.
type Module interface {
	Name() string
	Init(ctx context.Context, app *App) error
	Start(ctx context.Context, app *App) error
	Stop(ctx context.Context, app *App) error
}

// Use registers lifecycle modules for the default app.
//
// It must be called before Run.
func Use(mods ...Module) {
	DefaultApp.Use(mods...)
}

// Use registers lifecycle modules.
//
// It must be called before Run.
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
		return priority.Get(b) - priority.Get(a)
	})
}
