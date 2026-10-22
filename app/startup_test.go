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
	"slices"
	"strings"
	"testing"
	"time"
)

func TestStopSkipsRemainingStartupCallbacks(t *testing.T) {
	a := newTestRunApp()

	cleaned := false
	a.Cleanup(func() error { cleaned = true; return nil })

	a.On(StageInit, "", func(_ context.Context) error {
		a.Stop()
		return nil
	})
	a.On(StageInit, "", func(context.Context) error {
		t.Error("later startup callback must not run after Stop")
		return nil
	})

	if err := a.Run(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected startup cancellation, got %v", err)
	}
	if !cleaned {
		t.Fatal("startup cancellation must run cleanup")
	}
}

func TestStartupCancellationRollsBack(t *testing.T) {
	for _, tc := range []struct {
		cancelAt string
		want     []string
	}{
		{
			cancelAt: "before config",
			want:     []string{"stopping", "cleanup", "exited"},
		},
		{
			cancelAt: "config",
			want:     []string{"config", "stopping", "cleanup", "exited"},
		},
		{
			cancelAt: "first.Init",
			want:     []string{"config", "init", "first.Init", "stopping", "first.Stop", "cleanup", "exited"},
		},
		{
			cancelAt: "first.Start",
			want: []string{
				"config", "init", "first.Init", "second.Init", "start", "first.Start",
				"stopping", "second.Stop", "first.Stop", "cleanup", "exited",
			},
		},
	} {
		t.Run(tc.cancelAt, func(t *testing.T) {
			a := newTestRunApp()
			var events []string
			record := func(event string) func(ctx context.Context) error {
				return func(_ context.Context) error {
					events = append(events, event)
					if event == tc.cancelAt {
						a.Stop()
					}
					return nil
				}
			}

			a.SetConfigLoader(record("config"))
			for _, stage := range []Stage{
				StageInit,
				StageStart,
				StageReady,
				StageStopping,
				StageCleanup,
				StageExited,
			} {
				a.On(stage, "", record(string(stage)))
			}

			for _, name := range []string{"first", "second"} {
				a.Use(&testModule{
					name:  name,
					init:  testModuleCallback(record(name + ".Init")),
					start: testModuleCallback(record(name + ".Start")),
					stop:  testModuleCallback(record(name + ".Stop")),
				})
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if tc.cancelAt == "before config" {
				cancel()
			}

			if err := a.Run(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("Run error = %v, want context.Canceled", err)
			}
			if !slices.Equal(events, tc.want) {
				t.Fatalf("lifecycle = %v, want %v", events, tc.want)
			}
		})
	}
}

func TestBackgroundFailureAtReadyPreservesError(t *testing.T) {
	a := newTestRunApp()
	failure := errors.New("worker failed at readiness")
	cleaned := false
	a.Cleanup(func() error { cleaned = true; return nil })
	// Ready may return nil after a background failure canceled the run context.
	a.On(StageReady, "", func(ctx context.Context) error {
		a.Go("ready-worker", func(context.Context) error { return failure })
		<-ctx.Done()
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := a.Run(ctx)
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "ready-worker") {
		t.Fatalf("Run lost the named worker failure: %v", err)
	}
	if !cleaned {
		t.Fatal("cleanup did not run")
	}
}

func TestBackgroundFailurePublishedDuringShutdown(t *testing.T) {
	a := newTestRunApp()
	firstFailure := errors.New("first worker failed")
	lateFailure := errors.New("another worker failed before cancellation")
	a.On(StageReady, "", func(ctx context.Context) error {
		a.Go("first-worker", func(context.Context) error { return firstFailure })
		<-ctx.Done()
		return nil
	})
	a.On(StageStopping, "", func(_ context.Context) error {
		// Simulate a worker that passed its cancellation check before shutdown,
		// then published its failure after Run consumed the first worker error.
		a.errCh <- lateFailure
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := a.Run(ctx); !errors.Is(err, firstFailure) || !errors.Is(err, lateFailure) {
		t.Fatalf("Run must preserve both worker failures, got %v", err)
	}
}
