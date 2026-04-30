package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeferCleanup_Panics_EmptyName(t *testing.T) {
	defer func() { _ = recover() }()
	New().DeferCleanup("", func(ctx context.Context) error { return nil })
	t.Error("expected panic")
}

func TestDeferCleanup_Panics_NilFunc(t *testing.T) {
	defer func() { _ = recover() }()
	New().DeferCleanup("c", nil)
	t.Error("expected panic")
}

func TestDeferCleanup_Panics_DuringShutdown(t *testing.T) {
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
			app.DeferCleanup("c", func(ctx context.Context) error { return nil })
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

func TestCleanup_Executed(t *testing.T) {
	var called atomic.Bool
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.DeferCleanup("c", func(ctx context.Context) error {
		called.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Error("cleanup not called")
	}
}

func TestCleanup_ReverseOrder(t *testing.T) {
	var order []int
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	for i := range 3 {
		n := i
		app.DeferCleanup("c", func(ctx context.Context) error {
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

func TestCleanup_Error(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.DeferCleanup("c", func(ctx context.Context) error { return errors.New("cleanup fail") })

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := app.Run(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeferCleanup_FromInit(t *testing.T) {
	var called atomic.Bool
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	m := newTestModule("mod")
	m.init = func(ctx context.Context, app *App) error {
		app.DeferCleanup("from-init", func(ctx context.Context) error {
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
		t.Error("cleanup from init not called")
	}
}
