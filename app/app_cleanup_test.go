package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefer_Panics_EmptyName(t *testing.T) {
	defer func() { _ = recover() }()
	New().Defer("", func(ctx context.Context) error { return nil })
	t.Error("expected panic")
}

func TestDefer_Panics_NilFunc(t *testing.T) {
	defer func() { _ = recover() }()
	New().Defer("c", nil)
	t.Error("expected panic")
}

func TestDefer_Panics_DuringShutdown(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	var panicked bool
	app.On(StageStopping, func(ctx context.Context, app *App) error {
		func() {
			defer func() {
				if recover() != nil {
					panicked = true
				}
			}()
			app.Defer("c", func(ctx context.Context) error { return nil })
		}()
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)
	if !panicked {
		t.Error("expected panic inside stopping hook")
	}
}

func TestDefer_Executed(t *testing.T) {
	var called atomic.Bool
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.Defer("c", func(ctx context.Context) error {
		called.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Error("defer not called")
	}
}

func TestDefer_ReverseOrder(t *testing.T) {
	var order []int
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	for i := range 3 {
		n := i
		app.Defer("c", func(ctx context.Context) error {
			order = append(order, n)
			return nil
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 || order[0] != 2 || order[1] != 1 || order[2] != 0 {
		t.Errorf("unexpected order: %v", order)
	}
}

func TestDefer_Error(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.Defer("c", func(ctx context.Context) error { return errors.New("defer fail") })

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := app.Run(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDefer_FromInit(t *testing.T) {
	var called atomic.Bool
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	m := newTestModule("mod")
	m.init = func(ctx context.Context, app *App) error {
		app.Defer("from-init", func(ctx context.Context) error {
			called.Store(true)
			return nil
		})
		atomic.AddInt32(m.initCalled, 1)
		return nil
	}
	app.Use(m)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Error("defer from init not called")
	}
}
