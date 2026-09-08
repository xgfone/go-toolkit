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
	"testing"
	"time"

	"github.com/xgfone/go-toolkit/app"
)

func TestRegressionStopBeforeStartClosesListener(t *testing.T) {
	s := NewHttpServer("http", func() string { return "127.0.0.1:0" }, http.NotFoundHandler())

	a := app.New()
	if err := s.Init(context.Background(), a); err != nil {
		t.Fatal(err)
	}

	defer s.listen.Close() //nolint:errcheck
	addr := s.listen.Addr().String()
	if err := s.Stop(context.Background(), a); err != nil {
		t.Fatal(err)
	}

	c, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err == nil {
		_ = c.Close()
		t.Fatal("Stop after Init left the listener open")
	}
}

func TestRegressionServeErrorStopsApp(t *testing.T) {
	s := NewHttpServer("http", func() string { return "127.0.0.1:0" }, http.NotFoundHandler())
	s.WrapListener(func(l net.Listener) net.Listener { _ = l.Close(); return l })

	a := app.New()
	a.SetSignals()
	a.SetConfigLoader(func(context.Context, *app.App) error { return nil })
	a.Use(s)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := a.Run(ctx); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("Run must preserve the listener failure, got %v", err)
	}

	if ctx.Err() != nil {
		t.Fatal("Serve failure stopped the app only after the parent timed out")
	}
}
