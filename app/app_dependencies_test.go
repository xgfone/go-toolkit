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
	"slices"
	"strings"
	"testing"
	"time"
)

type dependencyModule struct {
	*testModule
	dependencies []string
	priority     int
}

func (m dependencyModule) Dependencies() []string { return m.dependencies }
func (m dependencyModule) Priority() int          { return m.priority }

func TestDependencyValidationBeforeStartup(t *testing.T) {
	module := func(name string, dependencies ...string) Module {
		return dependencyModule{
			testModule:   newTestModule(name),
			dependencies: dependencies,
		}
	}

	for _, tc := range []struct {
		name    string
		mods    []Module
		message string
	}{
		{"missing", []Module{module("a", "missing")}, "invalid dependency"},
		{"self", []Module{module("a", "a")}, "invalid dependency"},
		{"duplicate dependency", []Module{module("a", "b", "b"), module("b")}, "invalid dependency"},
		{"cycle", []Module{module("a", "b"), module("b", "a")}, "cyclic"},
		{"duplicate name", []Module{module("a"), module("a")}, "duplicate module"},
		{"wrapped duplicate", []Module{WithDependencies(module("a", "b"), "b"), module("b")}, "invalid dependency"},
		{"nested duplicate", []Module{WithDependencies(WithDependencies(module("a"), "b"), "b"), module("b")}, "invalid dependency"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestRunApp()
			a.SetConfigLoader(func(context.Context) error {
				t.Error("configuration ran before validation")
				return nil
			})
			a.Use(tc.mods...)
			result := a.Run(context.Background())
			if result == nil || !strings.Contains(result.Error(), tc.message) {
				t.Fatalf("Run=%v", result)
			}
			if got := a.Wait(context.Background()); got != result {
				t.Fatalf("Wait=%v, Run=%v", got, result)
			}
		})
	}
}

func TestMetadataPanicPublishesValidationFailure(t *testing.T) {
	a := newTestRunApp()
	failure := errors.New("dependency metadata panic")
	a.Use(panicDependencyModule{newTestModule("module"), failure})
	a.SetConfigLoader(func(context.Context) error {
		t.Error("configuration ran with invalid metadata")
		return nil
	})

	result := a.Run(context.Background())
	if !errors.Is(result, failure) || !strings.Contains(result.Error(), "panic") {
		t.Error(result)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if a.Wait(ctx) != result {
		t.Error("validation panic left Wait blocked")
	}
}

func TestFailedUsersRetainOnlyTheirTransitiveProviders(t *testing.T) {
	for _, tc := range []struct {
		name             string
		deps             [][]int
		failed, retained []int
		taskTimeout      bool
	}{
		{"shared", [][]int{nil, {0}, {0}, {1}, {2}}, []int{3}, []int{0, 1}, false},
		{"diamond", [][]int{nil, {0}, {0}, {1, 2}, nil, {4}}, []int{3}, []int{0, 1, 2}, false},
		{"multiple failures", [][]int{nil, {0}, {0}, {1}, nil, {4}, nil}, []int{3, 5}, []int{0, 1, 4}, false},
		// A group timeout exhausts the shared budget, but independent branches
		// must still receive Stop rather than being abandoned with the providers.
		{"expired budget", [][]int{nil, nil, {0}, {1}, {2}}, []int{4}, []int{0, 2}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestRunApp()
			failure := errors.New("cannot prove exit")
			release := make(chan struct{})
			if tc.taskTimeout {
				a.SetShutdownTimeout(20 * time.Millisecond)
				failure = context.DeadlineExceeded
			}
			t.Cleanup(func() {
				close(release)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				for _, group := range a.groups {
					group.Stop()
					if err := group.Wait(ctx); err != nil {
						t.Error(err)
					}
				}
			})
			stopped := make([]bool, len(tc.deps))
			for i, edges := range tc.deps {
				name := fmt.Sprint(i)
				m := newTestModule(name)
				names := make([]string, len(edges))
				for j, dep := range edges {
					names[j] = fmt.Sprint(dep)
				}
				m.stop = func(context.Context, *App) error {
					stopped[i] = true
					return nil
				}
				if slices.Contains(tc.failed, i) {
					if tc.taskTimeout {
						m.start = func(context.Context, *App) error {
							a.TaskGroup(name).Go("stuck", func(context.Context) error { <-release; return nil })
							return nil
						}
					} else {
						m.wait = func(context.Context, *App) error { return failure }
					}
				}
				m.close = func(context.Context, *App) error {
					t.Error("Close during incomplete shutdown")
					return nil
				}
				a.Use(WithDependencies(m, names...))
			}
			a.On(StageReady, "stop", func(context.Context) error {
				a.Stop()
				return nil
			})
			result := a.Run(context.Background())
			if !errors.Is(result, failure) || !errors.Is(result, ErrShutdownIncomplete) {
				t.Fatal(result)
			}
			for i, got := range stopped {
				retained := slices.Contains(tc.retained, i)
				if got == retained {
					t.Errorf("module %d: stopped=%v retained=%v", i, got, retained)
				}
			}
		})
	}
}

type panicDependencyModule struct {
	*testModule
	failure error
}

func (m panicDependencyModule) Dependencies() []string { panic(m.failure) }
