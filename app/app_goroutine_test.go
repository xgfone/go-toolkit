package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestGo_Panics_EmptyName(t *testing.T) {
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
			app.Go("", func(ctx context.Context) error { return nil })
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
			app.Go("t", nil)
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
	New().Go("t", func(ctx context.Context) error { return nil })
	t.Error("expected panic")
}

func TestGo_Success(t *testing.T) {
	var done atomic.Bool
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageStart, func(ctx context.Context, app *App) error {
		app.Go("task", func(ctx context.Context) error {
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
		app.Go("task", func(ctx context.Context) error {
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
		Go("pkgGo", func(ctx context.Context) error {
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
