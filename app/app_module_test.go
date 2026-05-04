package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// testModule is a minimal Module implementation for testing.
type testModule struct {
	name                    string
	init, start, stop       func(ctx context.Context, app *App) error
	initCalled, startCalled *int32
	stopCalled              *int32
}

func newTestModule(name string) *testModule {
	m := &testModule{name: name}
	m.initCalled = new(int32)
	m.startCalled = new(int32)
	m.stopCalled = new(int32)
	m.init = func(ctx context.Context, app *App) error {
		atomic.AddInt32(m.initCalled, 1)
		if name == "init-err" {
			return errors.New("init fail")
		}
		return nil
	}
	m.start = func(ctx context.Context, app *App) error {
		atomic.AddInt32(m.startCalled, 1)
		if name == "start-err" {
			return errors.New("start fail")
		}
		return nil
	}
	m.stop = func(ctx context.Context, app *App) error {
		atomic.AddInt32(m.stopCalled, 1)
		return nil
	}
	return m
}

func (m *testModule) Name() string                              { return m.name }
func (m *testModule) Init(ctx context.Context, app *App) error  { return m.init(ctx, app) }
func (m *testModule) Start(ctx context.Context, app *App) error { return m.start(ctx, app) }
func (m *testModule) Stop(ctx context.Context, app *App) error  { return m.stop(ctx, app) }

func TestUse_Panics_NilModule(t *testing.T) {
	defer func() { _ = recover() }()
	Use(nil)
	t.Error("expected panic")
}

func TestUse_Panics_EmptyName(t *testing.T) {
	defer func() { _ = recover() }()
	Use(newTestModule(""))
	t.Error("expected panic")
}

func TestUse_Panics_AfterRun(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)

	defer func() { _ = recover() }()
	app.Use(newTestModule("m"))
	t.Error("expected panic")
}

func TestModule_NormalLifecycle(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	mod := newTestModule("mod")
	app.Use(mod)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(mod.initCalled) != 1 {
		t.Error("init not called exactly once")
	}
	if atomic.LoadInt32(mod.startCalled) != 1 {
		t.Error("start not called exactly once")
	}
	if atomic.LoadInt32(mod.stopCalled) != 1 {
		t.Error("stop not called exactly once")
	}
}

func TestModule_InitError(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	mod := newTestModule("init-err")
	app.Use(mod)

	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if atomic.LoadInt32(mod.initCalled) != 1 {
		t.Error("init should be called")
	}
	if atomic.LoadInt32(mod.stopCalled) != 0 {
		t.Error("stop should NOT be called after init error")
	}
}

func TestModule_StartError(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	mod := newTestModule("start-err")
	app.Use(mod)

	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if atomic.LoadInt32(mod.initCalled) != 1 {
		t.Error("init should be called")
	}
	if atomic.LoadInt32(mod.startCalled) != 1 {
		t.Error("start should be called")
	}
	if atomic.LoadInt32(mod.stopCalled) != 1 {
		t.Error("stop should be called after start error")
	}
}

func TestModule_StopError(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	mod := newTestModule("mod")
	mod.stop = func(ctx context.Context, app *App) error {
		atomic.AddInt32(mod.stopCalled, 1)
		return errors.New("stop fail")
	}
	app.Use(mod)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := app.Run(ctx)
	if err == nil {
		t.Fatal("expected stop error")
	}
}

func TestModule_Multiple_ReverseStopOrder(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	var order []int32
	m1 := newTestModule("m1")
	m1.stop = func(ctx context.Context, app *App) error {
		atomic.AddInt32(m1.stopCalled, 1)
		order = append(order, 1)
		return nil
	}
	m2 := newTestModule("m2")
	m2.stop = func(ctx context.Context, app *App) error {
		atomic.AddInt32(m2.stopCalled, 1)
		order = append(order, 2)
		return nil
	}
	app.Use(m1, m2)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}

	if len(order) != 2 || order[0] != 2 || order[1] != 1 {
		t.Errorf("unexpected stop order: %v", order)
	}
}
