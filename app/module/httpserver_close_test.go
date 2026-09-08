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

package module

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type closeCountingListener struct {
	accepting chan struct{}
	closed    chan struct{}
	closes    atomic.Int32
	closeErr  error
}

func (l *closeCountingListener) Accept() (net.Conn, error) {
	close(l.accepting)
	<-l.closed
	return nil, net.ErrClosed
}

func (l *closeCountingListener) Close() error {
	if l.closes.Add(1) != 1 {
		return errors.New("listener closed more than once")
	}

	close(l.closed)
	return l.closeErr
}

func (l *closeCountingListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}
}

func TestHttpServerStopClosesListenerOnce(t *testing.T) {
	closeErr := errors.New("listener close failed")
	for _, tc := range []struct {
		name     string
		serving  bool
		closeErr error
	}{
		{name: "before serve"},
		{name: "during serve", serving: true},
		{name: "before serve with close error", closeErr: closeErr},
		{name: "during serve with close error", serving: true, closeErr: closeErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := &closeCountingListener{
				accepting: make(chan struct{}), closed: make(chan struct{}),
				closeErr: tc.closeErr,
			}
			s := &HttpServer{
				addr: l.Addr().String(), server: &http.Server{},
				listen: &onceCloseListener{Listener: l},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			defer s.listen.Close() //nolint:errcheck

			done := make(chan error, 1)
			serve := func() { done <- s.server.Serve(s.listen) }
			if tc.serving {
				go serve()
				select {
				case <-l.accepting:
				case <-ctx.Done():
					t.Fatal("Serve never started")
				}
			}

			for range 2 {
				if err := s.Stop(ctx, nil); err != tc.closeErr {
					t.Fatalf("Stop error = %v, want %v", err, tc.closeErr)
				}
			}

			if !tc.serving {
				go serve()
			}

			select {
			case err := <-done:
				if !errors.Is(err, http.ErrServerClosed) {
					t.Fatalf("Serve error = %v, want ErrServerClosed", err)
				}

			case <-ctx.Done():
				t.Fatal("Serve did not exit")
			}

			if got := l.closes.Load(); got != 1 {
				t.Fatalf("listener closed %d times, want 1", got)
			}
		})
	}
}

func TestHttpServerConcurrentStopClosesListenerOnce(t *testing.T) {
	l := &closeCountingListener{accepting: make(chan struct{}), closed: make(chan struct{})}
	s := &HttpServer{
		addr: l.Addr().String(), server: &http.Server{},
		listen: &onceCloseListener{Listener: l},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	defer s.listen.Close() //nolint:errcheck

	done := make(chan error, 1)
	go func() { done <- s.server.Serve(s.listen) }()

	stopped := make(chan error, 8)
	for range cap(stopped) {
		go func() { stopped <- s.Stop(ctx, nil) }()
	}

	for range cap(stopped) {
		select {
		case err := <-stopped:
			if err != nil {
				t.Fatal(err)
			}

		case <-ctx.Done():
			t.Fatal("Stop did not exit")
		}
	}

	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Serve error = %v, want ErrServerClosed", err)
		}

	case <-ctx.Done():
		t.Fatal("Serve did not exit")
	}

	if got := l.closes.Load(); got != 1 {
		t.Fatalf("listener closed %d times, want 1", got)
	}
}
