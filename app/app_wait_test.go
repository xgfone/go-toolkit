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
	"sync"
	"testing"
	"time"
)

func TestAppWaitFinalResultAndCancellation(t *testing.T) {
	a := newTestRunApp()
	ready := make(chan struct{})
	release := make(chan struct{})
	finalizing := make(chan struct{})
	releaseFinalizer := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseFinalizer)

	failure := errors.New("logger flush failed")
	a.On(StageReady, "ready", func(context.Context) error {
		close(ready)
		return nil
	})
	a.Finalize("logger", func(context.Context) error {
		close(finalizing)
		<-release
		return failure
	})

	returned := make(chan error, 1)
	go func() { returned <- a.Run(context.Background()) }()
	<-ready

	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	if err := a.Wait(canceled); err != context.Canceled {
		t.Fatal(err)
	}
	if a.Context().Err() != nil {
		t.Fatal("canceling Wait stopped app")
	}

	a.Stop()
	<-finalizing
	results := make(chan error, 16)
	for range cap(results) {
		go func() { results <- a.Wait(context.Background()) }()
	}

	select {
	case err := <-results:
		t.Fatalf("Wait returned before finalizer: %v", err)
	default:
	}

	releaseFinalizer()
	result := <-returned
	if !errors.Is(result, failure) {
		t.Fatal(result)
	}

	for range cap(results) {
		if got := <-results; got != result {
			t.Fatalf("Wait=%v Run=%v", got, result)
		}
	}
	if got := a.Wait(canceled); got != result {
		t.Fatalf("completed result lost to canceled context: %v", got)
	}
}

func TestWait(t *testing.T) {
	a := newTestRunApp()
	waited := make(chan struct{})
	go func() { _ = a.Wait(context.Background()); close(waited) }()

	select {
	case <-waited:
		t.Fatal("Wait returned before Run")
	case <-time.After(20 * time.Millisecond):
	}

	a.On(StageReady, "stop", func(context.Context) error {
		a.Stop()
		return nil
	})
	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	select {
	case <-waited:
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after Run")
	}
}

func TestWaitDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := New().Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait = %v, want DeadlineExceeded", err)
	}
}
