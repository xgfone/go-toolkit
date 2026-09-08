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
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xgfone/go-toolkit/app"
)

type wrappedListener struct{ net.Listener }

func getAddrFunc(addr string) func() string {
	return func() string { return addr }
}

// TestNewHttpServer verifies that NewHttpServer creates a Module
// with the given name, and the returned Name() matches.
func TestNewHttpServer(t *testing.T) {
	m := NewHttpServer("test-server", getAddrFunc(":0"), http.NotFoundHandler())
	if got := m.Name(); got != "test-server" {
		t.Errorf("Name() = %q, want %q", got, "test-server")
	}
}

// TestHttpServerLifecycle exercises the full Init → Start (concurrently) → Stop
// lifecycle on a random port.
func TestHttpServerLifecycle(t *testing.T) {
	m := NewHttpServer("lifecycle", getAddrFunc(":0"), http.NotFoundHandler())
	testHttpServerLifecycle(t, m)
}

// TestHttpServerInitFail verifies that Init fails when the port is already in use.
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
	if ln, ok := m.listen.(*onceCloseListener); !ok || ln.Listener != wrapped {
		t.Fatal("Init did not use the wrapped listener")
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
	})

	if err := m.Start(context.Background(), nil); err == nil {
		t.Fatal("Start without an App must fail")
	}
}

// TestHttpServerURLScheme verifies that Init correctly parses a "tcp://..." URL
// and extracts the network and address from it.
func TestHttpServerURLScheme(t *testing.T) {
	m := NewHttpServer("scheme", getAddrFunc("tcp://:0"), http.NotFoundHandler())
	testHttpServerLifecycle(t, m)
}

func testHttpServerLifecycle(t *testing.T, m *HttpServer) {
	t.Helper()

	a := app.New()
	a.SetSignals()
	a.SetConfigLoader(func(context.Context, *app.App) error { return nil })
	a.Use(m)

	ready := false
	a.On(app.StageReady, func(ctx context.Context, a *app.App) error {
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
