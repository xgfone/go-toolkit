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
	"sync/atomic"
	"testing"
	"time"
)

func TestGo_Panics_NilFunc(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	var panicked bool
	app.On(StageStart, func(ctx context.Context, app *App) error {
		func() {
			defer func() {
				if recover() != nil {
					panicked = true
				}
			}()
			app.Go(nil)
		}()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)
	if !panicked {
		t.Error("expected panic")
	}
}

func TestGo_Panics_BeforeRun(t *testing.T) {
	defer func() { _ = recover() }()
	New().Go(func(ctx context.Context) error { return nil })
	t.Error("expected panic")
}

func TestGo_Success(t *testing.T) {
	var done atomic.Bool
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageStart, func(ctx context.Context, app *App) error {
		app.Go(func(ctx context.Context) error {
			<-ctx.Done()
			done.Store(true)
			return nil
		})
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !done.Load() {
		t.Error("background task should have finished")
	}
}

func TestGo_Error_TriggersShutdown(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageStart, func(ctx context.Context, app *App) error {
		app.Go(func(ctx context.Context) error {
			return errors.New("task failure")
		})
		return nil
	})

	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from background task")
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
	app.GoNamed("worker", func(context.Context) error {
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

func TestGo_Convenience(t *testing.T) {
	orig := DefaultApp
	defer func() { DefaultApp = orig }()
	DefaultApp = New()
	DefaultApp.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	DefaultApp.SetSignals()

	var called atomic.Bool
	DefaultApp.On(StageStart, func(ctx context.Context, app *App) error {
		Go(func(ctx context.Context) error {
			<-ctx.Done()
			called.Store(true)
			return nil
		})
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	if err := DefaultApp.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Error("convenience Go not called")
	}
}
