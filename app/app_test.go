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
	"os"
	"runtime/debug"
	"testing"
	"time"
)

func newTestRunApp() *App {
	a := New()
	a.SetSignals()
	a.SetConfigLoader(func(context.Context) error { return nil })
	return a
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("expected panic")
		}
	}()
	fn()
}

func TestAppMetadata(t *testing.T) {
	a := New()
	if a.Name() == "" || a.Version() != "0.0.0" {
		t.Fatalf("unexpected defaults: name=%q, version=%q", a.Name(), a.Version())
	}

	a.SetName("myapp")
	a.SetVersion("v2")
	a.SetCommit("abc123")
	if a.Name() != "myapp" || a.Version() != "v2" || a.Commit() != "abc123" {
		t.Fatalf("metadata = %q, %q, %q", a.Name(), a.Version(), a.Commit())
	}
}

func TestAppConfigurationValidation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*App)
	}{
		{"empty name", func(a *App) { a.SetName("") }},
		{"zero timeout", func(a *App) { a.SetShutdownTimeout(0) }},
		{"negative timeout", func(a *App) { a.SetShutdownTimeout(-time.Second) }},
		{"zero cleanup timeout", func(a *App) { a.SetCleanupTimeout(0) }},
		{"negative exited timeout", func(a *App) { a.SetExitedTimeout(-time.Second) }},
		{"zero finalizer timeout", func(a *App) { a.SetFinalizerTimeout(0) }},
		{"nil signal", func(a *App) { a.SetSignals(nil) }},
		{"nil config loader", func(a *App) { a.SetConfigLoader(nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertPanics(t, func() {
				tc.configure(New())
			})
		})
	}
}

func TestAppConfigurationAfterRun(t *testing.T) {
	a := newTestRunApp()
	configurations := []struct {
		name      string
		configure func()
	}{
		{"name", func() { a.SetName("late") }},
		{"version", func() { a.SetVersion("late") }},
		{"commit", func() { a.SetCommit("late") }},
		{"build time", func() { a.SetBuildTime(time.Now()) }},
		{"timeout", func() { a.SetShutdownTimeout(time.Second) }},
		{"cleanup timeout", func() { a.SetCleanupTimeout(time.Second) }},
		{"exited timeout", func() { a.SetExitedTimeout(time.Second) }},
		{"finalizer timeout", func() { a.SetFinalizerTimeout(time.Second) }},
		{"signals", func() { a.SetSignals(os.Interrupt) }},
		{"config loader", func() { a.SetConfigLoader(func(context.Context) error { return nil }) }},
		{"module", func() { a.Use(newTestModule("late")) }},
	}

	check := func(state string) {
		for _, tc := range configurations {
			t.Run(state+"/"+tc.name, func(t *testing.T) { assertPanics(t, tc.configure) })
		}
	}
	a.On(StageReady, "validate", func(context.Context) error {
		check("running")
		a.Stop()
		return nil
	})

	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	check("exited")
}

func TestSetBuildTime(t *testing.T) {
	originalLocal := time.Local
	t.Cleanup(func() { time.Local = originalLocal })
	time.Local = time.FixedZone("TestZone", 3600*8)

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	app := New()
	app.SetBuildTime(now)
	got := app.BuildTime()
	if !got.Equal(now) {
		t.Errorf("expected %v, got %v", now, got)
	}
	if got.Location() != time.Local {
		t.Errorf("expected location %v, got %v", time.Local, got.Location())
	}
}

func TestName_Version_Convenience(t *testing.T) {
	orig := defaultApp
	defer func() { defaultApp = orig }()
	defaultApp = New()
	app := Default()
	if app == nil || app != defaultApp || app != Default() {
		t.Fatal("Default must return the same non-nil default app")
	}
	app.SetName("testname")
	app.SetVersion("testver")

	if Name() != "testname" {
		t.Errorf("expected 'testname', got %q", Name())
	}
	if Version() != "testver" {
		t.Errorf("expected 'testver', got %q", Version())
	}
}

func TestSetCommitFromBuildSettings(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings []debug.BuildSetting
		want     string
	}{
		{name: "revision", settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc1234567890"}}, want: "abc1234"},
		{name: "short revision", settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}}, want: "abc"},
		{name: "other setting", settings: []debug.BuildSetting{{Key: "other.key", Value: "val"}}},
		{name: "empty settings"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := New()
			setCommitFromBuildSettings(a, tc.settings)
			if a.Commit() != tc.want {
				t.Errorf("Commit = %q, want %q", a.Commit(), tc.want)
			}
		})
	}
}
