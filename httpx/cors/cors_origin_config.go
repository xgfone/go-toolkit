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

package cors

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

type subdomainOriginPattern struct {
	scheme string
	suffix string
	port   string
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

func (c *cors) compileAllowOrigins(origins []string) {
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
	if uport := u.Port(); uport != "" {
		// parseOriginURL can ensure the port is valid (0-65535),
		// so we can safely ignore the error returned by strconv.ParseUint.
		port, _ := strconv.ParseUint(uport, 10, 16)
		if !isDefaultOriginPort(scheme, port) {
			host = net.JoinHostPort(host, strconv.FormatUint(port, 10))
		}
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
