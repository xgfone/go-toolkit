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
