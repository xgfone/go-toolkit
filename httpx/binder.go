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
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"

	"github.com/xgfone/go-toolkit/codeint"
	"github.com/xgfone/go-toolkit/jsonx"
	"github.com/xgfone/go-toolkit/structx"
	"github.com/xgfone/go-toolkit/validation"
)

const (
	bindTagForm   = "form"
	bindTagHeader = "header"
	bindTagQuery  = "query"
	bindTagPath   = "path"
)

var errNilRequest = errors.New("httpx: request is nil")

// BindBody binds the request body into dst according to the request
// Content-Type, then sets defaults and validates dst.
//
// For JSON and XML bodies, BindBody uses the standard struct tags "json" and
// "xml". For form and multipart form bodies, it uses the "form" struct tag.
func BindBody[T any](r *http.Request, dst *T) error {
	return bindBodyRequest(r, dst)
}

func bindBodyRequest[T any](r *http.Request, dst *T) error {
	if r == nil {
		return errNilRequest
	}

	switch ct := ContentType(r.Header); ct {
	case "":
		return codeint.ErrMissingContentType

	case MIMEApplicationJSON:
		if err := jsonx.UnmarshalReader(dst, r.Body); err != nil {
			return err
		}

	case MIMEApplicationXML, MIMETextXML:
		if err := xml.NewDecoder(r.Body).Decode(dst); err != nil {
			return err
		}

	case MIMEApplicationForm:
		if err := r.ParseForm(); err != nil {
			return err
		}
		if err := bindForm(dst, r.PostForm); err != nil {
			return err
		}

	case MIMEMultipartForm:
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return err
		}
		if r.MultipartForm != nil {
			if err := bindForm(dst, r.MultipartForm.Value); err != nil {
				return err
			}
		}

	default:
		return codeint.ErrUnsupportedMediaType.WithReasonf("unsupported Content-Type %q", ct)
	}

	return defaultAndValidate(dst)
}

// BindHeader binds the request headers into dst using the "header" struct tag,
// then sets defaults and validates dst.
func BindHeader[T any](r *http.Request, dst *T) error {
	return bindHeaderRequest(r, dst)
}

func bindHeaderRequest[T any](r *http.Request, dst *T) error {
	if r == nil {
		return errNilRequest
	}

	if err := bindHeader(dst, r.Header); err != nil {
		return err
	}

	return defaultAndValidate(dst)
}

// BindQuery binds the request query parameters into dst using the "query"
// struct tag, then sets defaults and validates dst.
func BindQuery[T any](r *http.Request, dst *T) error {
	return bindQueryRequest(r, dst)
}

func bindQueryRequest[T any](r *http.Request, dst *T) error {
	if r == nil {
		return errNilRequest
	}

	if err := bindQuery(dst, r.URL.Query()); err != nil {
		return err
	}

	return defaultAndValidate(dst)
}

// BindPath binds the request's path wildcard values into dst using the "path"
// struct tag, then sets defaults and validates dst.
//
// Values come from r.PathValue, as populated by http.ServeMux or r.SetPathValue.
// Missing and empty values are ignored, as in BindQuery. Use validation to
// require a value. Query parameters and the request body are not read.
func BindPath[T any](r *http.Request, dst *T) error {
	return bindPathRequest(r, dst)
}

func bindPathRequest[T any](r *http.Request, dst *T) error {
	if r == nil {
		return errNilRequest
	}

	if err := structx.BindValues(dst, pathValues{r}, bindTagPath); err != nil {
		return err
	}

	return defaultAndValidate(dst)
}

type pathValues struct{ request *http.Request }

func (p pathValues) Get(name string) string {
	return p.request.PathValue(name)
}

func bindForm[T any](dst *T, form url.Values) error {
	return structx.BindValues(dst, form, bindTagForm)
}

func bindHeader[T any](dst *T, header http.Header) error {
	return structx.BindValues(dst, header, bindTagHeader)
}

func bindQuery[T any](dst *T, query url.Values) error {
	return structx.BindValues(dst, query, bindTagQuery)
}

func defaultAndValidate[T any](dst *T) error {
	if err := structx.SetDefault(dst); err != nil {
		return err
	}
	return validation.Validate(dst)
}
