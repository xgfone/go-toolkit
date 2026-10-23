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

func TestGoValidation(t *testing.T) {
	assertPanics(t, func() { New().Go("", nil) })
	assertPanics(t, func() { New().Go("", func(context.Context) error { return nil }) })
}

func TestReportErrorLifecycle(t *testing.T) {
	a := newTestRunApp()
	ignored := errors.New("reported outside Run")
	failure := errors.New("asynchronous failure")
	shutdownFailure := errors.New("failure during cleanup")
	a.ReportError(nil)
	a.ReportError(ignored)
	if a.Context().Err() != nil {
		t.Fatal("reporting before Run canceled the app")
	}

	a.On(StageReady, "report", func(ctx context.Context) error {
		a.ReportError(nil)
		if ctx.Err() != nil {
			t.Error("reporting nil stopped the app")
		}
		go a.ReportError(failure)
		<-ctx.Done()
		return nil
	})
	a.Cleanup(func() error {
		a.ReportError(shutdownFailure)
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := a.Run(ctx)
	if !errors.Is(result, failure) || !errors.Is(result, shutdownFailure) ||
		errors.Is(result, ignored) {
		t.Fatalf("Run lost reported failures or kept an ignored error: %v", result)
	}
	if ctx.Err() != nil {
		t.Fatal("ReportError did not trigger shutdown")
	}

	a.ReportError(ignored)
	a.ReportError(nil)
	if got := a.Wait(context.Background()); got != result {
		t.Fatalf("late ReportError changed the final result: %v", got)
	}
}

func TestGoCompletionAfterCancellation(t *testing.T) {
	for _, result := range []error{nil, errors.New("late task error")} {
		name := "success"
		if result != nil {
			name = "late error preserved"
		}
		t.Run(name, func(t *testing.T) {
			a := newTestRunApp()
			completed := false
			a.On(StageStart, "worker", func(ctx context.Context) error {
				a.Go("worker", func(taskCtx context.Context) error {
					if taskCtx != ctx {
						t.Error("managed task did not receive the run context")
					}
					<-taskCtx.Done()
					completed = true
					return result
				})
				return nil
			})
			a.On(StageReady, "stop", func(context.Context) error { a.Stop(); return nil })
			if err := a.Run(context.Background()); !errors.Is(err, result) {
				t.Fatalf("Run = %v, want %v", err, result)
			}
			if !completed {
				t.Fatal("Run returned before the managed task finished")
			}
		})
	}
}

func TestGoFailureStopsStartup(t *testing.T) {
	failure := errors.New("task failure")
	panicFailure := errors.New("panic failure")
	original := defaultApp
	t.Cleanup(func() { defaultApp = original })
	for _, tc := range []struct {
		name, taskName string
		packageGo      bool
		callback       func(context.Context) error
		wantErr        error
		message        string
	}{
		{name: "unnamed", callback: func(context.Context) error { return failure }, wantErr: failure, message: "task failure"},
		{name: "named", taskName: "worker", callback: func(context.Context) error { return failure }, wantErr: failure, message: "task failure"},
		{name: "package", taskName: "worker", packageGo: true, callback: func(context.Context) error { return failure }, wantErr: failure, message: "task failure"},
		{name: "panic error", taskName: "worker", callback: func(context.Context) error { panic(panicFailure) }, wantErr: panicFailure, message: "panic: panic failure"},
		{name: "panic value", callback: func(context.Context) error { panic("panic failure") }, message: "panic: panic failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestRunApp()
			start := a.Go
			if tc.packageGo {
				defaultApp = a
				start = Go
			}
			a.On(StageStart, "start-worker", func(ctx context.Context) error {
				start(tc.taskName, tc.callback)
				<-ctx.Done()
				return ctx.Err()
			})
			a.On(StageReady, "unexpected", func(context.Context) error {
				t.Error("Ready ran after the managed task failed")
				return nil
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := a.Run(ctx)
			label := fmt.Sprintf("background task %q: %s", tc.taskName, tc.message)
			if err == nil || !strings.Contains(err.Error(), label) {
				t.Fatalf("Run error = %v, want %s", err, label)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Run error = %v, want wrapped %v", err, tc.wantErr)
			}
			if ctx.Err() != nil {
				t.Fatal("task failure stopped the app only after the parent timed out")
			}
		})
	}
}

func TestConcurrentGoDuringShutdown(t *testing.T) {
	const attempts = 32
	for iteration := range 100 {
		a := newTestRunApp()
		start := make(chan struct{})
		firstRegistered := make(chan struct{})
		stopping := make(chan struct{})
		var registrations sync.WaitGroup
		var accepted, completedAtCleanup [attempts]bool
		var completed [attempts]atomic.Bool
		for i := range attempts {
			registrations.Go(func() {
				if i == 0 {
					defer close(firstRegistered)
				}
				if i == attempts-1 {
					<-stopping
				} else {
					<-start
				}

				defer func() {
					if r := recover(); r != nil && r != "app: Go can only be called while app is running" {
						t.Errorf("unexpected task registration panic: %v", r)
					}
				}()

				a.Go("concurrent", func(ctx context.Context) error {
					<-ctx.Done()
					completed[i].Store(true)
					return nil
				})
				accepted[i] = true
			})
		}

		a.On(StageReady, "stop", func(ctx context.Context) error {
			close(start)
			// Admit at least one task before Stop; the remaining callers race
			// with shutdown closing task registration.
			select {
			case <-firstRegistered:
				a.Stop()
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

		a.On(StageStopping, "late-registration", func(context.Context) error {
			close(stopping)
			return nil
		})

		a.On(StageCleanup, "check-workers", func(context.Context) error {
			for i := range completed {
				completedAtCleanup[i] = completed[i].Load()
			}
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := a.Run(ctx)
		cancel()
		registrations.Wait()
		if err != nil {
			t.Fatal(err)
		}
		if !accepted[0] {
			t.Fatal("task registration before Stop must succeed")
		}
		if accepted[attempts-1] {
			t.Fatal("task registration during Stopping must panic")
		}
		for i, registered := range accepted {
			if completedAtCleanup[i] != registered {
				t.Fatalf("iteration %d, task %d: accepted=%v, completed before Cleanup=%v",
					iteration, i, registered, completedAtCleanup[i])
			}
		}
	}
}

func TestBackgroundFailurePublishedDuringShutdown(t *testing.T) {
	a := newTestRunApp()
	firstFailure := errors.New("first worker failed")
	formatting := make(chan struct{})
	release := make(chan struct{})
	releaseError := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseError)
	lateFailure := &delayedTaskError{
		error:      errors.New("another worker failed before cancellation"),
		formatting: formatting,
		release:    release,
	}

	a.On(StageReady, "", func(ctx context.Context) error {
		a.Go("late-worker", func(context.Context) error { return lateFailure })
		select {
		case <-formatting:
		case <-ctx.Done():
			return ctx.Err()
		}

		// The late worker has passed its cancellation check, but is blocked
		// formatting its error. A different worker now triggers shutdown.
		a.Go("first-worker", func(context.Context) error { return firstFailure })
		<-ctx.Done()
		return nil
	})
	a.On(StageStopping, "", func(_ context.Context) error {
		// Run has consumed the first failure. Let the real task publish its
		// error while shutdown is waiting for managed tasks to finish.
		releaseError()
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := a.Run(ctx); !errors.Is(err, firstFailure) || !errors.Is(err, lateFailure) {
		t.Fatalf("Run must preserve both worker failures, got %v", err)
	} else if !strings.Contains(err.Error(), `background task "late-worker"`) {
		t.Fatalf("Run must preserve the late worker's name, got %v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("worker failure stopped the app only after the parent timed out")
	}
}

type delayedTaskError struct {
	error
	formatting chan struct{}
	release    <-chan struct{}
	once       sync.Once
}

func (e *delayedTaskError) Error() string {
	e.once.Do(func() { close(e.formatting) })
	<-e.release
	return e.error.Error()
}
