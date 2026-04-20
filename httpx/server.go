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

package httpx

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var serve = defaultServe

func defaultServe(l net.Listener, s *http.Server) {
	s.Serve(l)
}

// SetServeFunc sets the serve function that be used when starting the server.
//
// Default: server.Serve(ln)
func SetServeFunc(f func(ln net.Listener, server *http.Server)) {
	if f == nil {
		panic("httpx.SetServeFunc: the serve function is nil")
	}
	serve = f
}

// StartServer starts the http server with the address and handler.
//
// If the handler implements the interface{ Start(addr string) },
// it will be called directly. Otherwise, a new http.Server will
// be created and served.
func StartServer(addr string, handler http.Handler) {
	if h, ok := handler.(interface{ Start(string) }); ok {
		h.Start(addr)
	} else {
		startServer(addr, handler)
	}
}

func startServer(addr string, handler http.Handler) {
	network := "tcp"
	if addr == "" {
		addr = ":http"
	}

	if strings.Contains(addr, "://") {
		if u, err := url.Parse(addr); err == nil && u.Scheme != "" {
			network = u.Scheme
			addr = u.Host
		}
	}

	ln, err := net.Listen(network, addr)
	if err != nil {
		slog.Error("fail to open the listener on the address",
			"network", network, "addr", addr, "err", err)
		return
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,

		IdleTimeout:       time.Minute * 3,
		ReadHeaderTimeout: time.Second * 3,

		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}

	slog.Info("start the http server", "addr", addr)
	defer slog.Info("stop the http server", "addr", addr)

	serve(ln, server)
}
