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

//go:build unix

package app

import (
	"context"
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestRegressionNoSignals(t *testing.T) {
	a := newTestRunApp()
	a.SetSignals()

	ready := make(chan struct{})
	a.On(StageReady, "", func(context.Context) error { close(ready); return nil })

	done := make(chan error, 1)
	go func() { done <- a.Run(context.Background()) }()

	<-ready

	defer a.Stop()
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		t.Fatalf("SetSignals() still handles SIGWINCH and shuts down: %v", err)

	case <-time.After(150 * time.Millisecond):
		a.Stop()
		<-done
	}
}

func TestRegressionSignalCancelsStartup(t *testing.T) {
	a := newTestRunApp()
	a.SetSignals(syscall.SIGUSR1)

	started := make(chan struct{})
	a.On(StageInit, "wait-for-signal", func(ctx context.Context) error {
		close(started)
		<-a.Context().Done()
		if ctx.Err() == nil {
			t.Error("signal canceled the lifecycle before the run context")
		}
		return ctx.Err()
	})

	done := make(chan error, 1)
	go func() { done <- a.Run(context.Background()) }()
	<-started

	defer a.Stop()
	if err := syscall.Kill(os.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("signal startup error = %v, want context.Canceled", err)
		}
		select {
		case <-a.Done():
		default:
			t.Fatal("Done stayed open after signal shutdown")
		}
	case <-time.After(2 * time.Second):
		a.Stop()
		<-done
		t.Fatal("SIGUSR1 does not cancel the startup callback context")
	}
}
