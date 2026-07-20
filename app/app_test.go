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
	"runtime/debug"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	app := New()
	if app.Name() == "" {
		t.Error("expected non-empty default name")
	}
	if app.Version() != "0.0.0" {
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

func TestSetCommit(t *testing.T) {
	app := New()
	app.SetCommit("abc123")
	if app.Commit() != "abc123" {
		t.Errorf("expected 'abc123', got %q", app.Commit())
	}
}

func TestSetCommit_Panics_AfterRun(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)

	defer func() { _ = recover() }()
	app.SetCommit("x")
	t.Error("expected panic")
}

func TestSetBuildTime(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	app := New()
	app.SetBuildTime(now)
	if got := app.BuildTime(); !got.Equal(now) {
		t.Errorf("expected %v, got %v", now, got)
	}
}

func TestSetBuildTime_Panics_AfterRun(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)

	defer func() { _ = recover() }()
	app.SetBuildTime(time.Now())
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
		app.Go(func(ctx context.Context) error {
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

func TestGo_Error_ContextCancelled_NoShutdown(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	app.On(StageStart, func(ctx context.Context, app *App) error {
		app.Go(func(ctx context.Context) error {
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

func TestWait_BlocksUntilRunExits(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		_ = app.Run(ctx)
	}()

	waitDone := make(chan struct{})
	go func() {
		app.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Fatal("Wait should block before Run exits")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()

	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("Wait should return after Run exits")
	}

	<-runDone
}

func TestWait_ReturnsImmediatelyAfterRunExit(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		app.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Wait should return immediately after Run exits")
	}
}

func TestWaitContext_ReturnsCtxErrWhenCanceledFirst(t *testing.T) {
	app := New()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := app.WaitContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got: %v", err)
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Fatal("WaitContext should wait until context is done")
	}
}

func TestWaitContext_ReturnsNilWhenRunExited(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	runCtx, runCancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); runCancel() }()
	if err := app.Run(runCtx); err != nil {
		t.Fatal(err)
	}

	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()

	if err := app.WaitContext(waitCtx); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}

func TestStop_StopsRunningApp(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	runDone := make(chan error, 1)
	go func() {
		runDone <- app.Run(context.Background())
	}()

	time.Sleep(50 * time.Millisecond)
	app.Stop()

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("expected nil, got: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run should stop after Stop")
	}
}

func TestSetCommitFromBuildSettings(t *testing.T) {
	t.Run("with_vcs_revision", func(t *testing.T) {
		a := New()
		a.SetCommit("")
		setCommitFromBuildSettings(a, []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abc1234567890"},
		})
		if a.Commit() != "abc1234" {
			t.Errorf("expected abc1234, got %q", a.Commit())
		}
	})

	t.Run("without_vcs_revision", func(t *testing.T) {
		a := New()
		a.SetCommit("")
		setCommitFromBuildSettings(a, []debug.BuildSetting{
			{Key: "other.key", Value: "val"},
		})
		if a.Commit() != "" {
			t.Errorf("expected empty, got %q", a.Commit())
		}
	})

	t.Run("empty_settings", func(t *testing.T) {
		a := New()
		a.SetCommit("")
		setCommitFromBuildSettings(a, nil)
		if a.Commit() != "" {
			t.Errorf("expected empty, got %q", a.Commit())
		}
	})
}

func TestStop_NoOpWhenNotRunningOrExited(t *testing.T) {
	app := New()
	app.Stop() // no-op before Run

	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}

	app.Stop() // no-op after Run exits
	app.Stop()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer waitCancel()
	if err := app.WaitContext(waitCtx); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
}
