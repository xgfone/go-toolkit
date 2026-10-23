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

// Module owns initialized resources and the business work that uses them.
// Init acquires resources; failed Init must release its partial acquisitions.
// Prepare builds business state after all Init calls succeed. Start launches
// activity and returns once ready; every Prepare completes before any Start.
//
// Stop closes admission and initiates shutdown promptly, without waiting for
// draining or releasing dependencies. Wait joins additional business activity
// outside the module's TaskGroup, including its final writes. It may return nil
// immediately when there is no additional business activity. The app stops and
// joins registered group tasks separately; both joins must succeed before
// providers stop or resources close. TaskGroup use is optional. Resource-internal
// maintenance,  such as pool workers, belongs to Close. Stop and Wait must tolerate
// Init succeeding without Start, or a partial Start. Close releases resources only
// after all business joins succeed. Panics in lifecycle methods are returned as
// errors and follow the same rollback rules. Shutdown methods should be idempotent.
// Never call App.Wait from a lifecycle method.
//
// An optional Dependencies() []string declares required module names.
// [WithDependencies] adds dependencies to an existing module. Missing,
// duplicate, self, and cyclic dependencies are rejected before initialization.
// Dependencies start first and stop last. Descending Priority() int (default 1)
// and registration order break ties within dependency batches. Every batch is
// stopped in reverse order: all Stop calls, then all Wait calls and task joins.
// Logger lifetime belongs to App.Finalize, after all resource and exit callbacks.
// Module names and metadata must remain stable once registered.
type Module interface {
	Name() string
	Init(context.Context, *App) error
	Prepare(context.Context, *App) error
	Start(context.Context, *App) error
	Stop(context.Context, *App) error
	Wait(context.Context, *App) error
	Close(context.Context, *App) error
}

// WithDependencies wraps m to add the required module names in dependencies.
// The module's existing dependencies come first, and its name, lifecycle methods,
// and optional Priority() int are preserved (the default priority is 1).
// Wrapping a module repeatedly adds dependencies; it does not replace them.
//
// The supplied slice is copied. Run validates the combined dependencies,
// including repeated names, before initialization. A nil m panics.
func WithDependencies(m Module, dependencies ...string) Module {
	if m == nil {
		panic("app: nil module")
	}
	return dependentModule{
		dependencies: slices.Clone(dependencies),

		Module: m,
	}
}

type dependentModule struct {
	dependencies []string
	Module
}

func (m dependentModule) Dependencies() []string {
	var dependencies []string
	if module, ok := m.Module.(interface{ Dependencies() []string }); ok {
		dependencies = slices.Clone(module.Dependencies())
	}
	return append(dependencies, m.dependencies...)
}

func (m dependentModule) Priority() int { return priority.Get(m.Module) }

type moduleRun struct {
	name         string
	module       Module
	dependencies []string
	initialized  bool
	tasks        *TaskGroup
}

// Use registers modules on Default before Run.
func Use(mods ...Module) {
	defaultApp.Use(mods...)
}

// Use registers modules before Run. Names must be nonempty and unique.
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

func (a *App) prepareModulesLocked(ctx context.Context) error {
	sortModules(a.modules)
	byName := make(map[string]*moduleRun, len(a.modules))
	var pending []*moduleRun
	for _, m := range a.modules {
		name := m.Name()
		if byName[name] != nil {
			return fmt.Errorf("app: duplicate module %q", name)
		}

		r := &moduleRun{module: m, name: name, tasks: newTaskGroup(a, name, ctx)}
		if deps, ok := m.(interface{ Dependencies() []string }); ok {
			r.dependencies = slices.Clone(deps.Dependencies())
		}

		byName[name] = r
		pending = append(pending, r)
	}

	for _, r := range pending {
		seen := make(map[string]bool)
		for _, name := range r.dependencies {
			if name == r.name || byName[name] == nil || seen[name] {
				return fmt.Errorf("app: invalid dependency %q of module %q", name, r.name)
			}
			seen[name] = true
		}
	}

	// Build the entire plan before publishing it or registering callbacks.
	completed := make(map[string]bool)
	var batches [][]*moduleRun
	var ordered []*moduleRun
	for len(pending) > 0 {
		var batch, remaining []*moduleRun
		for _, r := range pending {
			ready := true
			for _, name := range r.dependencies {
				ready = ready && completed[name]
			}

			if ready {
				batch = append(batch, r)
			} else {
				remaining = append(remaining, r)
			}
		}

		if len(batch) == 0 {
			return fmt.Errorf("app: cyclic module dependencies")
		}

		for _, r := range batch {
			completed[r.name] = true
		}

		batches = append(batches, batch)
		ordered = append(ordered, batch...)
		pending = remaining
	}

	a.moduleBatches = batches
	a.groups = make(map[string]*TaskGroup, len(ordered))
	for _, r := range ordered {
		a.groups[r.name] = r.tasks
		a.registerModuleLocked(r)
	}

	return nil
}

func (a *App) registerModuleLocked(r *moduleRun) {
	for _, entry := range []struct {
		phase lifecyclePhase
		name  string
		fn    func(context.Context, *App) error
	}{
		{phaseModuleInit, "init", r.module.Init},
		{phaseModulePrepare, "prepare", r.module.Prepare},
		{phaseModuleStart, "start", r.module.Start},
		{phaseModuleClose, "close", r.module.Close},
	} {
		a.registerCallbackLocked(entry.phase, "", func(ctx context.Context) error {
			if entry.phase == phaseModuleClose && !r.initialized {
				return nil
			}

			err := a.callModule(ctx, r, entry.name, entry.fn)
			if entry.phase == phaseModuleInit {
				r.initialized = err == nil
			}
			return err
		})
	}
}

func (a *App) callModule(ctx context.Context, r *moduleRun, name string, fn func(context.Context, *App) error) error {
	err := saferun(ctx, func(ctx context.Context) error {
		return fn(ctx, a)
	})
	if err != nil {
		return fmt.Errorf("app: %s module %q: %w", name, r.name, err)
	}
	return nil
}
