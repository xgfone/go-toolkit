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
	"testing"
	"time"
)

func TestGoValidation(t *testing.T) {
	assertPanics(t, func() { New().Go("", nil) })
	assertPanics(t, func() { New().Go("", func(context.Context) error { return nil }) })
}

func TestGoCompletionAfterCancellation(t *testing.T) {
	for _, result := range []error{nil, errors.New("late task error")} {
		name := "success"
		if result != nil {
			name = "late error ignored"
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
			if err := a.Run(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !completed {
				t.Fatal("Run returned before the managed task finished")
			}
		})
	}
}

func TestGoFailureStopsStartup(t *testing.T) {
	failure, panicFailure := errors.New("task failure"), errors.New("panic failure")
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

func TestGo_Error_FullErrorChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := &App{
		state:     stateRunning,
		runCtx:    ctx,
		cancelRun: cancel,
		errCh:     make(chan error, 1),
	}
	queuedErr := errors.New("queued failure")
	app.errCh <- queuedErr

	// Keep the channel full until the task exits so the send must select default.
	app.Go("worker", func(context.Context) error {
		return errors.New("task failure")
	})

	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()

	if err := app.waitBackground(waitCtx); err != nil {
		t.Fatalf("background task blocked on a full error channel: %v", err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("background task must cancel the run context, got %v", ctx.Err())
	}

	select {
	case err := <-app.errCh:
		if err != queuedErr {
			t.Fatalf("queued error = %v, want %v", err, queuedErr)
		}

	default:
		t.Fatal("background task removed the queued error")
	}
}
