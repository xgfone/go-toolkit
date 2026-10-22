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
	"sync/atomic"
	"testing"
	"time"
)

func TestLifecycleCallbackAndModuleOrder(t *testing.T) {
	a := newTestRunApp()
	type contextKey struct{}
	parent, cancel := context.WithTimeout(context.WithValue(context.Background(), contextKey{}, "parent"), 2*time.Second)
	defer cancel()

	var runCtx, shutdownCtx context.Context
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
				if shutdownCtx == nil {
					shutdownCtx = ctx
				}
				if ctx != shutdownCtx || ctx == runCtx || ctx.Err() != nil {
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
			start: moduleCallback(name+".Start", false), stop: moduleCallback(name+".Stop", true)}
	}
	a.Use(newPriorityModule(math.MinInt, mod("low")), newPriorityModule(math.MaxInt, mod("first")))
	a.Use(mod("default"), newPriorityModule(math.MaxInt, mod("second")))
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
		"first.Init", "second.Init", "default.Init", "low.Init", "start",
		"first.Start", "second.Start", "default.Start", "low.Start", "ready",
		"stopping.first", "stopping.second", "low.Stop", "default.Stop", "second.Stop", "first.Stop",
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
			want: []string{"config", "init", "first.Init", "second.Init", "stopping", "first.Stop", "cleanup", "exited"},
		},
		{
			failAt: "start", label: `app: callback "prepare" at stage "start": startup failed`,
			want: []string{"config", "init", "first.Init", "second.Init", "third.Init", "start",
				"stopping", "third.Stop", "second.Stop", "first.Stop", "cleanup", "exited"},
		},
		{
			failAt: "second.Start", label: `app: start module "second": startup failed`,
			want: []string{"config", "init", "first.Init", "second.Init", "third.Init", "start", "first.Start", "second.Start",
				"stopping", "third.Stop", "second.Stop", "first.Stop", "cleanup", "exited"},
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
					name: name, init: testModuleCallback(record(name + ".Init")),
					start: testModuleCallback(record(name + ".Start")), stop: testModuleCallback(record(name + ".Stop")),
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
		})
	}
}

func TestLifecycleShutdownCollectsErrors(t *testing.T) {
	a := newTestRunApp()
	var events []string
	errs := make(map[string]error)
	fail := func(event string) func(ctx context.Context) error {
		errs[event] = errors.New(event)
		return func(context.Context) error {
			events = append(events, event)
			return errs[event]
		}
	}
	for _, name := range []string{"first", "second"} {
		m := newTestModule(name)
		m.stop = testModuleCallback(fail(name + ".Stop"))

		a.Use(m)
	}
	a.On(StageReady, "ready", fail("ready"))
	a.On(StageStopping, "stopping.1", fail("stopping.1"))
	a.On(StageStopping, "stopping.2", fail("stopping.2"))
	a.On(StageCleanup, "cleanup.1", fail("cleanup.1"))
	a.On(StageCleanup, "cleanup.2", fail("cleanup.2"))
	errs["cleanup.simple"] = errors.New("cleanup.simple")
	a.Cleanup(func() error {
		events = append(events, "cleanup.simple")
		return errs["cleanup.simple"]
	})
	a.On(StageExited, "exited.1", fail("exited.1"))
	a.On(StageExited, "exited.2", fail("exited.2"))

	err := a.Run(context.Background())
	for event, failure := range errs {
		if !errors.Is(err, failure) {
			t.Errorf("Run error = %v, missing %s", err, event)
		}
	}
	for _, label := range []string{
		`app: callback "ready" at stage "ready": ready`,
		`app: stop module "second": second.Stop`,
		`app: stop module "first": first.Stop`,
		`app: callback "cleanup.2" at stage "cleanup": cleanup.2`,
		`app: callback #2 at stage "cleanup": cleanup.simple`,
	} {
		if err == nil || !strings.Contains(err.Error(), label) {
			t.Errorf("Run error = %v, missing label %s", err, label)
		}
	}
	want := []string{"ready", "stopping.1", "stopping.2", "second.Stop", "first.Stop",
		"cleanup.simple", "cleanup.2", "cleanup.1", "exited.1", "exited.2"}
	if !slices.Equal(events, want) {
		t.Fatalf("lifecycle = %v, want %v", events, want)
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
	m.stop = func(_ context.Context, a *App) error {
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
	for _, expireAt := range []string{"stopping", "cleanup.second"} {
		t.Run(expireAt, func(t *testing.T) {
			a := newTestRunApp()
			a.SetShutdownTimeout(20 * time.Millisecond)
			var shutdownCtx context.Context
			var events []string
			expired := false
			observe := func(event string) func(context.Context) error {
				return func(ctx context.Context) error {
					events = append(events, event)
					if shutdownCtx == nil {
						shutdownCtx = ctx
						if ctx.Err() != nil || a.Context().Err() != context.Canceled {
							t.Error("shutdown must start with an independent uncanceled context")
						}
					}
					if ctx != shutdownCtx {
						t.Errorf("%s did not receive the shared shutdown context", event)
					}
					if event == expireAt {
						<-ctx.Done()
						expired = true
					}
					if expired && ctx.Err() != context.DeadlineExceeded {
						t.Errorf("%s did not receive the expired shutdown context", event)
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
			m.stop = testModuleCallback(observe("module.Stop"))
			a.Use(m)
			a.On(StageStopping, "stopping", observe("stopping"))
			a.On(StageCleanup, "cleanup.first", observe("cleanup.first"))
			a.On(StageCleanup, "cleanup.second", observe("cleanup.second"))
			a.Cleanup(func() error { events = append(events, "cleanup.simple"); return nil })
			a.On(StageExited, "exited", observe("exited"))
			a.On(StageReady, "stop", func(context.Context) error { a.Stop(); return nil })
			err := a.Run(context.Background())
			if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), `callback "`+expireAt+`"`) {
				t.Fatalf("Run error = %v, want a named shutdown timeout", err)
			}
			want := []string{"stopping", "module.Stop", "cleanup.simple", "cleanup.second", "cleanup.first", "exited"}
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

func TestLifecycleResourceCleanupWaitsForInitWorker(t *testing.T) {
	a := newTestRunApp()
	a.SetShutdownTimeout(time.Second)
	moduleStopped := make(chan struct{})
	workerExited := make(chan struct{})
	var resourceClosed, workerSawClosedResource atomic.Bool
	a.On(StageInit, "", func(_ context.Context) error {
		a.Go("init-worker", func(ctx context.Context) error {
			defer close(workerExited)
			<-ctx.Done()
			<-moduleStopped
			// Final work may still need the resource after Module.Stop returns.
			workerSawClosedResource.Store(resourceClosed.Load())
			return nil
		})
		return nil
	})
	m := newTestModule("resource")
	m.init = func(_ context.Context, a *App) error {
		a.On(StageCleanup, "", func(context.Context) error {
			select {
			case <-workerExited:
			default:
				t.Error("resource cleanup ran before the managed worker exited")
			}
			resourceClosed.Store(true)
			return nil
		})
		return nil
	}
	m.stop = func(context.Context, *App) error { close(moduleStopped); return nil }
	a.Use(m)
	a.On(StageReady, "", func(_ context.Context) error { a.Stop(); return nil })
	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if workerSawClosedResource.Load() || !resourceClosed.Load() {
		t.Fatal("resource must stay open during worker finalization and close during cleanup")
	}
}

func TestLifecycleStoppingJoinsWorkerBeforeModuleStop(t *testing.T) {
	a := newTestRunApp()
	a.SetShutdownTimeout(time.Second)
	workerExited := make(chan struct{})
	var resourceClosed, workerSawClosedResource atomic.Bool
	a.On(StageInit, "", func(_ context.Context) error {
		a.Go("init-worker", func(ctx context.Context) error {
			defer close(workerExited)
			<-ctx.Done()
			workerSawClosedResource.Store(resourceClosed.Load())
			return nil
		})
		// Joining a task during Stopping also works with existing modules that
		// release resources in Stop, without changing their public contract.
		a.On(StageStopping, "", func(ctx context.Context) error {
			select {
			case <-workerExited:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		return nil
	})
	m := newTestModule("resource")
	m.stop = func(context.Context, *App) error {
		select {
		case <-workerExited:
		default:
			t.Error("Module.Stop ran before the worker exited")
		}
		resourceClosed.Store(true)
		return nil
	}
	a.Use(m)
	a.On(StageReady, "", func(_ context.Context) error { a.Stop(); return nil })
	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if workerSawClosedResource.Load() || !resourceClosed.Load() {
		t.Fatal("resource must stay open until the stopping callback joins the worker")
	}
}
