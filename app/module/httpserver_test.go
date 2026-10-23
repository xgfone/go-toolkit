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
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xgfone/go-toolkit/app"
)

type wrappedListener struct{ net.Listener }

type failedAcceptListener struct {
	net.Listener
	err error
}

func (l failedAcceptListener) Accept() (net.Conn, error) { return nil, l.err }

func getAddrFunc(addr string) func() string {
	return func() string { return addr }
}

func TestHttpServerLifecycle(t *testing.T) {
	for _, tc := range []struct{ name, addr string }{
		{"plain", "127.0.0.1:0"},
		{"tcp URL", "tcp://127.0.0.1:0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewHttpServer(tc.name, getAddrFunc(tc.addr), http.NotFoundHandler())
			if got := m.Name(); got != tc.name {
				t.Fatalf("Name=%q, want %q", got, tc.name)
			}
			testHttpServerLifecycle(t, m)
		})
	}
}

func TestHttpServerInitFail(t *testing.T) {
	// Listen on a random port and hold it.
	holder := httptest.NewUnstartedServer(http.NotFoundHandler())
	holder.Start()
	defer holder.Close()

	m := NewHttpServer("conflict", getAddrFunc(holder.Listener.Addr().String()), http.NotFoundHandler())
	if err := m.Init(context.Background(), nil); err == nil {
		t.Error("expected Init to fail on a busy port, but got nil")
	}
}

func TestHttpServerWrapListener(t *testing.T) {
	m := NewHttpServer("wrap", getAddrFunc(":0"), http.NotFoundHandler())
	var captured net.Listener
	wrapped := new(wrappedListener)
	m.WrapListener(func(ln net.Listener) net.Listener {
		captured = ln
		wrapped.Listener = ln
		return wrapped
	})

	if err := m.Init(context.Background(), nil); err != nil {
		t.Fatalf("Init: unexpected error: %v", err)
	}
	defer m.listen.Close() //nolint:errcheck

	if captured == nil {
		t.Fatal("WrapListener was not called")
	}
	{
		ln, ok := m.listen.(*onceCloseListener)
		if !ok || ln.Listener != wrapped {
			t.Fatal("Init did not use the wrapped listener")
		}
	}
}

func TestHttpServerWaitRejectsIncompleteInit(t *testing.T) {
	s := NewHttpServer("http", getAddrFunc("127.0.0.1:0"), http.NotFoundHandler())
	wrapped := false
	s.WrapListener(func(listener net.Listener) net.Listener {
		wrapped = true
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// The listener exists, but Init has not set up shutdown tracking yet.
		// A reentrant Wait must fail promptly without disrupting startup.
		err := s.Wait(ctx, nil)
		if err == nil || err.Error() != "http server: Wait requires Stop" {
			t.Errorf("Wait during Init = %v, want a lifecycle error", err)
		}
		if ctx.Err() != nil {
			t.Error("Wait blocked on incomplete initialization")
		}

		return listener
	})

	testHttpServerLifecycle(t, s)
	if !wrapped {
		t.Fatal("WrapListener was not called")
	}
}

func TestHttpServerCapturesHandlerDuringInit(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name := "default handler"
		if explicit {
			name = "explicit handler"
		}
		t.Run(name, func(t *testing.T) {
			original := http.DefaultServeMux
			t.Cleanup(func() { http.DefaultServeMux = original })
			http.DefaultServeMux = http.NewServeMux()

			var handler http.Handler
			if explicit {
				handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				})
			}
			s := NewHttpServer("http", getAddrFunc("127.0.0.1:0"), handler)
			mux := http.NewServeMux()
			mux.HandleFunc("/configured", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusCreated)
			})
			// Configure the default mux after construction, before Init/Run.
			http.DefaultServeMux = mux
			if err := s.Init(context.Background(), nil); err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				if err := s.Stop(ctx, nil); err != nil {
					t.Error(err)
				}
				if err := s.Wait(ctx, nil); err != nil {
					t.Error(err)
				}
				if err := s.Close(ctx, nil); err != nil {
					t.Error(err)
				}
			})

			// No server is running. Replace the global only to verify that Init
			// captured its handler and requests do not read the global again.
			replacement := http.NewServeMux()
			replacement.HandleFunc("/configured", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			})
			http.DefaultServeMux = replacement

			w := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/configured", nil)
			s.server.Handler.ServeHTTP(w, request)
			want := http.StatusCreated
			if explicit {
				want = http.StatusNoContent
			}
			if w.Code != want {
				t.Fatalf("status = %d, want %d", w.Code, want)
			}
		})
	}
}

func TestHttpServerInvalidAddr(t *testing.T) {
	tests := []struct {
		name string
		addr func() string
	}{
		{name: "nil"},
		{name: "empty", addr: getAddrFunc("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewHttpServer(tt.name, tt.addr, http.NotFoundHandler())
			if err := m.Init(context.Background(), nil); err != nil {
				t.Fatalf("Init: unexpected error: %v", err)
			}
			if m.IsValid() {
				t.Fatal("server should be invalid")
			}
			if m.listen != nil || m.server != nil {
				t.Fatal("invalid server should not create a listener or server")
			}
			if err := m.Start(context.Background(), nil); err != nil {
				t.Fatalf("Start: unexpected error: %v", err)
			}
			if err := m.Stop(context.Background(), nil); err != nil {
				t.Fatalf("Stop: unexpected error: %v", err)
			}
			if err := m.Wait(context.Background(), nil); err != nil {
				t.Fatalf("Wait: unexpected error: %v", err)
			}
			if err := m.Close(context.Background(), nil); err != nil {
				t.Fatalf("Close: unexpected error: %v", err)
			}
		})
	}
}

func TestHttpServerStartRequiresApp(t *testing.T) {
	m := NewHttpServer("nil-app", getAddrFunc("127.0.0.1:0"), http.NotFoundHandler())
	if err := m.Init(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := m.Stop(context.Background(), nil); err != nil {
			t.Errorf("Stop after rejected Start: %v", err)
		}
		if err := m.Wait(context.Background(), nil); err != nil {
			t.Errorf("Wait after rejected Start: %v", err)
		}
		if err := m.Close(context.Background(), nil); err != nil {
			t.Errorf("Close after rejected Start: %v", err)
		}
	})

	if err := m.Start(context.Background(), nil); err == nil {
		t.Fatal("Start without an App must fail")
	}
}

func testHttpServerLifecycle(t *testing.T, m *HttpServer) {
	t.Helper()

	a := app.New()
	a.SetSignals()
	a.SetConfigLoader(func(context.Context) error { return nil })
	a.Use(m)

	ready := false
	a.On(app.StageReady, "", func(ctx context.Context) error {
		ready = true
		if !m.IsValid() {
			t.Error("server should be valid after Init")
		}

		addr := m.listen.Addr().(*net.TCPAddr)
		url := "http://" + net.JoinHostPort("127.0.0.1", fmt.Sprint(addr.Port))
		client := &http.Client{Timeout: time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return err
		}

		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("unexpected HTTP response: %d", resp.StatusCode)
		}

		a.Stop()
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := a.Run(ctx); err != nil {
		t.Fatalf("HTTP lifecycle failed: %v", err)
	}

	if !ready {
		t.Fatal("HTTP server never became ready")
	}
}

func TestHTTPInitFailureReleasesListener(t *testing.T) {
	for _, panics := range []bool{false, true} {
		name := "nil wrapper"
		if panics {
			name = "panicking wrapper"
		}

		t.Run(name, func(t *testing.T) {
			s := NewHttpServer("http",
				func() string { return "127.0.0.1:0" },
				http.NotFoundHandler(),
			)

			var raw net.Listener
			s.WrapListener(func(l net.Listener) net.Listener {
				raw = l
				if panics {
					panic("wrapper failed")
				}
				return nil
			})

			func() {
				defer func() {
					if p := recover(); p != nil && !panics {
						t.Errorf("unexpected panic: %v", p)
					}
				}()

				if err := s.Init(context.Background(), nil); err == nil && !panics {
					t.Error("nil wrapped listener accepted")
				}
			}()

			if s.IsValid() {
				t.Error("failed Init left the server valid")
			}
			if err := s.Stop(context.Background(), nil); err != nil {
				t.Error(err)
			}
			if err := s.Close(context.Background(), nil); err != nil {
				t.Error(err)
			}
			if raw == nil {
				t.Fatal("listener was not created")
			}
			defer raw.Close() //nolint:errcheck

			addr := raw.Addr().String()
			wait := 30 * time.Millisecond
			c, err := net.DialTimeout("tcp", addr, wait)
			if err == nil {
				_ = c.Close()
				t.Error("failed Init leaked its listening socket")
			}
		})
	}
}

func TestRegressionServeErrorStopsApp(t *testing.T) {
	for _, failure := range []error{
		net.ErrClosed,
		errors.New("wrapped listener failed"),
	} {
		t.Run(failure.Error(), func(t *testing.T) {
			s := NewHttpServer("http", getAddrFunc("127.0.0.1:0"), http.NotFoundHandler())
			s.WrapListener(func(l net.Listener) net.Listener {
				_ = l.Close()
				return failedAcceptListener{Listener: l, err: failure}
			})

			a := app.New()
			a.SetSignals()
			a.SetConfigLoader(func(context.Context) error { return nil })
			a.Use(s)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := a.Run(ctx); !errors.Is(err, failure) {
				t.Fatalf("Run must preserve the listener failure, got %v", err)
			}

			if ctx.Err() != nil {
				t.Fatal("Serve failure stopped the app only after the parent timed out")
			}
		})
	}
}
