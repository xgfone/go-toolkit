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
	"sync"
	"testing"
	"time"
)

func TestAppContextAndDoneBoundaries(t *testing.T) {
	a := newTestRunApp()
	lifecycle, done := a.Context(), a.Done()
	if lifecycle == nil || done == nil {
		t.Fatal("Context and Done must be available before Run")
	}

	a.Stop()
	if lifecycle.Err() != nil {
		t.Fatal("Stop before Run must remain a no-op")
	}

	ready, exited := make(chan struct{}), make(chan struct{})
	release := make(chan struct{})
	releaseExited := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseExited)
	a.On(StageReady, "ready", func(context.Context) error { close(ready); return nil })
	a.On(StageExited, "exited", func(ctx context.Context) error {
		close(exited)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	a.On(StageInit, "worker", func(context.Context) error {
		a.Go("lifecycle-worker", func(ctx context.Context) error {
			<-lifecycle.Done()
			if ctx.Err() == nil {
				t.Error("run context must be canceled before lifecycle workers are notified")
			}
			return lifecycle.Err()
		})
		return nil
	})

	parent, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	returned := make(chan error, 1)
	go func() { returned <- a.Run(parent) }()

	select {
	case <-ready:
	case <-parent.Done():
		t.Fatal("app did not become ready")
	}

	select {
	case <-done:
		t.Fatal("Done closed while app was running")
	default:
	}

	// A rejected second Run must not cancel the first Run's lifecycle context.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("second Run must panic")
			}
		}()
		_ = a.Run(context.Background())
	}()

	if lifecycle.Err() != nil {
		t.Fatal("second Run canceled the active lifecycle")
	}

	var callers sync.WaitGroup
	for range 8 {
		callers.Go(func() {
			if a.Context() != lifecycle || a.Done() != done {
				t.Error("lifecycle handles changed while running")
			}
			a.Stop()
		})
	}

	callers.Wait()
	if lifecycle.Err() != context.Canceled {
		t.Fatal("Stop must cancel the lifecycle context before returning")
	}

	select {
	case <-exited:
	case <-parent.Done():
		t.Fatal("app did not reach Exited")
	}

	select {
	case <-done:
		t.Fatal("Done closed before the final Exited callback returned")
	default:
	}

	canceled, cancelWait := context.WithCancel(context.Background())
	cancelWait()

	if err := a.WaitContext(canceled); err != context.Canceled {
		t.Fatalf("WaitContext during shutdown = %v, want context.Canceled", err)
	}

	releaseExited()
	select {
	case err := <-returned:
		if err != nil {
			t.Fatal(err)
		}
	case <-parent.Done():
		t.Fatal("Run did not return")
	}

	if a.Context() != lifecycle || a.Done() != done || lifecycle.Err() != context.Canceled {
		t.Fatal("lifecycle handles must stay stable after Run")
	}

	select {
	case <-done:
	default:
		t.Fatal("Done stayed open after Run returned")
	}

	if err := a.WaitContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	a.Stop()
	a.Stop()
	assertPanics(t, func() { a.Go("late", func(context.Context) error { return nil }) })
}

func TestAppContextCancellationSources(t *testing.T) {
	for _, source := range []string{
		"stop", "parent", "deadline", "pre-canceled",
		"config", "callback", "module", "worker",
	} {
		t.Run(source, func(t *testing.T) {
			a := newTestRunApp()
			failure := errors.New("startup failed")
			timeout := 2 * time.Second
			if source == "deadline" {
				timeout = 20 * time.Millisecond
			}

			parent, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			wantErr := error(context.Canceled)
			switch source {
			case "deadline":
				wantErr = context.DeadlineExceeded
			case "pre-canceled":
				cancel()
			case "config":
				wantErr = failure
				a.SetConfigLoader(func(context.Context) error { return failure })
			case "callback":
				wantErr = failure
			case "module":
				wantErr = failure
				m := newTestModule("failed")
				m.init = func(context.Context, *App) error { return failure }
				a.Use(m)
			case "worker":
				wantErr = failure
			}

			a.On(StageInit, "startup", func(ctx context.Context) error {
				switch source {
				case "stop":
					a.Stop()
				case "parent":
					cancel()
				case "worker":
					a.Go("failed-worker", func(context.Context) error { return failure })
				case "callback":
					return failure
				case "module":
					return nil
				}

				// Run cannot reach its shutdown loop until this callback returns.
				// Parent cancellation must therefore reach App.Context independently.
				select {
				case <-a.Context().Done():
					if ctx.Err() == nil {
						t.Error("lifecycle cancellation preceded run cancellation")
					}
					return ctx.Err()
				case <-time.After(3 * time.Second):
					return errors.New("lifecycle cancellation did not reach startup")
				}
			})

			shutdownCalls := 0
			for _, stage := range []Stage{StageStopping, StageCleanup, StageExited} {
				a.On(stage, string(stage), func(ctx context.Context) error {
					shutdownCalls++
					if a.Context().Err() != context.Canceled || ctx.Err() != nil {
						t.Error("shutdown did not receive an independent context after lifecycle cancellation")
					}
					return nil
				})
			}

			if err := a.Run(parent); !errors.Is(err, wantErr) {
				t.Fatalf("Run error = %v, want %v", err, wantErr)
			}
			if shutdownCalls != 3 {
				t.Fatalf("shutdown callbacks = %d, want 3", shutdownCalls)
			}

			select {
			case <-a.Done():
			default:
				t.Fatal("Done stayed open after cancellation and shutdown")
			}
		})
	}
}

func TestAppDoneAfterBackgroundTimeout(t *testing.T) {
	a := newTestRunApp()
	a.SetShutdownTimeout(20 * time.Millisecond)
	release, workerExited := make(chan struct{}), make(chan struct{})
	releaseWorker := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseWorker)
	a.On(StageInit, "worker", func(context.Context) error {
		a.Go("uncooperative-worker", func(context.Context) error {
			defer close(workerExited)
			<-release
			return nil
		})
		return nil
	})

	cleanupRan := false
	a.On(StageCleanup, "cleanup", func(ctx context.Context) error {
		cleanupRan = true
		if ctx.Err() != context.DeadlineExceeded {
			t.Error("cleanup must share the exhausted task-wait deadline")
		}
		return nil
	})

	a.On(StageReady, "stop", func(context.Context) error { a.Stop(); return nil })
	if err := a.Run(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run error = %v, want a task-wait timeout", err)
	}
	if !cleanupRan {
		t.Fatal("cleanup was skipped after the task-wait timeout")
	}

	select {
	case <-a.Done():
	default:
		t.Fatal("Done did not close after the task-wait timeout")
	}

	select {
	case <-workerExited:
		t.Fatal("test worker must remain blocked after shutdown completes")
	default:
	}

	releaseWorker()
	select {
	case <-workerExited:
	case <-time.After(time.Second):
		t.Fatal("worker did not exit after release")
	}
}
