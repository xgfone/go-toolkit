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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTaskGroupValidationAndCompletion(t *testing.T) {
	a := newTestRunApp()
	assertPanics(t, func() { a.TaskGroup("worker") })
	a.Use(newTestModule("worker"))
	a.On(StageReady, "validate", func(context.Context) error {
		assertPanics(t, func() { a.TaskGroup("unknown") })
		g := a.TaskGroup("worker")
		assertPanics(t, func() { g.Go("nil", nil) })
		select {
		case <-g.Done():
			t.Error("idle group completed before admission was sealed")
		default:
		}

		g.Stop()
		g.Stop()
		select {
		case <-g.Done():
		default:
			t.Error("sealed idle group did not complete")
		}
		a.Stop()
		return nil
	})
	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestTaskGroupCompletionWinsCanceledWait(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for range 64 {
		g := newTaskGroup(newTestRunApp(), "worker", context.Background())
		ctx := completionContext{canceled, sync.OnceFunc(g.Stop)}
		if err := g.Wait(ctx); err != nil {
			t.Fatalf("completed group lost to canceled wait: %v", err)
		}
	}
}

func TestTaskGroupsKeepDownstreamAdmissionOpen(t *testing.T) {
	a := newTestRunApp()
	type key struct{}
	valuectx := context.WithValue(context.Background(), key{}, "value")
	parent, cancel := context.WithTimeout(valuectx, 2*time.Second)
	defer cancel()

	ingress := newTestModule("ingress")
	worker := newTestModule("worker")
	ingressExited := make(chan struct{})
	var downstream *TaskGroup
	var jobFinished atomic.Bool
	worker.start = func(context.Context, *App) error {
		downstream = a.TaskGroup("worker")
		if downstream.Context().Value(key{}) != "value" {
			t.Error("group lost parent values")
		}
		if _, ok := downstream.Context().Deadline(); ok {
			t.Error("group inherited the running deadline")
		}
		return nil
	}
	ingress.start = func(context.Context, *App) error {
		a.TaskGroup("ingress").Go("last-request", func(ctx context.Context) error {
			defer close(ingressExited)
			<-ctx.Done()
			if downstream.Context().Err() != nil {
				t.Error("worker canceled before ingress drained")
			}
			downstream.Go("last-job", func(context.Context) error {
				jobFinished.Store(true)
				return nil
			})
			return nil
		})
		return nil
	}
	worker.stop = func(context.Context, *App) error {
		select {
		case <-ingressExited:
		default:
			t.Error("worker stopped before ingress joined")
		}
		return nil
	}
	worker.close = func(context.Context, *App) error {
		if !jobFinished.Load() {
			t.Error("resource closed before final job")
		}
		return nil
	}

	a.Use(WithDependencies(ingress, "worker"), worker)
	a.On(StageReady, "stop", func(context.Context) error {
		a.Stop()
		return nil
	})

	if err := a.Run(parent); err != nil {
		t.Fatal(err)
	}
	if !jobFinished.Load() {
		t.Fatal("final job did not finish")
	}
}

func TestTaskGroupAdmissionRacesStop(t *testing.T) {
	for range 50 {
		a := newTestRunApp()
		m := newTestModule("worker")
		a.Use(m)

		var calls sync.WaitGroup
		var accepted, completed atomic.Int32
		a.On(StageReady, "race", func(context.Context) error {
			g := a.TaskGroup("worker")
			for range 24 {
				calls.Go(func() {
					defer func() {
						if r := recover(); r != nil && r != "app: task group is stopping" {
							t.Errorf("panic: %v", r)
						}
					}()

					g.Go("race", func(ctx context.Context) error {
						<-ctx.Done()
						completed.Add(1)
						return nil
					})
					accepted.Add(1)
				})
			}
			a.Stop()
			return nil
		})

		if err := a.Run(context.Background()); err != nil {
			t.Fatal(err)
		}

		calls.Wait()
		if accepted.Load() != completed.Load() {
			t.Fatal("admitted task escaped join")
		}
	}
}

func TestModuleTaskFailureAndPanicAreFinalResults(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(fmt.Sprint(panics), func(t *testing.T) {
			a := newTestRunApp()
			m := newTestModule("worker")
			failure := errors.New("owned task failed")
			m.start = func(context.Context, *App) error {
				a.TaskGroup("worker").Go("owned", func(context.Context) error {
					if panics {
						panic(failure)
					}
					return failure
				})
				return nil
			}
			a.Use(m)

			parent, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			result := a.Run(parent)
			if !errors.Is(result, failure) ||
				!strings.Contains(result.Error(), `module "worker" background task "owned"`) {
				t.Fatal(result)
			}
			if parent.Err() != nil {
				t.Fatal("task error did not trigger stop")
			}
			if got := a.Wait(context.Background()); got != result {
				t.Fatal("Wait lost task result")
			}
		})
	}
}

type emptyTaskError struct{ error }

func (*emptyTaskError) Unwrap() []error { return nil }

func TestTaskCancellationErrorClassification(t *testing.T) {
	failure := errors.New("final write failed")
	noCauses := &emptyTaskError{failure}
	for _, tc := range []struct {
		name   string
		result error
		want   error
	}{
		{name: "canceled", result: context.Canceled},
		{name: "wrapped", result: fmt.Errorf("stopped: %w", context.Canceled)},
		{name: "joined", result: errors.Join(context.Canceled, context.Canceled)},
		{name: "joined failure", result: errors.Join(context.Canceled, failure), want: failure},
		{name: "empty causes", result: noCauses, want: noCauses},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestRunApp()
			m := newTestModule("worker")
			m.start = func(context.Context, *App) error {
				a.TaskGroup("worker").Go("flush", func(ctx context.Context) error {
					<-ctx.Done()
					return tc.result
				})
				return nil
			}
			a.Use(m)
			a.On(StageReady, "stop", func(context.Context) error {
				a.Stop()
				return nil
			})
			if err := a.Run(context.Background()); !errors.Is(err, tc.want) {
				t.Fatalf("Run = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCancellationPanicRemainsFailure(t *testing.T) {
	a := newTestRunApp()
	m := newTestModule("worker")
	m.start = func(context.Context, *App) error {
		a.TaskGroup("worker").Go("panic", func(ctx context.Context) error {
			<-ctx.Done()
			panic(ctx.Err())
		})
		return nil
	}
	a.Use(m)

	a.On(StageReady, "stop", func(context.Context) error {
		a.Stop()
		return nil
	})

	if err := a.Run(context.Background()); err == nil ||
		!strings.Contains(err.Error(), "panic: context canceled") {
		t.Fatal(err)
	}
}

func TestRetainedTaskGroupsRemainUsableAfterIncompleteRun(t *testing.T) {
	a := newTestRunApp()
	a.SetShutdownTimeout(20 * time.Millisecond)
	provider := newTestModule("provider")
	user := newTestModule("user")
	release := make(chan struct{})
	releaseTask := sync.OnceFunc(func() { close(release) })
	var users, providers *TaskGroup
	var submitted atomic.Bool
	t.Cleanup(func() {
		releaseTask()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if users != nil {
			users.Stop()
			_ = users.Wait(ctx)
		}
		if providers != nil {
			providers.Stop()
			_ = providers.Wait(ctx)
		}
	})

	provider.start = func(context.Context, *App) error {
		providers = a.TaskGroup("provider")
		return nil
	}
	user.start = func(context.Context, *App) error {
		users = a.TaskGroup("user")
		users.Go("late-request", func(context.Context) error {
			<-release
			// The app retained this dependency so a still-live request can
			// finish its downstream work even after Run reports incomplete.
			a.TaskGroup("provider").Go("final-job", func(context.Context) error {
				submitted.Store(true)
				return nil
			})
			return nil
		})
		return nil
	}

	a.Use(provider, dependencyModule{testModule: user, dependencies: []string{"provider"}})
	a.On(StageReady, "stop", func(context.Context) error { a.Stop(); return nil })

	result := a.Run(context.Background())
	if !errors.Is(result, ErrShutdownIncomplete) {
		t.Fatal(result)
	}

	releaseTask()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := users.Wait(ctx); err != nil {
		t.Fatal(err)
	}

	providers.Stop()
	if err := providers.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if !submitted.Load() {
		t.Fatal("retained dependency could not receive final work")
	}
	if a.Wait(context.Background()) != result {
		t.Fatal("late work changed Run's frozen result")
	}
}
