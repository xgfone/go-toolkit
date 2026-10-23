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
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestModuleShutdownOrderAndResourceSafety(t *testing.T) {
	a := newTestRunApp()
	workerExited := make(chan struct{})
	servicesStopped := make(chan struct{})
	var events []string
	a.On(StageInit, "logger", func(context.Context) error {
		events = append(events, "logger.init")
		// Registering the logger first must still close it after later Exited callbacks.
		a.Finalize("logger", func(ctx context.Context) error {
			if ctx.Err() != nil {
				t.Error("logger received an expired context")
			}
			events = append(events, "logger.close")
			return nil
		})
		return nil
	})

	for _, name := range []string{"first", "second"} {
		m := newTestModule(name)
		m.init = func(context.Context, *App) error {
			events = append(events, name+".init")
			a.On(StageExited, name, func(context.Context) error {
				events = append(events, name+".exited")
				return nil
			})
			return nil
		}
		m.close = func(context.Context, *App) error {
			select {
			case <-workerExited:
			default:
				t.Error("resource closed before the managed worker exited")
			}
			events = append(events, name+".close")
			return nil
		}
		m.stop = func(context.Context, *App) error {
			events = append(events, name+".stop")
			if name == "first" {
				close(servicesStopped)
			}
			return nil
		}
		m.wait = func(context.Context, *App) error {
			events = append(events, name+".wait")
			return nil
		}
		a.Use(WithDependencies(m, "database"))
	}

	db := newTestModule("database")
	db.init = func(context.Context, *App) error {
		events = append(events, "database.init")
		return nil
	}
	db.stop = func(context.Context, *App) error {
		select {
		case <-workerExited:
		default:
			t.Error("provider batch stopped before the app-level worker joined")
		}
		events = append(events, "database.stop")
		return nil
	}
	db.close = func(context.Context, *App) error {
		events = append(events, "database.close")
		return nil
	}
	a.Use(db)

	a.On(StageInit, "worker", func(context.Context) error {
		a.Go("worker", func(ctx context.Context) error {
			defer close(workerExited)
			<-ctx.Done()
			<-servicesStopped
			return nil
		})
		return nil
	})

	a.Cleanup(func() error {
		events = append(events, "cleanup")
		return nil
	})
	a.On(StageReady, "stop", func(context.Context) error {
		a.Stop()
		return nil
	})

	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"logger.init", "database.init", "first.init", "second.init",
		"second.stop", "first.stop", "second.wait", "first.wait",
		"database.stop", "second.close", "first.close", "database.close",
		"cleanup", "first.exited", "second.exited", "logger.close",
	}
	if !slices.Equal(events, want) {
		t.Fatalf("lifecycle = %v, want %v", events, want)
	}
}

func TestShutdownTimeoutRetainsResources(t *testing.T) {
	for _, source := range []string{
		"app task without modules",
		"app task",
		"module task",
		"module wait",
	} {
		t.Run(source, func(t *testing.T) {
			a := newTestRunApp()
			a.SetShutdownTimeout(20 * time.Millisecond)
			release := make(chan struct{})
			exited := make(chan struct{})
			releaseWork := sync.OnceFunc(func() { close(release) })
			t.Cleanup(func() {
				releaseWork()
				select {
				case <-exited:
				case <-time.After(time.Second):
					t.Error("work did not exit during test cleanup")
				}
			})
			work := func(context.Context) error {
				defer close(exited)
				<-release
				return nil
			}
			m := newTestModule("server")
			db := newTestModule("database")
			m.start = func(context.Context, *App) error {
				switch source {
				case "app task", "app task without modules":
					a.Go("uncooperative", work)
				case "module task":
					a.TaskGroup("server").Go("uncooperative", work)
				case "module wait":
					go work(context.Background()) // Joined explicitly by Module.Wait.
				}
				return nil
			}
			if source == "module wait" {
				m.wait = func(ctx context.Context, _ *App) error {
					select {
					case <-exited:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			db.stop = func(context.Context, *App) error {
				t.Error("provider stopped with live users")
				return nil
			}
			for _, module := range []*testModule{m, db} {
				module.close = func(context.Context, *App) error {
					t.Error("resource closed with live work")
					return nil
				}
			}

			if source == "app task without modules" {
				a.On(StageStart, "worker", func(ctx context.Context) error {
					return m.start(ctx, a)
				})
			} else {
				a.Use(db, WithDependencies(m, "database"))
			}
			for _, stage := range []Stage{StageCleanup, StageExited} {
				a.On(stage, "retain", func(context.Context) error {
					t.Errorf("%s ran with live work", stage)
					return nil
				})
			}

			a.Finalize("logger", func(context.Context) error {
				t.Error("logger closed with live work")
				return nil
			})
			a.On(StageReady, "stop", func(context.Context) error {
				a.Stop()
				return nil
			})

			result := a.Run(context.Background())
			if !errors.Is(result, ErrShutdownIncomplete) ||
				!errors.Is(result, context.DeadlineExceeded) {
				t.Fatalf("Run=%v, want incomplete shutdown and timeout", result)
			}

			select {
			case <-a.Done():
			default:
				t.Fatal("Done did not close after the shutdown attempt")
			}
			if got := a.Wait(context.Background()); got != result {
				t.Fatalf("Wait=%v, Run=%v", got, result)
			}
			select {
			case <-exited:
				t.Fatal("uncooperative work exited before release")
			default:
			}
			releaseWork()
			select {
			case <-exited:
			case <-time.After(time.Second):
				t.Fatal("work did not exit after release")
			}
			assertPanics(t, func() {
				a.Finalize("late", func(context.Context) error { return nil })
			})
		})
	}
}

func TestRunDeadlineStartsIndependentShutdownBudgets(t *testing.T) {
	a := newTestRunApp()
	type key struct{}
	valuectx := context.WithValue(context.Background(), key{}, "parent")
	parent, cancel := context.WithTimeout(valuectx, 20*time.Millisecond)
	defer cancel()

	a.On(StageInit, "wait", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	check := func(ctx context.Context) error {
		if ctx.Err() != nil || ctx.Value(key{}) != "parent" {
			t.Error("shutdown lost values or inherited run cancellation")
		}
		deadline, ok := ctx.Deadline()
		runDeadline, _ := parent.Deadline()
		if !ok || !deadline.After(runDeadline) {
			t.Error("shutdown inherited the expired run deadline")
		}
		return nil
	}

	for _, stage := range []Stage{StageStopping, StageCleanup, StageExited} {
		a.On(stage, "context", check)
	}

	a.Finalize("logger", check)
	if err := a.Run(parent); !errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, ErrShutdownIncomplete) {
		t.Fatalf("Run = %v", err)
	}
}

func TestFinalizeValidationAndErrors(t *testing.T) {
	a := newTestRunApp()
	assertPanics(t, func() { a.Finalize("nil", nil) })
	failure := errors.New("logger flush failed")
	a.Finalize("logger", func(context.Context) error {
		select {
		case <-a.Done():
			t.Error("Done closed before logger finalization")
		default:
		}

		assertPanics(t, func() {
			a.Finalize("late", func(context.Context) error { return nil })
		})
		return failure
	})

	assertPanics(t, func() {
		a.Finalize("duplicate", func(context.Context) error { return nil })
	})

	a.On(StageReady, "stop", func(context.Context) error {
		a.Stop()
		return nil
	})

	if err := a.Run(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("Run lost finalizer error: %v", err)
	}
}

func TestWaitPanicRetainsDependencyAndJoinsOtherTasks(t *testing.T) {
	a := newTestRunApp()
	failure := errors.New("wait panic")
	provider := newTestModule("provider")
	user := newTestModule("user")
	var exited atomic.Bool
	user.start = func(context.Context, *App) error {
		a.TaskGroup("user").Go("worker", func(ctx context.Context) error {
			<-ctx.Done()
			exited.Store(true)
			return nil
		})
		return nil
	}
	user.wait = func(context.Context, *App) error {
		panic(failure)
	}
	provider.stop = func(context.Context, *App) error {
		t.Error("provider stopped without proof of exit")
		return nil
	}

	a.Use(provider, dependencyModule{
		testModule:   user,
		dependencies: []string{"provider"},
	})
	a.Finalize("logger", func(context.Context) error {
		t.Error("logger closed without proof of exit")
		return nil
	})
	a.On(StageReady, "stop", func(context.Context) error {
		a.Stop()
		return nil
	})

	result := a.Run(context.Background())
	if !errors.Is(result, failure) || !errors.Is(result, ErrShutdownIncomplete) {
		t.Error(result)
	}
	if !exited.Load() {
		// Ensure the pre-fix failure does not leave the test task behind.
		group := a.groups["user"]
		group.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		_ = group.Wait(ctx)
		t.Error("module Wait panic bypassed task-group join")
	}
}
