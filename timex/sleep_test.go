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

package timex

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	for _, duration := range []time.Duration{-time.Second, 0, time.Millisecond} {
		start := time.Now()
		if err := Sleep(context.Background(), duration); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); duration > 0 && elapsed < duration {
			t.Fatalf("slept %v, want at least %v", elapsed, duration)
		}
	}
}

func TestSleepAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("stop"))
	for _, duration := range []time.Duration{-time.Second, 0, time.Hour} {
		if err := Sleep(ctx, duration); !errors.Is(err, context.Canceled) {
			t.Fatalf("duration=%v: got %v, want context.Canceled", duration, err)
		}
	}
}

func TestSleepCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	observed := &sleepContext{Context: ctx, ready: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- Sleep(observed, time.Hour) }()

	select {
	case <-observed.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("Sleep did not start waiting for cancellation")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want context.Canceled", err)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Sleep did not return after cancellation")
	}
}

type sleepContext struct {
	context.Context
	ready chan struct{}
	once  sync.Once
}

func (c *sleepContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.ready) })
	return c.Context.Done()
}

func TestSleepDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- Sleep(ctx, time.Hour) }()

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("got %v, want context.DeadlineExceeded", err)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Sleep did not return after deadline")
	}
}
