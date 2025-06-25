// Copyright 2025 xgfone
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
	"net/http"
	"sync/atomic"
)

var _defaultclient atomic.Value

type _Client struct{ Client }

func init() {
	SetClient(http.DefaultClient)
}

// SetClient resets the default http client.
//
// Default: http.DefaultClient
func SetClient(client Client) {
	if client == nil {
		panic("httpx.SetClient: client must not be nil")
	}
	_defaultclient.Store(_Client{client})
}

// GetClient returns the default http client.
func GetClient() Client {
	return _defaultclient.Load().(_Client).Client
}

// Client is a http client interface that sends a http request and returns a http response.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// DoFunc is a function that sends a http request and returns a http response.
type DoFunc func(req *http.Request) (*http.Response, error)

// Do implements the Client interface.
func (f DoFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

type wclient struct {
	client Client
	wrapf  func(Client, *http.Request) (*http.Response, error)
}

func (c *wclient) Unwrap() Client                             { return c.client }
func (c *wclient) Do(r *http.Request) (*http.Response, error) { return c.wrapf(c.client, r) }

// WrapClient wraps the client to handler the http request and
// returns a new Client that has implemented the interface { Unwrap() Client }
// to unwrap the inner client.
func WrapClient(c Client, f func(Client, *http.Request) (*http.Response, error)) Client {
	return &wclient{client: c, wrapf: f}
}

// UnwrapClient unwraps and returns the inner client.
//
// Return nil if client has not implemented the interface { Unwrap() Client }.
func UnwrapClient(client Client) Client {
	if c, ok := client.(interface{ Unwrap() Client }); ok {
		return c.Unwrap()
	}
	return nil
}
