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

// Package cors provides a CORS implementation.
package cors

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/xgfone/go-toolkit/slicex"
)

// HostNormalizer normalizes a non-IP origin host. It may be used to apply
// optional features such as IDNA ToASCII outside this package.
//
// It must return a host without a port or brackets. Returning false rejects it.
type HostNormalizer func(host string) (normalized string, ok bool)

// Config is used to configure CORS.
type Config struct {
	// AllowOrigins defines a list of origins that may access the resource.
	// Each origin may be "*", a serialized origin such as "https://example.com",
	// "null", or a subdomain pattern such as "https://*.example.com".
	//
	// Optional. Default: nil, which does not allow any cross-origin request.
	AllowOrigins []string `json:"allowOrigins" yaml:"allowOrigins"`

	// AllowHeaders indicates a list of request headers used in response to
	// a preflight request to indicate which HTTP headers can be used when
	// making the actual request.
	//
	// If empty, all valid requested headers are reflected, so the preflight
	// succeeds for any non-safelisted request header the browser asks to use.
	// Prefer an explicit allow-list for stricter APIs.
	//
	// If AllowCredentials is true, "*" is also reflected from the request
	// because browsers do not treat it as a wildcard for credentialed requests.
	//
	// Optional. Default: nil.
	AllowHeaders []string `json:"allowHeaders" yaml:"allowHeaders"`

	// AllowMethods indicates methods allowed when accessing the resource.
	// This is used in response to a preflight request. If AllowCredentials is
	// true, "*" is reflected from the request because browsers do not treat it
	// as a wildcard for credentialed requests.
	//
	// Optional. Default: nil.
	AllowMethods []string `json:"allowMethods" yaml:"allowMethods"`

	// ExposeHeaders indicates response headers browsers are allowed to access
	// from an actual CORS response. If AllowCredentials is true, "*" is omitted
	// because browsers do not treat it as a wildcard for credentialed requests.
	//
	// Optional. Default: nil.
	ExposeHeaders []string `json:"exposeHeaders" yaml:"exposeHeaders"`

	// AllowCredentials indicates whether or not the response to the request
	// can be exposed when the credentials flag is true. When used as part of
	// a response to a preflight request, this indicates whether or not the
	// actual request can be made using credentials.
	//
	// Optional. Default: false.
	AllowCredentials bool `json:"allowCredentials" yaml:"allowCredentials"`

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached. Set it to nil to omit Access-Control-Max-Age.
	//
	// Optional. Default: nil.
	MaxAge *int `json:"maxAge" yaml:"maxAge"`

	// NormalizeHost optionally normalizes non-IP origin hosts in AllowOrigins.
	//
	// If nil, hosts are only lower-cased; IDNA is intentionally not handled
	// by default. A caller may provide an IDNA ToASCII implementation here.
	//
	// Optional. Default: nil.
	NormalizeHost HostNormalizer `json:"-" yaml:"-"`
}

var defaultAllowMethods = []string{
	http.MethodHead, http.MethodGet,
	http.MethodPost, http.MethodPut,
	http.MethodPatch, http.MethodDelete,
}

// NewDefaultConfig returns a default CORS config that only presets
//   - AllowOrigins: []string{"*"}
//   - AllowMethods: []string{"GET", "PUT", "HEAD", "POST", "PATCH", "DELETE"}
func NewDefaultConfig() Config {
	return Config{
		AllowOrigins: []string{"*"},
		AllowMethods: slices.Clone(defaultAllowMethods),
	}
}

// CORS returns a CORS middleware with the given priority.
func (c Config) CORS(priority int) *CORS {
	cors := &CORS{
		priority: priority,

		allowCredentials: c.AllowCredentials,
	}

	if c.AllowCredentials {
		c.ExposeHeaders = removeWildcard(c.ExposeHeaders)
	}

	if c.MaxAge != nil {
		if *c.MaxAge < 0 {
			panic(fmt.Errorf("cors.Config.MaxAge: invalid value %d", *c.MaxAge))
		}
		cors.maxAgeStr = fmt.Sprintf("%d", *c.MaxAge)
	}

	allowOrigins := normalizeAllowOrigins(c.AllowOrigins, c.NormalizeHost)
	cors.compileAllowOrigins(allowOrigins)
	cors.staticAllowOrigin, cors.varyOrigin = originResponseMode(allowOrigins, c.AllowCredentials)
	cors.varyActual, cors.varyPreflight = varyResponseValues(cors.varyOrigin)

	cors.allowMethods = joinHeaderValues("AllowMethods", c.AllowMethods, true)
	cors.allowHeaders = joinHeaderValues("AllowHeaders", c.AllowHeaders, true)
	cors.exposeHeaders = joinHeaderValues("ExposeHeaders", c.ExposeHeaders, true)
	cors.allowMethodsList = splitHeaderValues([]string{cors.allowMethods})
	cors.allowHeadersList = splitHeaderValues([]string{cors.allowHeaders})
	cors.allowMethodsWildcard = containsWildcard(c.AllowMethods)
	cors.allowHeadersWildcard = containsWildcard(c.AllowHeaders)

	return cors
}

// CORS is a CORS implementation.
type CORS struct {
	allowCredentials  bool
	allowAllOrigins   bool
	exactAllowOrigin  string
	staticAllowOrigin string
	allowOrigins      map[string]struct{}
	subdomainOrigins  []subdomainOriginPattern

	priority      int
	allowMethods  string
	allowHeaders  string
	exposeHeaders string
	maxAgeStr     string

	allowMethodsList     []string
	allowHeadersList     []string
	allowMethodsWildcard bool
	allowHeadersWildcard bool

	varyOrigin    bool
	varyActual    string
	varyPreflight string

	next http.Handler
}

// Priority returns the priority, which may be used as the priority of httpx.Middleware.
func (c *CORS) Priority() int { return c.priority }

// HTTPHandler implements the interface httpx.Middleware.
func (c *CORS) HTTPHandler(next http.Handler) http.Handler {
	if next == nil {
		panic("CORS.HTTPHandler: next http.Handler is nil")
	}

	_c := *c
	_c.next = next
	return &_c
}

// ServeHTTP implements the interface http.Handler.
//
// The method writes CORS Vary fields before passing actual requests to the next
// handler. Therefore, downstream handlers should use Header().Add("Vary", field)
// instead of Header().Set when adding their own Vary fields, otherwise they may
// overwrite the CORS fields.
//
// A failed preflight request is rejected with 403. A rejected or invalid Origin
// on an actual request is passed to the next handler without CORS allow headers;
// use CSRF or application-level origin checks when server-side rejection is required.
func (c *CORS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.next == nil {
		w.WriteHeader(500)
		_, _ = io.WriteString(w, "NO NEXT HANDLER")
		return
	}

	respHeader := w.Header()
	rawOrigin := r.Header.Get("Origin")
	if c.staticAllowOrigin != "" && !isPreflightRequest(r, rawOrigin != "") {
		respHeader.Set("Access-Control-Allow-Origin", c.staticAllowOrigin)
		if c.exposeHeaders != "" {
			respHeader.Set("Access-Control-Expose-Headers", c.exposeHeaders)
		}
		c.next.ServeHTTP(w, r)
		return
	}

	origin, hasOrigin := parseRequestOrigin(rawOrigin)
	allowOrigin, ok := c.allowOrigin(origin)
	if !ok {
		if isPreflightRequest(r, rawOrigin != "") {
			addVaryHeader(respHeader, c.varyPreflight)
			w.WriteHeader(http.StatusForbidden)
		} else {
			// CORS only controls browser access to the response for actual requests.
			addVaryHeader(respHeader, c.varyActual)
			c.next.ServeHTTP(w, r)
		}
		return
	}

	if isPreflightRequest(r, hasOrigin) {
		// Preflight request
		methods, methodsOK := c.preflightAllowMethods(r)
		headers, headersOK := c.preflightAllowHeaders(r)
		if !methodsOK || !headersOK {
			addVaryHeader(respHeader, c.varyPreflight)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		respHeader.Set("Access-Control-Allow-Origin", allowOrigin)
		if c.allowCredentials {
			respHeader.Set("Access-Control-Allow-Credentials", "true")
		}

		if methods != "" {
			respHeader.Set("Access-Control-Allow-Methods", methods)
		}

		if headers != "" {
			respHeader.Set("Access-Control-Allow-Headers", headers)
		}

		if c.maxAgeStr != "" {
			respHeader.Set("Access-Control-Max-Age", c.maxAgeStr)
		}

		addVaryHeader(respHeader, c.varyPreflight)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	respHeader.Set("Access-Control-Allow-Origin", allowOrigin)
	if c.allowCredentials {
		respHeader.Set("Access-Control-Allow-Credentials", "true")
	}

	// Actual CORS request.
	if c.exposeHeaders != "" {
		respHeader.Set("Access-Control-Expose-Headers", c.exposeHeaders)
	}

	addVaryHeader(respHeader, c.varyActual)
	c.next.ServeHTTP(w, r)
}

func (c *CORS) preflightAllowMethods(r *http.Request) (string, bool) {
	method := strings.TrimSpace(r.Header.Get("Access-Control-Request-Method"))
	if !validToken(method) {
		return "", false
	}

	if c.allowCredentials && c.allowMethodsWildcard {
		return method, true
	}

	if c.allowMethodsWildcard || slices.Contains(c.allowMethodsList, method) {
		return c.allowMethods, true
	}

	return "", false
}

func (c *CORS) preflightAllowHeaders(r *http.Request) (string, bool) {
	rheaders := r.Header.Values("Access-Control-Request-Headers")
	if len(rheaders) == 0 || (len(rheaders) == 1 && rheaders[0] == "") {
		if c.allowHeadersWildcard && c.allowCredentials {
			return "", true
		}
		return c.allowHeaders, true
	}

	requestHeaders, hasNonWildcardRequestHeader, ok := parseRequestHeaderValues(rheaders)
	if !ok {
		return "", false
	}

	if c.allowHeaders == "" || (c.allowHeadersWildcard && (c.allowCredentials || hasNonWildcardRequestHeader)) {
		return strings.Join(requestHeaders, ", "), true
	}

	if c.allowHeadersWildcard || slicex.ContainsAllFunc(c.allowHeadersList, requestHeaders, strings.EqualFold) {
		return c.allowHeaders, true
	}

	return "", false
}

func isPreflightRequest(r *http.Request, hasOrigin bool) bool {
	return hasOrigin && r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}

func addVaryHeader(h http.Header, value string) {
	if value == "" {
		return
	}

	varyValues := h.Values("Vary")
	if len(varyValues) == 0 {
		h.Set("Vary", value)
		return
	}

	fields := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	for _, varyValue := range varyValues {
		for field := range strings.SplitSeq(varyValue, ",") {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}

			if field == "*" {
				return
			}

			key := strings.ToLower(field)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				fields = append(fields, field)
			}
		}
	}

	for field := range strings.SplitSeq(value, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		key := strings.ToLower(field)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			fields = append(fields, field)
		}
	}

	if len(fields) > 0 {
		h.Set("Vary", strings.Join(fields, ", "))
	}
}
