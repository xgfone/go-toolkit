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
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xgfone/go-toolkit/app"
)

func TestHTTPDrainTimeoutRetainsDatabaseAndLogger(t *testing.T) {
	a := app.New()
	a.SetSignals()
	a.SetConfigLoader(func(context.Context) error { return nil })
	a.SetShutdownTimeout(20 * time.Millisecond)
	parent, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	started := make(chan struct{})
	canceled := make(chan struct{})
	exited := make(chan struct{})
	release := make(chan struct{})
	releaseHandler := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseHandler)
	s := NewHttpServer("http",
		func() string { return "127.0.0.1:0" },
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	)
	s.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(exited)
		close(started)
		<-r.Context().Done()
		close(canceled)
		// Closing the socket and canceling the request still do not prove this
		// handler has finished its application-level final work.
		<-release
	})

	var closed atomic.Bool
	a.Use(pipelineModule{
		name: "database",
		stop: func(context.Context, *app.App) error {
			t.Error("database stopped with a live HTTP handler")
			return nil
		},
		close: func(context.Context, *app.App) error {
			closed.Store(true)
			return nil
		},
	}, app.WithDependencies(s, "database"))
	for _, stage := range []app.Stage{app.StageCleanup, app.StageExited} {
		a.On(stage, "retain", func(context.Context) error {
			t.Errorf("%s ran with a live handler", stage)
			return nil
		})
	}
	a.Finalize("logger", func(context.Context) error {
		t.Error("logger closed with a live handler")
		return nil
	})

	transport := &http.Transport{DisableKeepAlives: true}
	defer transport.CloseIdleConnections()

	client := &http.Client{Transport: transport, Timeout: time.Second}
	requestDone := make(chan struct{})
	var requestErr error
	a.On(app.StageReady, "request", func(ctx context.Context) error {
		go func() {
			defer close(requestDone)
			{
				url := "http://" + s.listen.Addr().String()
				response, err := client.Get(url)
				requestErr = err
				if err == nil {
					_ = response.Body.Close()
				}
			}
		}()
		select {
		case <-started:
		case <-ctx.Done():
			return ctx.Err()
		}
		a.Stop()
		return nil
	})

	err := a.Run(parent)
	if !errors.Is(err, app.ErrShutdownIncomplete) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run = %v", err)
	}
	if closed.Load() {
		t.Fatal("database closed while an HTTP handler was still active")
	}

	select {
	case <-canceled:
	case <-parent.Done():
		t.Fatal("HTTP timeout did not cancel the request")
	}

	select {
	case <-s.shutdownDone:
	case <-parent.Done():
		t.Fatal("HTTP shutdown did not finish closing connections")
	}

	select {
	case <-requestDone:
		if requestErr == nil {
			t.Fatal("request succeeded while its handler was still blocked")
		}
		var networkError net.Error
		if errors.As(requestErr, &networkError) && networkError.Timeout() {
			t.Fatalf("request timed out instead of being forcibly closed: %v", requestErr)
		}
	case <-parent.Done():
		t.Fatal("forced shutdown did not close the active connection")
	}

	wait, cancelWait := context.WithCancel(context.Background())
	cancelWait()
	if err := s.Wait(wait, a); err != context.Canceled {
		t.Fatalf("Wait returned before the remaining handler exited: %v", err)
	}

	select {
	case <-exited:
		t.Fatal("handler exited before explicit release")
	default:
	}

	releaseHandler()
	select {
	case <-exited:
	case <-parent.Done():
		t.Fatal("handler did not finish after release")
	}

	if err := s.Wait(parent, a); err != nil {
		t.Fatalf("Wait did not recognize the handler's eventual exit: %v", err)
	}
	if err := s.Wait(parent, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("standalone Wait lost the graceful shutdown error: %v", err)
	}
	if closed.Load() {
		t.Fatal("app must not finalize asynchronously after returning incomplete shutdown")
	}
}

func TestHTTPRejectsNewHandlersAfterShutdown(t *testing.T) {
	called := false
	s := NewHttpServer("http",
		func() string { return "127.0.0.1:0" },
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }),
	)
	if err := s.Init(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	defer s.Close(context.Background(), nil) //nolint:errcheck

	addr := s.listen.Addr().String()
	if err := s.Stop(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	{
		wait := 100 * time.Millisecond
		conn, err := net.DialTimeout("tcp", addr, wait)
		if err == nil {
			_ = conn.Close()
			t.Fatal("Stop before Start left the listener open")
		}
	}

	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if called || w.Code != http.StatusServiceUnavailable {
		t.Fatalf("called=%v, status=%d", called, w.Code)
	}
	if err := s.Wait(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.Wait(ctx, nil); err != nil {
		t.Fatalf("finished handlers must win over cancellation: %v", err)
	}
}

func TestHTTPWaitCancellationDoesNotStopRequests(t *testing.T) {
	for _, beginStop := range []bool{false, true} {
		name := "before Stop"
		if beginStop {
			name = "during grace period"
		}

		t.Run(name, func(t *testing.T) {
			a := app.New()
			a.SetSignals()
			a.SetConfigLoader(func(context.Context) error { return nil })
			ready := make(chan struct{})
			release := make(chan struct{})
			started := make(chan context.Context, 1)
			releaseRequest := sync.OnceFunc(func() { close(release) })
			s := NewHttpServer("http",
				func() string { return "127.0.0.1:0" },
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					started <- r.Context()
					<-release
					w.WriteHeader(http.StatusNoContent)
				}),
			)
			a.Use(s)
			a.On(app.StageReady, "ready", func(context.Context) error {
				close(ready)
				return nil
			})

			returned := make(chan error, 1)
			go func() { returned <- a.Run(context.Background()) }()
			t.Cleanup(func() {
				releaseRequest()
				a.Stop()
				select {
				case <-returned:
				case <-time.After(2 * time.Second):
					t.Error("app cleanup did not finish")
				}
			})

			select {
			case <-ready:
			case <-time.After(time.Second):
				t.Fatal("not ready")
			}

			responseDone := make(chan error, 1)
			go func() {
				client := &http.Client{Timeout: 2 * time.Second}
				url := "http://" + s.listen.Addr().String()
				response, err := client.Get(url)
				if response != nil {
					_ = response.Body.Close()
				}
				responseDone <- err
			}()

			var requestCtx context.Context
			select {
			case requestCtx = <-started:
			case <-time.After(time.Second):
				t.Fatal("handler did not start")
			}

			grace, cancelGrace := context.WithTimeout(context.Background(), time.Second)
			defer cancelGrace()
			if beginStop {
				if err := s.Stop(grace, a); err != nil {
					t.Fatal(err)
				}
			}

			wait, cancelWait := context.WithCancel(context.Background())
			cancelWait()
			if err := s.Wait(wait, a); !errors.Is(err, context.Canceled) {
				t.Fatalf("Wait=%v", err)
			}
			if requestCtx.Err() != nil {
				t.Error("canceling Wait canceled a live request")
			}
			if grace.Err() != nil {
				t.Error("grace period unexpectedly expired")
			}

			releaseRequest()
			if !beginStop {
				if err := s.Stop(grace, a); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.Wait(grace, a); err != nil {
				t.Fatal(err)
			}
			if err := <-responseDone; err != nil {
				t.Errorf("observer canceled the request: %v", err)
			}
		})
	}
}
