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
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLifecycleCallbackAndModuleOrder(t *testing.T) {
	a := newTestRunApp()
	type contextKey struct{}
	base := context.WithValue(context.Background(), contextKey{}, "parent")
	parent, cancel := context.WithTimeout(base, 2*time.Second)
	defer cancel()

	var runCtx context.Context
	shutdownContexts := make(map[string]context.Context)
	var events []string
	record := func(event string, shutdown bool) func(ctx context.Context) error {
		return func(ctx context.Context) error {
			events = append(events, event)
			if !shutdown {
				if runCtx == nil {
					runCtx = ctx
				}
				if ctx != runCtx || ctx.Value(contextKey{}) != "parent" {
					t.Errorf("%s lost the run context or parent values", event)
				}
				if deadline, ok := ctx.Deadline(); !ok {
					t.Errorf("%s lost the parent deadline", event)
				} else if want, _ := parent.Deadline(); deadline != want {
					t.Errorf("%s deadline = %v, want %v", event, deadline, want)
				}
			} else {
				group := "cleanup"
				if strings.HasPrefix(event, "stopping") ||
					strings.HasSuffix(event, ".Stop") ||
					strings.HasSuffix(event, ".Wait") {
					group = "drain"
				}
				if strings.HasPrefix(event, "exited") {
					group = "exited"
				}
				if shutdownContexts[group] == nil {
					shutdownContexts[group] = ctx
				}
				if ctx != shutdownContexts[group] || ctx == runCtx || ctx.Err() != nil {
					t.Errorf("%s did not receive the independent shutdown context", event)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Errorf("%s did not receive the shutdown deadline", event)
				}
				if runCtx.Err() != context.Canceled {
					t.Errorf("%s ran before the run context was canceled", event)
				}
				if a.Context().Err() != context.Canceled {
					t.Errorf("%s ran before lifecycle cancellation", event)
				}
			}

			select {
			case <-a.Done():
				t.Errorf("Done closed before %s finished", event)
			default:
			}
			return nil
		}
	}

	moduleCallback := func(event string, shutdown bool) func(context.Context, *App) error {
		callback := record(event, shutdown)
		return func(ctx context.Context, got *App) error {
			if got != a {
				t.Errorf("%s received a different App", event)
			}
			return callback(ctx)
		}
	}

	a.SetConfigLoader(record("config", false))
	a.On(StageInit, "", record("init.first", false))
	a.On(StageInit, "second", record("init.second", false))
	mod := func(name string) *testModule {
		return &testModule{name: name, init: moduleCallback(name+".Init", false),
			prepare: moduleCallback(name+".Prepare", false),
			start:   moduleCallback(name+".Start", false),
			stop:    moduleCallback(name+".Stop", true),
			wait:    moduleCallback(name+".Wait", true),
			close:   moduleCallback(name+".Close", true)}
	}
	// The wrappers must forward all six methods and priority. Dependency edges
	// override priority, while priority and registration order break batch ties.
	a.Use(WithDependencies(newPriorityModule(math.MinInt, mod("low")), "provider"),
		WithDependencies(newPriorityModule(math.MaxInt, mod("first")), "provider"))
	a.Use(WithDependencies(mod("default"), "provider"),
		WithDependencies(newPriorityModule(math.MaxInt, mod("second")), "provider"),
		newPriorityModule(math.MinInt, mod("provider")))
	// Use must not change the boundary between callbacks and module methods.
	a.On(StageInit, "third", record("init.third", false))
	a.On(StageStart, "start", record("start", false))
	a.On(StageReady, "ready", func(ctx context.Context) error {
		if err := record("ready", false)(ctx); err != nil {
			return err
		}
		a.Stop()
		return nil
	})
	a.On(StageStopping, "first", record("stopping.first", true))
	a.On(StageStopping, "second", record("stopping.second", true))
	a.Cleanup(func() error { events = append(events, "cleanup.simple"); return nil })
	a.On(StageCleanup, "first", record("cleanup.first", true))
	a.On(StageCleanup, "second", record("cleanup.second", true))
	a.On(StageExited, "first", record("exited.first", true))
	a.On(StageExited, "second", record("exited.second", true))
	if err := a.Run(parent); err != nil {
		t.Fatal(err)
	}
	want := []string{"config", "init.first", "init.second", "init.third",
		"provider.Init", "first.Init", "second.Init", "default.Init", "low.Init",
		"provider.Prepare", "first.Prepare", "second.Prepare", "default.Prepare", "low.Prepare", "start",
		"provider.Start", "first.Start", "second.Start", "default.Start", "low.Start", "ready",
		"stopping.first", "stopping.second",
		"low.Stop", "default.Stop", "second.Stop", "first.Stop",
		"low.Wait", "default.Wait", "second.Wait", "first.Wait",
		"provider.Stop", "provider.Wait",
		"low.Close", "default.Close", "second.Close", "first.Close", "provider.Close",
		"cleanup.second", "cleanup.first", "cleanup.simple", "exited.first", "exited.second"}
	if !slices.Equal(events, want) {
		t.Fatalf("lifecycle = %v, want %v", events, want)
	}
	if a.Context().Value(contextKey{}) != nil {
		t.Error("lifecycle context must not inherit parent values")
	}
	if _, ok := a.Context().Deadline(); ok {
		t.Error("lifecycle context must not inherit the parent deadline")
	}
}

func TestLifecycleStartupRollback(t *testing.T) {
	for _, tc := range []struct {
		failAt string
		label  string
		want   []string
	}{
		{
			failAt: "config", label: "startup failed",
			want: []string{"config", "stopping", "cleanup", "exited"},
		},
		{
			failAt: "init", label: `app: callback #0 at stage "init": startup failed`,
			want: []string{"config", "init", "stopping", "cleanup", "exited"},
		},
		{
			failAt: "second.Init", label: `app: init module "second": startup failed`,
			want: []string{"config", "init", "first.Init", "second.Init", "stopping", "first.Close", "cleanup", "exited"},
		},
		{
			failAt: "start", label: `app: callback "prepare" at stage "start": startup failed`,
			want: []string{"config", "init", "first.Init", "second.Init", "third.Init", "start",
				"stopping", "third.Close", "second.Close", "first.Close", "cleanup", "exited"},
		},
		{
			failAt: "second.Start", label: `app: start module "second": startup failed`,
			want: []string{"config", "init", "first.Init", "second.Init", "third.Init", "start", "first.Start", "second.Start",
				"stopping", "third.Close", "second.Close", "first.Close", "cleanup", "exited"},
		},
	} {
		t.Run(tc.failAt, func(t *testing.T) {
			a := newTestRunApp()
			failure := errors.New("startup failed")
			var events []string
			record := func(event string) func(ctx context.Context) error {
				return func(context.Context) error {
					events = append(events, event)
					if event == tc.failAt {
						return failure
					}
					return nil
				}
			}
			a.SetConfigLoader(record("config"))
			a.On(StageInit, "", record("init"))
			for _, name := range []string{"first", "second", "third"} {
				a.Use(&testModule{
					name:  name,
					init:  testModuleCallback(record(name + ".Init")),
					start: testModuleCallback(record(name + ".Start")),
					close: testModuleCallback(record(name + ".Close")),
				})
			}
			a.On(StageStart, "prepare", record("start"))
			a.On(StageReady, "", record("ready"))
			a.On(StageStopping, "", record("stopping"))
			a.On(StageCleanup, "", record("cleanup"))
			a.On(StageExited, "", record("exited"))

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := a.Run(ctx)
			if !errors.Is(err, failure) || err.Error() != tc.label {
				t.Fatalf("Run error = %v, want %s wrapping the startup failure", err, tc.label)
			}
			if !slices.Equal(events, tc.want) {
				t.Fatalf("lifecycle = %v, want %v", events, tc.want)
			}
			if got := a.Wait(context.Background()); got != err {
				t.Fatalf("Wait=%v, Run=%v", got, err)
			}
		})
	}
}

func TestLifecycleShutdownCollectsErrors(t *testing.T) {
	for _, mode := range []string{"error", "panic"} {
		t.Run(mode, func(t *testing.T) {
			a := newTestRunApp()
			var events []string
			errs := make(map[string]error)
			fail := func(event string) func(context.Context) error {
				errs[event] = errors.New(event)
				return func(context.Context) error {
					events = append(events, event)
					if mode == "panic" {
						panic(errs[event])
					}
					return errs[event]
				}
			}
			for _, name := range []string{"first", "second"} {
				m := newTestModule(name)
				m.stop = testModuleCallback(fail(name + ".Stop"))
				m.close = testModuleCallback(fail(name + ".Close"))
				a.Use(m)
			}

			a.On(StageReady, "ready", fail("ready"))
			a.On(StageStopping, "stopping.1", fail("stopping.1"))
			a.On(StageStopping, "stopping.2", fail("stopping.2"))
			a.On(StageCleanup, "cleanup.1", fail("cleanup.1"))
			a.On(StageCleanup, "cleanup.2", fail("cleanup.2"))

			simple := fail("cleanup.simple")
			a.Cleanup(func() error { return simple(context.Background()) })
			a.On(StageExited, "exited.1", fail("exited.1"))
			a.On(StageExited, "exited.2", fail("exited.2"))
			a.Finalize("logger", fail("logger"))

			err := a.Run(context.Background())
			for event, failure := range errs {
				if !errors.Is(err, failure) {
					t.Errorf("Run error = %v, missing %s", err, event)
				}
			}
			if errors.Is(err, ErrShutdownIncomplete) {
				t.Fatal("Stop errors must not skip cleanup when all joins succeed")
			}
			if got := a.Wait(context.Background()); got != err {
				t.Fatalf("Wait=%v, Run=%v", got, err)
			}

			message := ""
			if mode == "panic" {
				message = "panic: "
			}

			for _, label := range []string{
				`app: callback "ready" at stage "ready": ` + message + "ready",
				`app: close module "second": ` + message + "second.Close",
				`app: close module "first": ` + message + "first.Close",
				`app: callback "cleanup.2" at stage "cleanup": ` + message + "cleanup.2",
				`app: callback #2 at stage "cleanup": ` + message + "cleanup.simple",
			} {
				if err == nil || !strings.Contains(err.Error(), label) {
					t.Errorf("Run error = %v, missing label %s", err, label)
				}
			}

			want := []string{
				"ready", "stopping.1", "stopping.2", "second.Stop", "first.Stop",
				"second.Close", "first.Close", "cleanup.simple", "cleanup.2", "cleanup.1",
				"exited.1", "exited.2", "logger",
			}
			if !slices.Equal(events, want) {
				t.Fatalf("lifecycle = %v, want %v", events, want)
			}
		})
	}
}

func TestLifecycleFutureCallbackRegistration(t *testing.T) {
	a := newTestRunApp()
	var events []string
	record := func(event string) func(ctx context.Context) error {
		return func(context.Context) error {
			events = append(events, event)
			return nil
		}
	}
	m := newTestModule("dynamic")
	m.init = func(_ context.Context, a *App) error {
		a.On(StageStart, "", record("start"))
		a.On(StageCleanup, "", record("cleanup.init"))
		return nil
	}
	m.start = func(_ context.Context, a *App) error {
		// Module.Start is still within StageStart, even though its public callbacks
		// have already finished. Registering another Start callback must panic.
		assertPanics(t, func() {
			a.On(StageStart, "", record("late start"))
		})
		a.On(StageReady, "", func(_ context.Context) error {
			events = append(events, "ready")
			a.Stop()
			return nil
		})
		return nil
	}
	m.close = func(_ context.Context, a *App) error {
		a.On(StageCleanup, "", func(_ context.Context) error {
			events = append(events, "cleanup.stop")
			a.On(StageExited, "", record("exited"))
			return nil
		})
		return nil
	}
	a.Use(m)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := a.Run(ctx); err != nil {
		t.Fatal(err)
	}
	want := []string{"start", "ready", "cleanup.stop", "cleanup.init", "exited"}
	if !slices.Equal(events, want) {
		t.Fatalf("lifecycle = %v, want %v", events, want)
	}
}

func TestLifecycleShutdownTimeout(t *testing.T) {
	for _, expireAt := range []string{"stopping", "cleanup", "exited"} {
		t.Run(expireAt, func(t *testing.T) {
			a := newTestRunApp()
			a.SetShutdownTimeout(20 * time.Millisecond)
			a.SetCleanupTimeout(20 * time.Millisecond)
			a.SetExitedTimeout(20 * time.Millisecond)
			a.SetFinalizerTimeout(time.Second)
			contexts := make(map[string]context.Context)
			var events []string
			observe := func(event string, budget string) func(context.Context) error {
				return func(ctx context.Context) error {
					events = append(events, event)
					if contexts[budget] == nil {
						if ctx.Err() != nil || a.Context().Err() != context.Canceled {
							t.Errorf("%s must start with a fresh context", budget)
						}
						for other, seen := range contexts {
							if ctx == seen {
								t.Errorf("%s shares the %s budget", budget, other)
							}
						}
						contexts[budget] = ctx
					} else if ctx != contexts[budget] {
						t.Errorf("%s lost its phase context", event)
					}
					if event == expireAt {
						<-ctx.Done()
					}
					select {
					case <-a.Done():
						t.Errorf("Done closed before %s finished", event)
					default:
					}
					return ctx.Err()
				}
			}
			m := newTestModule("module")
			m.close = testModuleCallback(observe("module.Close", "cleanup"))
			a.Use(m)
			a.On(StageStopping, "stopping", observe("stopping", "drain"))
			a.Cleanup(func() error { events = append(events, "cleanup.simple"); return nil })
			a.On(StageCleanup, "cleanup", observe("cleanup", "cleanup"))
			a.On(StageExited, "exited", observe("exited", "exited"))
			a.Finalize("logger", observe("logger", "finalizer"))
			a.On(StageReady, "stop", func(context.Context) error { a.Stop(); return nil })
			err := a.Run(context.Background())
			if !errors.Is(err, context.DeadlineExceeded) ||
				errors.Is(err, ErrShutdownIncomplete) ||
				!strings.Contains(err.Error(), `callback "`+expireAt+`"`) {
				t.Fatalf("Run error = %v, want a named shutdown timeout", err)
			}

			want := []string{
				"stopping", "module.Close", "cleanup",
				"cleanup.simple", "exited", "logger",
			}
			if !slices.Equal(events, want) {
				t.Fatalf("shutdown = %v, want %v", events, want)
			}

			select {
			case <-a.Done():
			default:
				t.Fatal("Done did not close after timed-out shutdown")
			}
		})
	}
}
