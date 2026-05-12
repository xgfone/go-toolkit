package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestOnNamed_Default(t *testing.T) {
	origApp := DefaultApp
	defer func() { DefaultApp = origApp }()

	defer func() { _ = recover() }()
	OnNamed(StageInit, "", nil)
	t.Error("expected panic")
}

func TestOn_Panics_NilHook(t *testing.T) {
	defer func() { _ = recover() }()
	On(StageInit, nil)
	t.Error("expected panic")
}

func TestOn_Panics_InvalidStage(t *testing.T) {
	defer func() { _ = recover() }()
	On("bogus", func(ctx context.Context, app *App) error { return nil })
	t.Error("expected panic")
}

func TestOn_Panics_AfterRun(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)

	defer func() { _ = recover() }()
	app.On(StageInit, func(ctx context.Context, app *App) error { return nil })
	t.Error("expected panic")
}

func TestHook_AllStages(t *testing.T) {
	var order []string
	record := func(s string) func(ctx context.Context, app *App) error {
		return func(ctx context.Context, app *App) error {
			order = append(order, s)
			return nil
		}
	}

	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageInit, record("init"))
	app.On(StageStart, record("start"))
	app.On(StageReady, record("ready"))
	app.On(StageStopping, record("stopping"))
	app.On(StageCleanup, record("cleanup"))
	app.On(StageExited, record("exited"))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}

	expected := []string{"init", "start", "ready", "stopping", "cleanup", "exited"}
	if len(order) != len(expected) {
		t.Fatalf("unexpected order length: %v", order)
	}
	for i, s := range expected {
		if order[i] != s {
			t.Errorf("index %d: expected %q, got %q", i, s, order[i])
		}
	}
}

func TestHook_Named(t *testing.T) {
	var called atomic.Bool
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()
	app.OnNamed(StageInit, "myhook", func(ctx context.Context, app *App) error {
		called.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Error("named hook not called")
	}
}

func TestHook_InitError(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageInit, func(ctx context.Context, app *App) error { return errors.New("init hook fail") })
	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHook_StartError(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageStart, func(ctx context.Context, app *App) error { return errors.New("start hook fail") })
	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHook_ReadyError(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageReady, func(ctx context.Context, app *App) error { return errors.New("ready hook fail") })
	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHook_StoppingError_Continues(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	var secondCalled atomic.Bool
	app.On(StageStopping, func(ctx context.Context, app *App) error { return errors.New("first fail") })
	app.On(StageStopping, func(ctx context.Context, app *App) error {
		secondCalled.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := app.Run(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !secondCalled.Load() {
		t.Error("second stopping hook should still execute")
	}
}

func TestHook_ExitedError_Continues(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	var secondCalled atomic.Bool
	app.On(StageExited, func(ctx context.Context, app *App) error { return errors.New("first fail") })
	app.On(StageExited, func(ctx context.Context, app *App) error {
		secondCalled.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := app.Run(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !secondCalled.Load() {
		t.Error("second exited hook should still execute")
	}
}

func TestHook_MultiplePerStage(t *testing.T) {
	var order []int
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	for i := range 3 {
		n := i
		app.On(StageInit, func(ctx context.Context, app *App) error {
			order = append(order, n)
			return nil
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 || order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Errorf("unexpected order: %v", order)
	}
}

func TestStageOn(t *testing.T) {
	origApp := DefaultApp
	defer func() { DefaultApp = origApp }()

	DefaultApp = New()
	DefaultApp.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	DefaultApp.SetSignals()

	var called atomic.Bool
	StageInit.On(func(ctx context.Context, app *App) error {
		called.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := DefaultApp.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Error("Stage.On hook not called")
	}
}

func TestStageOnNamed(t *testing.T) {
	origApp := DefaultApp
	defer func() { DefaultApp = origApp }()

	DefaultApp = New()
	DefaultApp.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	DefaultApp.SetSignals()

	var called atomic.Bool
	StageInit.OnNamed("test-hook", func(ctx context.Context, app *App) error {
		called.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := DefaultApp.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Error("Stage.OnNamed hook not called")
	}
}

func TestStageOn_Panics_NilHook(t *testing.T) {
	defer func() { _ = recover() }()
	StageInit.On(nil)
	t.Error("expected panic")
}

func TestStageOnNamed_Panics_NilHook(t *testing.T) {
	defer func() { _ = recover() }()
	StageInit.OnNamed("name", nil)
	t.Error("expected panic")
}

func TestStageOn_Panics_InvalidStage(t *testing.T) {
	defer func() { _ = recover() }()
	Stage("bogus").On(func(ctx context.Context, app *App) error { return nil })
	t.Error("expected panic")
}

func TestStageOnNamed_Panics_InvalidStage(t *testing.T) {
	defer func() { _ = recover() }()
	Stage("bogus").OnNamed("name", func(ctx context.Context, app *App) error { return nil })
	t.Error("expected panic")
}

func TestHookLabel(t *testing.T) {
	if l := hookLabel(StageInit, "n", 0); l != `"n" at stage "init"` {
		t.Errorf("unexpected named label: %s", l)
	}
	if l := hookLabel(StageInit, "", 3); l != `#3 at stage "init"` {
		t.Errorf("unexpected unnamed label: %s", l)
	}
}

func TestHook_Cleanup_Executed(t *testing.T) {
	var called atomic.Bool
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageCleanup, func(ctx context.Context, app *App) error {
		called.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if !called.Load() {
		t.Error("StageCleanup hook not called")
	}
}

func TestHook_Cleanup_ReverseOrder(t *testing.T) {
	var order []int
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	for i := range 3 {
		n := i
		app.On(StageCleanup, func(ctx context.Context, app *App) error {
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
		t.Errorf("unexpected reverse order: %v", order)
	}
}

func TestHook_Cleanup_Error_Continues(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	var secondCalled atomic.Bool
	app.On(StageCleanup, func(ctx context.Context, app *App) error {
		return errors.New("first cleanup fail")
	})
	app.On(StageCleanup, func(ctx context.Context, app *App) error {
		secondCalled.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := app.Run(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !secondCalled.Load() {
		t.Error("second cleanup hook should still execute after first error")
	}
}
