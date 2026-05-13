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

func NewHttpServer(name, addr string, handler http.Handler) app.Module {
	return &httpServer{name: name, addr: addr, handler: handler}
}

type httpServer struct {
	name string
	addr string

	handler http.Handler
	server  *http.Server
	listen  net.Listener
}

func (s *httpServer) Name() string {
	return s.name
}

func (s *httpServer) Init(ctx context.Context, a *app.App) (err error) {
	network := "tcp"
	if s.addr == "" {
		s.addr = ":http"
	}

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

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: s.handler,

		IdleTimeout:       time.Minute * 3,
		ReadHeaderTimeout: time.Second * 3,

		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	return
}

func (s *httpServer) Start(context.Context, *app.App) (err error) {
	slog.Info("start the http server", "addr", s.addr)
	_ = s.server.Serve(s.listen)
	return
}

func (s *httpServer) Stop(ctx context.Context, app *app.App) (err error) {
	slog.Info("stop the http server", "addr", s.addr)
	return s.server.Shutdown(ctx)
}
