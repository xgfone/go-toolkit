package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	app := New()
	if app.Name() == "" {
		t.Error("expected non-empty default name")
	}
	if app.Version() != "" {
		t.Error("expected empty default version")
	}
}

func TestSetName(t *testing.T) {
	app := New()
	app.SetName("myapp")
	if app.Name() != "myapp" {
		t.Errorf("expected 'myapp', got %q", app.Name())
	}
}

func TestSetName_Panics_Empty(t *testing.T) {
	defer func() { _ = recover() }()
	New().SetName("")
	t.Error("expected panic")
}

func TestSetName_Panics_AfterRun(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	_ = app.Run(ctx)

	defer func() { _ = recover() }()
	app.SetName("x")
	t.Error("expected panic")
}

func TestSetVersion(t *testing.T) {
	app := New()
	app.SetVersion("v2")
	if app.Version() != "v2" {
		t.Errorf("expected 'v2', got %q", app.Version())
	}
}

func TestSetVersion_Panics_AfterRun(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)

	defer func() { _ = recover() }()
	app.SetVersion("x")
	t.Error("expected panic")
}

func TestSetShutdownTimeout_Panics_NonPositive(t *testing.T) {
	defer func() { _ = recover() }()
	New().SetShutdownTimeout(0)
	t.Error("expected panic")
}

func TestSetShutdownTimeout_Panics_AfterRun(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)

	defer func() { _ = recover() }()
	app.SetShutdownTimeout(time.Second)
	t.Error("expected panic")
}

func TestSetShutdownTimeout(t *testing.T) {
	app := New()
	app.SetShutdownTimeout(5 * time.Second)
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(10 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSetSignals_Panics_Nil(t *testing.T) {
	defer func() { _ = recover() }()
	New().SetSignals(nil)
	t.Error("expected panic")
}

func TestSetSignals_Empty(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals() // no signals — only stops via ctx cancel or error

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestRun(t *testing.T) {
	origApp := DefaultApp
	defer func() { DefaultApp = origApp }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := Run(ctx); err != nil {
		t.Errorf("expect nil, but got an error: %v", err)
	}
}

func TestRun_ConfigError(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return errors.New("config fail") })
	app.SetSignals()

	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRun_Panics_DoubleCall(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		_ = app.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	func() {
		defer func() {
			if recover() == nil {
				t.Error("expected panic on double Run")
			}
		}()
		_ = app.Run(context.Background())
	}()

	cancel()
	<-firstDone
}

func TestShutdownBackgroundTaskTimeout(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()
	app.SetShutdownTimeout(50 * time.Millisecond)

	app.On(StageStart, func(ctx context.Context, app *App) error {
		app.Go("slow", func(ctx context.Context) error {
			select {}
		})
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()

	err := app.Run(ctx)
	if err == nil {
		t.Fatal("expected timeout error waiting for background tasks")
	}
}

func TestName_Version_Convenience(t *testing.T) {
	orig := DefaultApp
	defer func() { DefaultApp = orig }()
	DefaultApp = New()
	DefaultApp.SetName("testname")
	DefaultApp.SetVersion("testver")

	if Name() != "testname" {
		t.Errorf("expected 'testname', got %q", Name())
	}
	if Version() != "testver" {
		t.Errorf("expected 'testver', got %q", Version())
	}
}

func TestHookLabel(t *testing.T) {
	if l := hookLabel(StageInit, "n", 0); l != `"n" at stage "init"` {
		t.Errorf("unexpected named label: %s", l)
	}
	if l := hookLabel(StageInit, "", 3); l != `#3 at stage "init"` {
		t.Errorf("unexpected unnamed label: %s", l)
	}
}

func TestValidStage(t *testing.T) {
	for _, s := range []Stage{StageInit, StageStart, StageReady, StageStopping, StageExited} {
		if !validStage(s) {
			t.Errorf("expected valid stage %q", s)
		}
	}
	if validStage("bogus") {
		t.Error("expected invalid")
	}
}

func TestGo_Error_ContextCancelled_NoShutdown(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageStart, func(ctx context.Context, app *App) error {
		app.Go("task", func(ctx context.Context) error {
			<-ctx.Done()
			return errors.New("late error")
		})
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal("expected no error because task error happens after cancel:", err)
	}
}
