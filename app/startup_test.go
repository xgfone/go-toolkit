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
	"sync/atomic"
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
			want:     []string{"config", "init", "first.Init", "stopping", "first.Close", "cleanup", "exited"},
		},
		{
			cancelAt: "first.Start",
			want: []string{
				"config", "init", "first.Init", "second.Init", "start", "first.Start",
				"stopping", "second.Close", "first.Close", "cleanup", "exited",
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
					close: testModuleCallback(record(name + ".Close")),
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

func TestPartialStartupRollback(t *testing.T) {
	for _, phase := range []string{"init", "prepare", "start"} {
		for _, mode := range []string{"error", "panic"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				a := newTestRunApp()
				failure := errors.New(phase)
				var exited atomic.Bool
				var closed, finalized bool
				m := newTestModule("module")
				work := func(context.Context, *App) error {
					a.TaskGroup("module").Go("partial", func(ctx context.Context) error {
						<-ctx.Done()
						exited.Store(true)
						return nil
					})
					if mode == "panic" {
						panic(failure)
					}
					return failure
				}

				switch phase {
				case "init":
					m.init = work
					m.stop = func(context.Context, *App) error {
						t.Error("Stop after failed Init")
						return nil
					}
					m.wait = func(context.Context, *App) error {
						t.Error("Wait after failed Init")
						return nil
					}
				case "prepare":
					m.prepare = work
					m.start = func(context.Context, *App) error {
						t.Error("Start after failed Prepare")
						return nil
					}
				case "start":
					m.start = work
				}

				m.close = func(context.Context, *App) error {
					closed = true
					if !exited.Load() {
						t.Error("Close before partial work exited")
					}
					return nil
				}

				a.Use(m)
				a.On(StageReady, "unexpected", func(context.Context) error {
					t.Error("Ready after startup failure")
					return nil
				})
				a.Finalize("logger", func(context.Context) error {
					finalized = true
					return nil
				})

				result := a.Run(context.Background())
				if !errors.Is(result, failure) || errors.Is(result, ErrShutdownIncomplete) {
					t.Fatal(result)
				}
				if !exited.Load() || !finalized || closed != (phase != "init") {
					t.Fatalf("rollback: exited=%v finalized=%v closed=%v", exited.Load(), finalized, closed)
				}
				if got := a.Wait(context.Background()); got != result {
					t.Fatalf("Wait=%v, Run=%v", got, result)
				}
			})
		}
	}
}

func TestStartupCallbackPanicsRollbackAndPublishResult(t *testing.T) {
	for _, phase := range []string{"config", "init-callback", "start-callback", "ready"} {
		t.Run(phase, func(t *testing.T) {
			a := newTestRunApp()
			failure := errors.New("panic in " + phase)
			m := newTestModule("module")
			var closed, finalized bool
			m.close = func(context.Context, *App) error { closed = true; return nil }
			fail := func(context.Context) error { panic(failure) }
			switch phase {
			case "config":
				a.SetConfigLoader(fail)
			case "init-callback":
				a.On(StageInit, "panic", fail)
			case "start-callback":
				a.On(StageStart, "panic", fail)
			case "ready":
				a.On(StageReady, "panic", fail)
			}

			a.Use(m)
			a.Finalize("logger", func(context.Context) error { finalized = true; return nil })
			a.On(StageReady, "stop", func(context.Context) error { a.Stop(); return nil })

			result := a.Run(context.Background())
			if !errors.Is(result, failure) {
				t.Fatalf("Run lost panic cause: %v", result)
			}
			initialized := phase != "config" && phase != "init-callback"
			if closed != initialized || !finalized {
				t.Errorf("rollback incomplete: closed=%v finalized=%v", closed, finalized)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			if got := a.Wait(ctx); got != result {
				t.Errorf("Wait did not publish final result: %v / %v", got, result)
			}
		})
	}
}
