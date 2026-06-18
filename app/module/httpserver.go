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
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xgfone/go-toolkit/app"
)

// NewHttpServer returns a new HttpServer instance.
//
// The addr function is called during Init to get the listen address. If addr is
// nil or returns an empty string, the HTTP server is disabled and won't start.
func NewHttpServer(name string, addr func() string, handler http.Handler) *HttpServer {
	return &HttpServer{name: name, getAddr: addr, handler: handler}
}

// HttpServer is an app module that starts an HTTP server.
type HttpServer struct {
	name string
	addr string

	getAddr func() string
	handler http.Handler
	server  *http.Server
	listen  net.Listener
	wrapln  func(net.Listener) net.Listener
}

// WrapListener registers wrap to replace the listener created by Init,
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

	s.listen, err = net.Listen(network, s.addr)
	if err != nil {
		return
	}

	if s.wrapln != nil {
		s.listen = s.wrapln(s.listen)
	}

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: s.handler,

		IdleTimeout:       time.Minute * 3,
		ReadHeaderTimeout: time.Second * 3,

		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	return
}

func (s *HttpServer) Start(context.Context, *app.App) (err error) {
	if !s.IsValid() {
		return
	}

	slog.Info("start the http server", "modname", s.name, "addr", s.addr)
	go s.server.Serve(s.listen)
	return
}

func (s *HttpServer) Stop(ctx context.Context, app *app.App) (err error) {
	if !s.IsValid() {
		return
	}

	slog.Info("stop the http server", "modname", s.name, "addr", s.addr)
	return s.server.Shutdown(ctx)
}
