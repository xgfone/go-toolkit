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
	"testing"
)

// testModule is a minimal [Module] implementation for testing.
type testModule struct {
	name              string
	init, start, stop func(context.Context, *App) error
}

func testModuleCallback(callback func(context.Context) error) func(context.Context, *App) error {
	return func(ctx context.Context, _ *App) error { return callback(ctx) }
}

func newTestModule(name string) *testModule {
	noop := func(context.Context, *App) error { return nil }
	return &testModule{name: name, init: noop, start: noop, stop: noop}
}

func (m *testModule) Name() string                            { return m.name }
func (m *testModule) Init(ctx context.Context, a *App) error  { return m.init(ctx, a) }
func (m *testModule) Start(ctx context.Context, a *App) error { return m.start(ctx, a) }
func (m *testModule) Stop(ctx context.Context, a *App) error  { return m.stop(ctx, a) }

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
