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

	// NormalizeHost optionally normalizes non-IP origin hosts in AllowOrigins
	// and request Origin headers.
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

		normalizeHost:    c.NormalizeHost,
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
	normalizeHost     HostNormalizer

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

type subdomainOriginPattern struct {
	scheme string
	suffix string
	port   string
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

	origin, hasOrigin := normalizeRequestOrigin(rawOrigin, c.normalizeHost)
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

func (c *CORS) allowOrigin(origin string) (string, bool) {
	if c.staticAllowOrigin != "" {
		return c.staticAllowOrigin, true
	}

	if origin == "" {
		return "", false
	}

	if c.allowAllOrigins {
		return origin, true
	}

	if c.exactAllowOrigin != "" && origin == c.exactAllowOrigin {
		return origin, true
	}

	if _, ok := c.allowOrigins[origin]; ok {
		return origin, true
	}

	if len(c.subdomainOrigins) > 0 && c.matchSubdomainOrigin(origin) {
		return origin, true
	}

	return "", false
}

func (c *CORS) matchSubdomainOrigin(origin string) bool {
	u, ok := parseOriginURL(origin)
	if !ok {
		return false
	}

	scheme := strings.ToLower(u.Scheme)
	port := u.Port()
	host := strings.ToLower(u.Hostname())
	if len(host) > 253 {
		return false
	}

	for _, pattern := range c.subdomainOrigins {
		if scheme == pattern.scheme && port == pattern.port &&
			strings.HasSuffix(host, "."+pattern.suffix) {
			return true
		}
	}
	return false
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

func parseRequestHeaderValues(values []string) (headers []string, hasNonWildcardRequestHeader bool, ok bool) {
	if capacity := estimateRequestHeaderCount(values); capacity > 0 {
		headers = make([]string, 0, capacity)
	}

	for _, value := range values {
		for {
			part := value
			if i := strings.IndexByte(value, ','); i >= 0 {
				part = value[:i]
				value = value[i+1:]
			} else {
				value = ""
			}

			header := strings.TrimSpace(part)
			if header != "" {
				if !validToken(header) {
					return nil, false, false
				}
				if isCORSNonWildcardRequestHeaderName(header) {
					hasNonWildcardRequestHeader = true
				}
				headers = append(headers, header)
			}

			if value == "" {
				break
			}
		}
	}
	return headers, hasNonWildcardRequestHeader, true
}

func estimateRequestHeaderCount(values []string) int {
	const maxRequestHeadersPrealloc = 4

	count := 0
	for _, value := range values {
		if value == "" {
			continue
		}

		count++
		if count >= maxRequestHeadersPrealloc {
			return maxRequestHeadersPrealloc
		}

		for i := 0; i < len(value); i++ {
			if value[i] == ',' {
				count++
				if count >= maxRequestHeadersPrealloc {
					return maxRequestHeadersPrealloc
				}
			}
		}
	}
	return count
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

func (c *CORS) compileAllowOrigins(origins []string) {
	c.allowAllOrigins = false
	c.exactAllowOrigin = ""
	c.subdomainOrigins = nil
	c.allowOrigins = nil

	var exacts []string
	for _, origin := range origins {
		if origin == "*" {
			c.allowAllOrigins = true
			return
		}

		if pattern, ok := parseSubdomainOriginPattern(origin); ok {
			c.subdomainOrigins = append(c.subdomainOrigins, pattern)
			continue
		}

		exacts = append(exacts, origin)
	}

	switch len(exacts) {
	case 0:

	case 1:
		c.exactAllowOrigin = exacts[0]

	default:
		c.allowOrigins = make(map[string]struct{}, len(exacts))
		for _, origin := range exacts {
			c.allowOrigins[origin] = struct{}{}
		}
	}
}

func parseSubdomainOriginPattern(origin string) (subdomainOriginPattern, bool) {
	u, ok := parseOriginURL(origin)
	if !ok {
		return subdomainOriginPattern{}, false
	}

	host := strings.ToLower(u.Hostname())
	if !strings.HasPrefix(host, "*.") {
		return subdomainOriginPattern{}, false
	}

	suffix := host[2:]
	if suffix == "" || strings.Contains(suffix, "*") {
		return subdomainOriginPattern{}, false
	}

	return subdomainOriginPattern{
		scheme: strings.ToLower(u.Scheme),
		suffix: suffix,
		port:   u.Port(),
	}, true
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

func varyResponseValues(varyOrigin bool) (actual, preflight string) {
	if varyOrigin {
		return "Origin", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"
	}
	return "", "Access-Control-Request-Method, Access-Control-Request-Headers"
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

	var p uint64
	for i := 0; i < len(port); i++ {
		c := port[i]
		if c < '0' || c > '9' {
			return 0, false
		}

		p = p*10 + uint64(c-'0')
		if p > 65535 {
			return 0, false
		}
	}

	return p, true
}

func validToken(value string) bool {
	if value == "" {
		return false
	}

	for i := 0; i < len(value); i++ {
		if !isTChar(value[i]) {
			return false
		}
	}

	return true
}

func isTChar(c byte) bool {
	return tcharTable[c]
}

var tcharTable = func() [256]bool {
	var table [256]bool
	for c := byte('0'); c <= byte('9'); c++ {
		table[c] = true
	}
	for c := byte('A'); c <= byte('Z'); c++ {
		table[c] = true
	}
	for c := byte('a'); c <= byte('z'); c++ {
		table[c] = true
	}
	for _, c := range []byte("!#$%&'*+-.^_`|~") {
		table[c] = true
	}
	return table
}()

func addVaryHeader(h http.Header, value string) {
	if value == "" {
		return
	}

	varyValues := h.Values("Vary")
	if len(varyValues) == 0 {
		h.Set("Vary", value)
		return
	}

	var fields []string
	seen := make(map[string]struct{}, 4)
	for _, varyValue := range varyValues {
		for _, field := range strings.Split(varyValue, ",") {
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

	for _, field := range strings.Split(value, ",") {
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
