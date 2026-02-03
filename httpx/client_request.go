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
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/xgfone/go-toolkit/internal/pools"
	"github.com/xgfone/go-toolkit/jsonx"
	"github.com/xgfone/go-toolkit/unsafex"
)

type _ClientError struct {
	req *http.Request  `json:"-"`
	rsp *http.Response `json:"-"`

	code int
	body string
	err  error
}

func newClientError(req *http.Request, rsp *http.Response) _ClientError {
	return _ClientError{req: req, rsp: rsp, code: rsp.StatusCode}
}

func (e _ClientError) WithError(err error) _ClientError {
	e.err = err
	return e
}

func (e _ClientError) WithBody(body []byte) _ClientError {
	e.body = unsafex.String(body)
	return e
}

func (e _ClientError) Request() *http.Request   { return e.req }
func (e _ClientError) Response() *http.Response { return e.rsp }
func (e _ClientError) ResponseBody() string     { return e.body }
func (e _ClientError) StatusCode() int          { return e.code }

func (e _ClientError) Unwrap() error { return e.err }
func (e _ClientError) Error() string {
	switch {
	case e.err != nil && e.body != "":
		return fmt.Sprintf("statuscode=%d, body=%s, err=%v", e.code, e.body, e.err)

	case e.err != nil:
		return fmt.Sprintf("statuscode=%d, err=%v", e.code, e.err)

	case e.body != "":
		return fmt.Sprintf("statuscode=%d, body=%s", e.code, e.body)

	default:
		return fmt.Sprintf("statuscode=%d", e.code)
	}
}

// Get sends a GET request to the specified URL and decodes the response body
// into the provided response object.
//
// The respbody parameter supports the following types:
//   - nil: response body is ignored, only HTTP status code 200 is checked
//   - func(*http.Response) error: custom response handler function
//   - any other type: response body is automatically decoded as JSON into the variable
func Get(ctx context.Context, url string, respbody any) (err error) {
	return request(ctx, http.MethodGet, url, respbody, nil)
}

// Post sends a POST request to the specified URL with the provided request body
// and decodes the response body into the provided response object.
//
// The respbody parameter supports the following types:
//   - nil: response body is ignored, only HTTP status code 200 is checked
//   - func(*http.Response) error: custom response handler function
//   - any other type: response body is automatically decoded as JSON into the variable
//
// The reqbody parameter supports the following types:
//   - nil: no request body will be sent
//   - io.Reader: used directly as the request body
//   - any other type: automatically encoded as JSON
//
// If reqbody is not nil, it will set the Content-Type header to "application/json".
func Post(ctx context.Context, url string, respbody any, reqbody any) (err error) {
	return request(ctx, http.MethodPost, url, respbody, reqbody)
}

func request(ctx context.Context, method, url string, resp, req any) (err error) {
	var body io.Reader
	switch r := req.(type) {
	case nil:

	case io.Reader:
		body = r

	default:
		pool, buf := pools.GetBuffer(1024)
		defer pools.PutBuffer(pool, buf)

		if err = jsonx.MarshalWriter(buf, r); err != nil {
			return fmt.Errorf("fail to encode request body: %w", err)
		}

		body = buf
	}

	_req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return
	}

	if req != nil {
		SetContentType(_req.Header, MIMEApplicationJSON)
	}

	_rsp, err := GetClient().Do(_req)
	if err != nil {
		return err
	}
	defer _rsp.Body.Close()

	if f, ok := resp.(func(*http.Response) error); ok {
		return f(_rsp)
	}

	data, err := io.ReadAll(_rsp.Body)
	if err != nil {
		err = fmt.Errorf("fail to read the response body: %w", err)
		return newClientError(_req, _rsp).WithError(err)
	}

	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		var reqbody any
		if _, ok := req.(io.Reader); !ok {
			reqbody = req
		}

		slog.Debug("log http response",
			"method", _req.Method,
			"url", _req.URL.String(),
			"reqheader", _req.Header,
			"reqbody", reqbody,
			"statuscode", _rsp.StatusCode,
			"respheader", _rsp.Header,
			"respbody", unsafex.String(data))
	}

	if _rsp.StatusCode != 200 {
		return newClientError(_req, _rsp).WithBody(data)
	}

	if resp != nil {
		if err = jsonx.UnmarshalBytes(data, &resp); err != nil {
			err = fmt.Errorf("fail to decode the response body: %w", err)
			return newClientError(_req, _rsp).WithBody(data).WithError(err)
		}
	}

	return
}
