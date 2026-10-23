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
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCallbackDefaultRegistration(t *testing.T) {
	original := defaultApp
	t.Cleanup(func() { defaultApp = original })
	defaultApp = newTestRunApp()
	var events []string
	StageInit.On("", func(context.Context) error {
		events = append(events, "stage")
		return nil
	})
	Default().On(StageInit, "app", func(context.Context) error {
		events = append(events, "app")
		return nil
	})
	m := newTestModule("module")
	m.init = func(_ context.Context, got *App) error {
		if got != Default() {
			t.Error("package Use did not register on Default")
		}
		events = append(events, "module")
		return nil
	}
	Use(m)
	defaultApp.On(StageReady, "stop", func(context.Context) error { defaultApp.Stop(); return nil })
	if err := Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(events, []string{"stage", "app", "module"}) {
		t.Fatalf("default registrations = %v", events)
	}
}

func TestCallbackRegistrationValidation(t *testing.T) {
	a := newTestRunApp()
	noop := func(context.Context) error { return nil }
	assertPanics(t, func() { a.On(StageInit, "nil", nil) })
	assertPanics(t, func() { a.Cleanup(nil) })
	for _, stage := range []Stage{"", "invalid"} {
		assertPanics(t, func() { a.On(stage, "invalid", noop) })
	}
	stages := []Stage{StageInit, StageStart, StageReady, StageStopping, StageCleanup, StageExited}
	for current, stage := range stages {
		a.On(stage, "validate", func(context.Context) error {
			for target, next := range stages {
				register := func() { a.On(next, "future", noop) }
				if target <= current {
					assertPanics(t, register)
				} else {
					register()
				}
			}
			if stage == StageStart {
				a.On(StageReady, "stop", func(context.Context) error { a.Stop(); return nil })
			}
			return nil
		})
	}
	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertPanics(t, func() { a.On(StageExited, "late", noop) })
}

func TestConcurrentCallbackRegistrationDuringShutdown(t *testing.T) {
	const attempts = 32
	for iteration := range 100 {
		a := newTestRunApp()
		start := make(chan struct{})
		firstRegistered := make(chan struct{})
		cleanupStarted := make(chan struct{})
		var registrations sync.WaitGroup
		var accepted [attempts]bool
		var calls [attempts]atomic.Int32
		for i := range attempts {
			registrations.Go(func() {
				if i == 0 {
					defer close(firstRegistered)
				}
				if i == attempts-1 {
					<-cleanupStarted
				} else {
					<-start
				}

				defer func() {
					switch r := recover(); r {
					case nil,
						`app: cannot register callback for stage "cleanup" after stage "cleanup"`,
						`app: cannot register callback for stage "cleanup" after stage "exited"`:
					default:
						t.Errorf("unexpected registration panic: %v", r)
					}
				}()

				a.On(StageCleanup, "concurrent", func(context.Context) error {
					calls[i].Add(1)
					return nil
				})
				accepted[i] = true
			})
		}

		a.On(StageStopping, "register", func(ctx context.Context) error {
			close(start)
			// Keep Cleanup in the future until at least one concurrent registration
			// succeeds. The remaining callers race with entering Cleanup.
			select {
			case <-firstRegistered:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

		a.On(StageReady, "stop", func(context.Context) error {
			a.Stop()
			return nil
		})

		a.On(StageCleanup, "late-registration", func(context.Context) error {
			close(cleanupStarted)
			return nil
		})

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := a.Run(ctx)
		cancel()
		registrations.Wait()
		if err != nil {
			t.Fatal(err)
		}
		if !accepted[0] {
			t.Fatal("registration before Cleanup must succeed")
		}
		if accepted[attempts-1] {
			t.Fatal("registration after Cleanup starts must panic")
		}
		for i, registered := range accepted {
			var want int32
			if registered {
				want = 1
			}
			if got := calls[i].Load(); got != want {
				t.Fatalf("iteration %d, callback %d: called %d times, want %d", iteration, i, got, want)
			}
		}
	}
}
