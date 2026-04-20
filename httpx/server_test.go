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
	"net"
	"net/http"
	"testing"
	"time"
)

func TestDefaultServe(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	server := &http.Server{}
	go func() {
		time.Sleep(time.Millisecond * 100)
		server.Close()
	}()
	defaultServe(ln, server)
}

func TestSetServeFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil serve function")
		}
		// Restore default
		SetServeFunc(func(l net.Listener, s *http.Server) { s.Serve(l) })
	}()

	SetServeFunc(nil)
}

func TestStartServerNetworkError(t *testing.T) {
	SetServeFunc(func(net.Listener, *http.Server) {})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	StartServer("invalid-address-format", handler)
}

func TestStartServerRegularHandler(t *testing.T) {
	serveCalled := false
	SetServeFunc(func(net.Listener, *http.Server) {
		serveCalled = true
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	StartServer(":0", handler)

	if !serveCalled {
		t.Error("serve function should have been called")
	}
}

func TestStartServerAddressParsing(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{"empty address", ""},
		{"port only", ":0"},
		{"host and port", "localhost:0"},
		{"URL with scheme", "tcp://localhost:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serveCalled := false
			SetServeFunc(func(net.Listener, *http.Server) {
				serveCalled = true
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

			StartServer(tt.addr, handler)

			if !serveCalled {
				t.Error("serve function should have been called")
			}
		})
	}
}

func TestStartServerWithStartInterface(t *testing.T) {
	handler := &mockHandlerWithStart{}

	SetServeFunc(func(net.Listener, *http.Server) {})
	StartServer(":8080", handler)

	if !handler.startCalled {
		t.Error("handler.Start should have been called")
	}
	if handler.startAddr != ":8080" {
		t.Errorf("expected address ':8080', got '%s'", handler.startAddr)
	}
}

type mockHandlerWithStart struct {
	startCalled bool
	startAddr   string
}

func (m *mockHandlerWithStart) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func (m *mockHandlerWithStart) Start(addr string) {
	m.startCalled = true
	m.startAddr = addr
}
