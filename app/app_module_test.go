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
	"testing"
)

// testModule is a minimal [Module] implementation for testing.
type testModule struct {
	name    string
	init    func(context.Context, *App) error
	prepare func(context.Context, *App) error
	start   func(context.Context, *App) error
	stop    func(context.Context, *App) error
	wait    func(context.Context, *App) error
	close   func(context.Context, *App) error
}

func testModuleCallback(callback func(context.Context) error) func(context.Context, *App) error {
	return func(ctx context.Context, _ *App) error { return callback(ctx) }
}

func newTestModule(name string) *testModule {
	noop := func(context.Context, *App) error { return nil }
	return &testModule{name: name, init: noop, start: noop, close: noop}
}

func (m *testModule) Name() string                            { return m.name }
func (m *testModule) Init(ctx context.Context, a *App) error  { return m.init(ctx, a) }
func (m *testModule) Start(ctx context.Context, a *App) error { return m.start(ctx, a) }
func (m *testModule) Close(ctx context.Context, a *App) error { return m.close(ctx, a) }

func (m *testModule) Prepare(ctx context.Context, a *App) error {
	if m.prepare != nil {
		return m.prepare(ctx, a)
	}
	return nil
}

func (m *testModule) Stop(ctx context.Context, a *App) error {
	if m.stop != nil {
		return m.stop(ctx, a)
	}
	return nil
}

func (m *testModule) Wait(ctx context.Context, a *App) error {
	if m.wait != nil {
		return m.wait(ctx, a)
	}
	return nil
}

func TestUseValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		module Module
	}{
		{name: "nil"},
		{name: "unnamed", module: newTestModule("")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPanics(t, func() { New().Use(tc.module) })
		})
	}
}

type testPriorityModule struct {
	priority int
	Module
}

func newPriorityModule(priority int, mod Module) testPriorityModule {
	return testPriorityModule{priority: priority, Module: mod}
}

func (m testPriorityModule) Priority() int { return m.priority }

func TestWithDependenciesPreservesMetadataAndCopiesNames(t *testing.T) {
	assertPanics(t, func() { WithDependencies(nil, "database") })
	assertPanics(t, func() { WithDependencies(nil) })
	plain := newTestModule("plain")
	if WithDependencies(plain) != plain {
		t.Fatal("empty dependencies must preserve the original module")
	}
	original := dependencyModule{
		testModule:   newTestModule("api"),
		dependencies: []string{"database"},
		priority:     7,
	}
	names := []string{"worker"}
	wrapped := WithDependencies(WithDependencies(original, names...), "cache")
	names[0] = "changed"

	if got := wrapped.Name(); got != original.Name() {
		t.Fatalf("Name=%q, want %q", got, original.Name())
	}
	if got := wrapped.(interface{ Priority() int }).Priority(); got != 7 {
		t.Fatalf("Priority=%d, want 7", got)
	}
	if got := WithDependencies(plain, "database").(interface{ Priority() int }).
		Priority(); got != 1 {
		t.Fatalf("default Priority=%d, want 1", got)
	}

	metadata := wrapped.(interface{ Dependencies() []string })
	want := []string{"database", "worker", "cache"}
	got := metadata.Dependencies()
	if !slices.Equal(got, want) {
		t.Fatalf("Dependencies=%v, want %v", got, want)
	}
	for i := range got {
		got[i] = "changed"
	}
	if got := metadata.Dependencies(); !slices.Equal(got, want) {
		t.Fatalf("caller mutated Dependencies: %v", got)
	}
	if !slices.Equal(original.dependencies, []string{"database"}) {
		t.Fatalf("wrapper mutated original dependencies: %v", original.dependencies)
	}
}
