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
	"sync/atomic"
	"testing"
	"time"
)

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

func TestValidStage(t *testing.T) {
	for _, s := range []Stage{StageInit, StageStart, StageReady, StageStopping, StageCleanup, StageExited} {
		if !validStage(s) {
			t.Errorf("expected valid stage %q", s)
		}
	}
	if validStage("bogus") {
		t.Error("expected invalid")
	}
}
