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
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sync"

	"github.com/xgfone/go-toolkit/codeint"
	"github.com/xgfone/go-toolkit/mapx"
	"github.com/xgfone/go-toolkit/result"
)

var _ctxpool = sync.Pool{
	New: func() any {
		return &Context{
			Data: mapx.NewSMap[any](4),
		}
	},
}

// AcquireContext acquires a context from the pool and resets it with the given request.
func AcquireContext(w http.ResponseWriter, r *http.Request) *Context {
	c := _ctxpool.Get().(*Context)
	c.Reset(w, r)
	return c
}

// ReleaseContext releases the context back to the pool.
func ReleaseContext(c *Context) {
	c.Reset(nil, nil)
	_ctxpool.Put(c)
}

// GetContext returns the Context from the context.Context.
//
// Return nil if the context.Context does not contain a Context.
func GetContext(ctx context.Context) *Context {
	c, _ := ctx.Value(_CtxKeyType(1)).(*Context)
	return c
}

// SetContext sets the Context into the context.Context and returns a new context.Context.
func SetContext(ctx context.Context, c *Context) context.Context {
	return context.WithValue(ctx, _CtxKeyType(1), c)
}

type _CtxKeyType int

// Context is the context of the request.
type Context struct {
	context.Context
	ResponseWriter
	*http.Request

	Auth any
	Data mapx.SMap[any]

	Error error
}

// Reset resets the request context.
func (c *Context) Reset(w http.ResponseWriter, r *http.Request) {
	var ctx context.Context
	if r != nil {
		ctx = r.Context()
	}

	var rw ResponseWriter
	if w != nil {
		rw = NewResponseWriter(w)
	}

	clear(c.Data)
	*c = Context{
		Context:        ctx,
		Request:        r,
		ResponseWriter: rw,

		Data: c.Data,
	}
}

// AppendError appends the error err into c.Error.
func (c *Context) AppendError(err error) {
	if err != nil {
		if c.Error == nil {
			c.Error = err
		} else {
			c.Error = errors.Join(c.Error, err)
		}
	}
}

// SetContentType sets the response header "Content-Type" to ct,
func (c *Context) SetContentType(ct string) {
	SetContentType(c.ResponseWriter.Header(), ct)
}

// SetConnectionClose sets the response header "Content-Disposition".
// For example,
//
//	Content-Disposition: inline
//	Content-Disposition: attachment
//	Content-Disposition: attachment; filename="filename.jpg"
//
// dtype must be either "inline" or "attachment". But, filename is optional.
func (c *Context) SetContentDisposition(dtype, filename string) {
	switch dtype {
	case "inline", "attachment":
	default:
		panic(fmt.Errorf("Context.SetContentDisposition: unknown dtype '%s'", dtype))
	}

	var disposition string
	if filename == "" {
		disposition = "Content-Disposition: " + dtype
	} else {
		params := map[string]string{"filename": filename}
		disposition = mime.FormatMediaType(dtype, params)
	}

	c.ResponseWriter.Header().Set(HeaderContentDisposition, disposition)
}

// Redirect redirects the request to a provided URL with status code.
func (c *Context) Redirect(code int, toURL string) {
	if code < 300 || code >= 400 {
		panic(fmt.Errorf("invalid the redirect status code '%d'", code))
	}

	c.ResponseWriter.Header().Set(HeaderLocation, toURL)
	c.WriteHeader(code)
}

// NoContent is the alias of WriteHeader.
func (c *Context) NoContent(code int) { c.WriteHeader(code) }

// JSON sends a JSON response with the status code.
func (c *Context) JSON(code int, v any) {
	c.AppendError(JSON(c.ResponseWriter, code, v))
}

// Stream sends a streaming response with the status code.
func (c *Context) Stream(code int, r io.Reader) {
	c.WriteHeader(code)
	_, err := io.Copy(c.ResponseWriter, r)
	c.AppendError(err)
}

// Success sends the success response with data.
func (c *Context) Success(data any) {
	result.Success(c, data)
}

// Respond implements the interface result.Responder.
func (c *Context) Respond(response result.Response) {
	respond(c, response)
}

var respond func(*Context, result.Response)

func init() { SetRespond(defaultRespond) }

// SetRespond sets the respond function.
func SetRespond(f func(*Context, result.Response)) {
	if f == nil {
		panic("SetRespond: the respond function cannot be nil")
	}
	respond = f
}

func defaultRespond(c *Context, response result.Response) {
	if response.Error != nil {
		respondError(c, response)
	} else if response.Data != nil {
		c.JSON(200, response)
	} else {
		c.NoContent(200)
	}
}

func respondError(c *Context, response result.Response) {
	statuscode := 500

	switch e := response.Error.(type) {
	case codeint.Error:
		statuscode = e.StatusCode()

	case *codeint.Error:
		statuscode = e.StatusCode()

	case interface{ StatusCode() int }:
		statuscode = e.StatusCode()
		response.Error = codeint.ErrInternalServerError.WithError(response.Error)

	default:
		response.Error = codeint.ErrInternalServerError.WithError(response.Error)
	}

	c.JSON(statuscode, response)
}
