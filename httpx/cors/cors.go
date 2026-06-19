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
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
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
	// Optional. Default: []string{}.
	AllowHeaders []string `json:"allowHeaders" yaml:"allowHeaders"`

	// AllowMethods indicates methods allowed when accessing the resource.
	// This is used in response to a preflight request. If AllowCredentials is
	// true, "*" is reflected from the request because browsers do not treat it
	// as a wildcard for credentialed requests.
	//
	// Optional. Default: DefaultAllowMethods.
	AllowMethods []string `json:"allowMethods" yaml:"allowMethods"`

	// ExposeHeaders indicates response headers browsers are allowed to access
	// from an actual CORS response. If AllowCredentials is true, "*" is omitted
	// because browsers do not treat it as a wildcard for credentialed requests.
	//
	// Optional. Default: []string{}.
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

	// NormalizeHost optionally normalizes non-IP origin hosts in AllowOrigins
	// and request Origin headers.
	//
	// If nil, hosts are only lower-cased; IDNA is intentionally not handled
	// by default. A caller may provide an IDNA ToASCII implementation here.
	//
	// Optional. Default: nil.
	NormalizeHost HostNormalizer `json:"-" yaml:"-"`
}

var DefaultAllowMethods = []string{
	http.MethodHead, http.MethodGet,
	http.MethodPost, http.MethodPut,
	http.MethodPatch, http.MethodDelete,
}

// CORS returns a CORS middleware.
func (c Config) CORS(priority int) *CORS {
	if len(c.AllowMethods) == 0 {
		c.AllowMethods = slices.Clone(DefaultAllowMethods)
	}

	allowOrigins := normalizeAllowOrigins(c.AllowOrigins, c.NormalizeHost)
	cors := &CORS{
		priority: priority,

		allowOrigins:     allowOrigins,
		allowCredentials: c.AllowCredentials,
		normalizeHost:    c.NormalizeHost,
	}

	if c.MaxAge != nil {
		if *c.MaxAge < 0 {
			panic(fmt.Errorf("cors.Config.MaxAge: invalid value %d", *c.MaxAge))
		}
		cors.maxAgeStr = fmt.Sprintf("%d", *c.MaxAge)
	}

	cors.staticAllowOrigin, cors.varyOrigin = originResponseMode(allowOrigins, c.AllowCredentials)
	cors.allowMethodsWildcard = containsWildcard(c.AllowMethods)
	cors.allowHeadersWildcard = containsWildcard(c.AllowHeaders)

	if c.AllowCredentials {
		c.ExposeHeaders = removeWildcard(c.ExposeHeaders)
	}

	cors.allowMethods = joinHeaderValues("AllowMethods", c.AllowMethods, true)
	cors.allowHeaders = joinHeaderValues("AllowHeaders", c.AllowHeaders, true)
	cors.exposeHeaders = joinHeaderValues("ExposeHeaders", c.ExposeHeaders, true)
	cors.allowMethodsList = splitHeaderValues([]string{cors.allowMethods})
	cors.allowHeadersList = splitHeaderValues([]string{cors.allowHeaders})

	return cors
}

// CORS is a CORS implementation.
type CORS struct {
	allowCredentials  bool
	allowOrigins      []string
	staticAllowOrigin string
	normalizeHost     HostNormalizer

	priority      int
	allowMethods  string
	allowHeaders  string
	exposeHeaders string
	maxAgeStr     string

	allowMethodsList []string
	allowHeadersList []string

	allowMethodsWildcard bool
	allowHeadersWildcard bool
	varyOrigin           bool

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
func (c *CORS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if c.next == nil {
		w.WriteHeader(500)
		_, _ = io.WriteString(w, "NO NEXT HANDLER")
		return
	}

	respHeader := w.Header()
	var vary []string
	if c.varyOrigin {
		vary = append(vary, "Origin")
	}

	rawOrigin := r.Header.Get("Origin")
	origin, hasOrigin := normalizeRequestOrigin(rawOrigin, c.normalizeHost)
	allowOrigin, ok := c.allowOrigin(origin)
	if !ok {
		if isPreflightRequest(r, rawOrigin != "") {
			vary = append(vary, "Access-Control-Request-Method", "Access-Control-Request-Headers")
			addVaryValues(respHeader, vary...)
			w.WriteHeader(http.StatusForbidden)
		} else {
			addVaryValues(w.Header(), vary...)
			c.next.ServeHTTP(w, r)
		}
		return
	}

	if isPreflightRequest(r, hasOrigin) {
		// Preflight request
		vary = append(vary, "Access-Control-Request-Method", "Access-Control-Request-Headers")
		methods, methodsOK := c.preflightAllowMethods(r)
		headers, headersOK := c.preflightAllowHeaders(r)
		if !methodsOK || !headersOK {
			addVaryValues(respHeader, vary...)
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

		addVaryValues(respHeader, vary...)
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

	addVaryValues(w.Header(), vary...)
	c.next.ServeHTTP(w, r)
}

func (c *CORS) allowOrigin(origin string) (string, bool) {
	if c.staticAllowOrigin != "" {
		return c.staticAllowOrigin, true
	}

	if origin == "" {
		return "", false
	}

	for _, o := range c.allowOrigins {
		switch o {
		case "*":
			return origin, true

		case origin:
			return o, true

		default:
			if matchSubdomain(origin, o) {
				return origin, true
			}
		}
	}
	return "", false
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
	requestHeaders := splitHeaderValues(r.Header.Values("Access-Control-Request-Headers"))
	if !validRequestHeaderValues(requestHeaders) {
		return "", false
	}

	if c.allowHeaders == "" || (c.allowHeadersWildcard && (c.allowCredentials || containsCORSNonWildcardRequestHeaderName(requestHeaders))) {
		headers := joinRequestHeaderValues(requestHeaders)
		return headers, true
	}

	if c.allowHeadersWildcard || allowedRequestHeaders(c.allowHeadersList, requestHeaders) {
		return c.allowHeaders, true
	}

	return "", false
}

func isPreflightRequest(r *http.Request, hasOrigin bool) bool {
	return hasOrigin && r.Method == http.MethodOptions &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

func joinHeaderValues(name string, values []string, allowWildcard bool) string {
	if len(values) == 0 {
		return ""
	}

	n := 0
	values = slices.Clone(values)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if value != "*" || !allowWildcard {
			if !validToken(value) {
				panic(fmt.Errorf("cors.Config.%s: invalid value %q", name, value))
			}
		}

		if value != "" {
			values[n] = value
			n++
		}
	}

	return strings.Join(values[:n], ", ")
}

func splitHeaderValues(values []string) []string {
	var hs []string
	for _, value := range values {
		for _, h := range strings.Split(value, ",") {
			if h = strings.TrimSpace(h); h != "" {
				hs = append(hs, h)
			}
		}
	}
	return hs
}

func joinRequestHeaderValues(values []string) string {
	if len(values) == 0 {
		return ""
	}

	hs := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if !validToken(value) {
			return ""
		}

		hs = append(hs, value)
	}
	return strings.Join(hs, ", ")
}

func validRequestHeaderValues(values []string) bool {
	for _, value := range values {
		if !validToken(strings.TrimSpace(value)) {
			return false
		}
	}
	return true
}

func allowedRequestHeaders(allowed, requested []string) bool {
	for _, request := range requested {
		if !containsHeaderName(allowed, request) {
			return false
		}
	}
	return true
}

func containsHeaderName(headers []string, header string) bool {
	for _, h := range headers {
		if strings.EqualFold(h, header) {
			return true
		}
	}
	return false
}

func containsCORSNonWildcardRequestHeaderName(headers []string) bool {
	return slices.ContainsFunc(headers, isCORSNonWildcardRequestHeaderName)
}

func isCORSNonWildcardRequestHeaderName(header string) bool {
	return strings.EqualFold(header, "Authorization")
}

func containsWildcard(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "*" {
			return true
		}
	}
	return false
}

func removeWildcard(values []string) []string {
	n := 0
	values = slices.Clone(values)
	for _, value := range values {
		if strings.TrimSpace(value) != "*" {
			values[n] = value
			n++
		}
	}
	return values[:n]
}

func normalizeAllowOrigins(origins []string, normalizeHost HostNormalizer) []string {
	if len(origins) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(origins))
	seen := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}

		var o string
		switch {
		case origin == "*":
			o = origin

		case strings.Contains(origin, "://*."):
			var ok bool
			o, ok = normalizeSubdomainPattern(origin, normalizeHost)
			if !ok {
				panic(fmt.Errorf("cors.Config.AllowOrigins: invalid origin %q", origin))
			}

		default:
			var ok bool
			o, ok = normalizeOrigin(origin, normalizeHost)
			if !ok {
				panic(fmt.Errorf("cors.Config.AllowOrigins: invalid origin %q", origin))
			}
		}

		if o == "*" {
			return []string{"*"}
		}

		if _, ok := seen[o]; !ok {
			seen[o] = struct{}{}
			normalized = append(normalized, o)
		}
	}

	return normalized
}

func originResponseMode(origins []string, allowCredentials bool) (staticAllowOrigin string, varyOrigin bool) {
	switch {
	case len(origins) == 0:
		return "", false

	case len(origins) == 1 && origins[0] == "*" && !allowCredentials:
		return "*", false

	default:
		return "", true
	}
}

func normalizeRequestOrigin(origin string, normalizeHost HostNormalizer) (string, bool) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "", false
	}
	return normalizeOrigin(origin, normalizeHost)
}

func normalizeOrigin(origin string, normalizeHost HostNormalizer) (string, bool) {
	if origin == "null" {
		return origin, true
	}

	u, ok := parseOriginURL(origin)
	if !ok {
		return "", false
	}

	origin = serializeOriginURL(u, normalizeHost)
	return origin, origin != ""
}

func normalizeSubdomainPattern(pattern string, normalizeHost HostNormalizer) (string, bool) {
	if !strings.Contains(pattern, "://*.") {
		return "", false
	}

	u, ok := parseOriginURL(pattern)
	if !ok {
		return "", false
	}

	host, ok := serializeSubdomainPatternHost(u.Hostname(), normalizeHost)
	if !ok || len(host) <= 2 {
		return "", false
	}

	scheme := strings.ToLower(u.Scheme)
	if port, ok := normalizeURLPort(scheme, u.Port()); ok && port != "" {
		host = net.JoinHostPort(host, port)
	} else if !ok {
		return "", false
	}
	return scheme + "://" + host, true
}

func parseOriginURL(origin string) (*url.URL, bool) {
	if strings.ContainsAny(origin, "\r\n\t ") {
		return nil, false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, false
	}

	if !supportedOriginScheme(u.Scheme) {
		return nil, false
	}

	if strings.ContainsAny(u.Host, "\r\n\t ") || u.Hostname() == "" {
		return nil, false
	}

	if port := u.Port(); port != "" && !validPort(port) {
		return nil, false
	}

	return u, true
}

func supportedOriginScheme(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "http", "https", "ws", "wss":
		return true
	default:
		return false
	}
}

func serializeOriginURL(u *url.URL, normalizeHost HostNormalizer) string {
	scheme := strings.ToLower(u.Scheme)
	host, ok := serializeOriginHost(u.Hostname(), normalizeHost)
	if !ok {
		return ""
	}

	if port, ok := normalizeURLPort(scheme, u.Port()); !ok {
		return ""
	} else if port != "" {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return scheme + "://" + host
}

func serializeOriginHost(host string, normalizeHost HostNormalizer) (string, bool) {
	if host == "" || strings.ContainsAny(host, "\r\n\t ") {
		return "", false
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if addr.Zone() != "" {
			return "", false
		}
		return strings.ToLower(addr.String()), true
	}

	if normalizeHost != nil {
		var ok bool
		host, ok = normalizeHost(host)
		if !ok {
			return "", false
		}
	}

	if host == "" || strings.ContainsAny(host, "\r\n\t :[]") || strings.Contains(host, "*") {
		return "", false
	}
	return strings.ToLower(host), true
}

func serializeSubdomainPatternHost(host string, normalizeHost HostNormalizer) (string, bool) {
	if host == "" || strings.ContainsAny(host, "\r\n\t ") {
		return "", false
	}

	host = strings.ToLower(host)
	if !strings.HasPrefix(host, "*.") || strings.Contains(host[2:], "*") {
		return "", false
	}

	suffix, ok := serializeOriginHost(host[2:], normalizeHost)
	if !ok || strings.Contains(suffix, ":") {
		return "", false
	}

	return "*." + suffix, true
}

func normalizeURLPort(scheme, port string) (string, bool) {
	if port == "" {
		return "", true
	}

	p, ok := parsePort(port)
	if !ok {
		return "", false
	}

	if isDefaultOriginPort(scheme, p) {
		return "", true
	}
	return strconv.FormatUint(p, 10), true
}

func isDefaultOriginPort(scheme string, port uint64) bool {
	switch strings.ToLower(scheme) {
	case "http", "ws":
		return port == 80

	case "https", "wss":
		return port == 443

	default:
		return false
	}
}

func validPort(port string) bool {
	_, ok := parsePort(port)
	return ok
}

func parsePort(port string) (uint64, bool) {
	if port == "" {
		return 0, false
	}

	for _, r := range port {
		if r < '0' || r > '9' {
			return 0, false
		}
	}

	p, err := strconv.ParseUint(port, 10, 16)
	return p, err == nil
}

func validToken(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		if !isTChar(r) {
			return false
		}
	}

	return true
}

func isTChar(r rune) bool {
	return r == '!' || r == '#' || r == '$' || r == '%' || r == '&' ||
		r == '\'' || r == '*' || r == '+' || r == '-' || r == '.' ||
		r == '^' || r == '_' || r == '`' || r == '|' || r == '~' ||
		(r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// matchSubdomain compares authority with wildcard.
func matchSubdomain(domain, pattern string) bool {
	du, dok := parseOriginURL(domain)
	pu, pok := parseOriginURL(pattern)
	if !dok || !pok || du.Scheme != pu.Scheme || du.Port() != pu.Port() {
		return false
	}

	domHost := strings.ToLower(du.Hostname())
	patHost := strings.ToLower(pu.Hostname())
	if len(domHost) > 253 || !isSubdomainPattern(pattern) {
		return false
	}

	suffix := strings.TrimPrefix(patHost, "*.")
	return strings.HasSuffix(domHost, "."+suffix)
}

func isSubdomainPattern(pattern string) bool {
	u, ok := parseOriginURL(pattern)
	return ok && strings.HasPrefix(strings.ToLower(u.Hostname()), "*.")
}

func addVaryValues(h http.Header, values ...string) {
	if len(values) == 0 {
		return
	}

	var fields []string
	seen := make(map[string]struct{}, len(values))
	for _, value := range h.Values("Vary") {
		for _, field := range strings.Split(value, ",") {
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

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		key := strings.ToLower(value)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			fields = append(fields, value)
		}
	}

	if len(fields) > 0 {
		h.Set("Vary", strings.Join(fields, ", "))
	}
}
