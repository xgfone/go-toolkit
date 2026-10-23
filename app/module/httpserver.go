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
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xgfone/go-toolkit/app"
)

var _ app.Module = (*HttpServer)(nil)

// NewHttpServer returns a new [HttpServer] instance.
//
// The addr function is called during [HttpServer.Init] to get the listen address.
// If addr is nil or returns an empty string, the HTTP server is disabled and won't
// start. A nil handler uses the [http.DefaultServeMux] captured during Init.
// Configure or replace the default mux before [app.App.Run]; do not reassign
// http.DefaultServeMux while the app is running.
func NewHttpServer(name string, addr func() string, handler http.Handler) *HttpServer {
	return &HttpServer{name: name, getAddr: addr, handler: handler}
}

// HttpServer is an [app.Module] that starts an HTTP server.
type HttpServer struct {
	name string
	addr string

	getAddr func() string
	handler http.Handler
	server  *http.Server
	listen  net.Listener
	wrapln  func(net.Listener) net.Listener

	requests       *requestGroup
	cancelRequests context.CancelFunc
	stopOnce       sync.Once
	shutdownDone   chan struct{}
	shutdownErr    error
	stopErr        error
}

// onceCloseListener shares one close operation between [http.Server.Serve],
// [HttpServer.Stop], and [HttpServer.Close], including rollback before
// [http.Server.Serve] has registered the listener. Accept errors after Close
// begins become ErrServerClosed, including custom errors from wrapped listeners.
type onceCloseListener struct {
	net.Listener
	once sync.Once
	err  error

	closing atomic.Bool
}

func (l *onceCloseListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil && l.closing.Load() {
		err = http.ErrServerClosed
	}
	return conn, err
}

func (l *onceCloseListener) Close() error {
	l.once.Do(func() {
		l.closing.Store(true)
		l.err = l.Listener.Close()
	})
	return l.err
}

// WrapListener registers wrap to replace the listener created by [HttpServer.Init],
// which must be called before app runs.
func (s *HttpServer) WrapListener(wrap func(net.Listener) net.Listener) {
	s.wrapln = wrap
}

// IsValid reports whether the http server is valid.
func (s *HttpServer) IsValid() bool {
	return s.addr != ""
}

func (s *HttpServer) Name() string {
	return s.name
}

func (s *HttpServer) Init(ctx context.Context, a *app.App) (err error) {
	var listener net.Listener
	initialized := false
	defer func() {
		if !initialized {
			// Init owns the raw socket even if wrapping fails or panics.
			if listener != nil {
				_ = listener.Close()
			}
			if s.cancelRequests != nil {
				s.cancelRequests()
			}

			s.addr = ""
			s.listen = nil
			s.server = nil
			s.requests = nil
		}
	}()

	if s.getAddr != nil {
		s.addr = s.getAddr()
	}

	if s.addr == "" {
		return
	}

	network := "tcp"
	if strings.Contains(s.addr, "://") {
		if u, err := url.Parse(s.addr); err == nil && u.Scheme != "" {
			network = u.Scheme
			s.addr = u.Host
		}
	}

	listener, err = net.Listen(network, s.addr)
	if err != nil {
		return err
	}
	s.listen = listener

	if s.wrapln != nil {
		s.listen = s.wrapln(s.listen)
		if s.listen == nil {
			return errors.New("http server: WrapListener returned nil")
		}
	}

	s.listen = &onceCloseListener{Listener: s.listen}
	s.shutdownDone = make(chan struct{})
	s.requests = &requestGroup{done: make(chan struct{})}
	requestCtx, cancelRequests := context.WithCancel(context.WithoutCancel(ctx))
	s.cancelRequests = cancelRequests
	handler := s.handler
	if handler == nil {
		handler = http.DefaultServeMux
	}

	s.server = &http.Server{
		Addr: s.addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !s.requests.enter() {
				http.Error(w, "server shutting down", http.StatusServiceUnavailable)
				return
			}
			defer s.requests.leave()
			handler.ServeHTTP(w, r)
		}),
		BaseContext: func(net.Listener) context.Context {
			return requestCtx
		},

		IdleTimeout:       time.Minute * 3,
		ReadHeaderTimeout: time.Second * 3,

		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	initialized = true
	return
}

// Prepare completes business preparation before any modules start.
func (s *HttpServer) Prepare(context.Context, *app.App) error { return nil }

// Start runs [http.Server.Serve] in the server module's [app.TaskGroup].
// An enabled server requires a running [app.App].
func (s *HttpServer) Start(_ context.Context, a *app.App) (err error) {
	if !s.IsValid() {
		return
	}
	if a == nil {
		return errors.New("http server: Start requires a running app")
	}

	slog.Info("start the http server", "modname", s.name, "addr", s.addr)
	a.TaskGroup(s.name).Go("serve", func(context.Context) error {
		err := s.server.Serve(s.listen)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	})
	return
}

// Stop gates new requests and starts graceful HTTP shutdown without waiting.
// On drain timeout, request contexts and active connections are closed. Wait
// must still prove that handlers returned; closing sockets cannot kill them.
// Detached work and hijacked connections must be explicitly owned and joined.
func (s *HttpServer) Stop(ctx context.Context, a *app.App) error {
	if !s.IsValid() {
		return nil
	}

	s.stopOnce.Do(func() {
		if s.shutdownDone == nil {
			s.shutdownDone = make(chan struct{})
		}
		if s.requests != nil {
			s.requests.stop()
		}

		slog.Info("stop the http server", "modname", s.name, "addr", s.addr)

		// Also closes a listener acquired by Init but not yet owned by Serve.
		s.stopErr = s.listen.Close()
		if errors.Is(s.stopErr, net.ErrClosed) {
			s.stopErr = nil
		}

		go func() {
			defer close(s.shutdownDone)
			defer func() {
				if s.shutdownErr != nil && a != nil {
					a.ReportError(fmt.Errorf("http server %q: %w", s.name, s.shutdownErr))
				}
			}()

			s.shutdownErr = s.server.Shutdown(ctx)
			if s.shutdownErr != nil {
				if s.cancelRequests != nil {
					s.cancelRequests()
				}

				if err := s.server.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
					s.shutdownErr = errors.Join(s.shutdownErr, err)
				}
			}
		}()
	})
	return s.stopErr
}

// Wait joins the graceful shutdown operation and all admitted HTTP handlers.
// The app separately joins Serve through the module's task group. Canceling
// this wait does not shorten the grace budget supplied to Stop. A completed
// handler set takes precedence over wait-context cancellation. A graceful-close
// error is reported separately; proven live work prevents dependencies from
// being released even after connections have been closed.
func (s *HttpServer) Wait(ctx context.Context, a *app.App) error {
	if !s.IsValid() {
		return nil
	}
	if s.shutdownDone == nil {
		return errors.New("http server: Wait requires Stop")
	}

	if err := waitHTTPShutdown(ctx, s.shutdownDone); err != nil {
		return err
	}

	if s.requests != nil {
		if err := s.requests.wait(ctx); err != nil {
			return err
		}
	}

	if s.cancelRequests != nil {
		s.cancelRequests()
	}

	// Grace errors were reported once by the shutdown operation; nil here
	// proves all users ended and permits safe dependency release.
	if a == nil {
		return s.shutdownErr
	}

	return nil
}

// Close releases remaining server resources after Wait succeeds.
func (s *HttpServer) Close(ctx context.Context, a *app.App) error {
	if !s.IsValid() {
		return nil
	}
	if s.cancelRequests != nil {
		s.cancelRequests()
	}

	if err := s.listen.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}

	return nil
}
