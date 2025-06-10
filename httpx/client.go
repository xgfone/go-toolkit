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
	"context"
	"io"
	"net/http"
	"sync/atomic"

	"github.com/xgfone/go-toolkit/httpx/option"
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

// Doer is the alias of Client.
//
// Deprecated.
type Doer = Client

// Client is a http client interface that sends a http request and returns a http response.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// DoFunc is a function that sends a http request and returns a http response.
type DoFunc func(req *http.Request) (*http.Response, error)

// Do implements the Doer interface.
func (f DoFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

// WrapClientWithOptions wraps the original client and returns a new one
// that configures the http request with the given options.
func WrapClientWithOptions(client Client, options ...option.Option) Client {
	return DoFunc(func(req *http.Request) (*http.Response, error) {
		return client.Do(option.Apply(req, options...))
	})
}

// Get sends a http GET request.
func Get(ctx context.Context, url string, do func(*http.Response) error, options ...option.Option) error {
	return request(ctx, http.MethodGet, url, nil, do, options...)
}

// Get sends a http POST request.
func Post(ctx context.Context, url string, body io.Reader, do func(*http.Response) error, options ...option.Option) error {
	return request(ctx, http.MethodGet, url, body, do, options...)
}

func request(ctx context.Context, method, url string, body io.Reader,
	do func(*http.Response) error, options ...option.Option) (err error) {

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return
	}

	req = option.Apply(req, options...)
	resp, err := GetClient().Do(req)
	if err != nil {
		return
	}

	return do(resp)
}
