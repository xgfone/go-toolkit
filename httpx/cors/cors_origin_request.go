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
	"net/netip"
	"strings"
)

type requestOrigin struct {
	value  string
	scheme string
	host   string
	port   string
}

func (c *CORS) allowOrigin(origin requestOrigin) (string, bool) {
	if c.staticAllowOrigin != "" {
		return c.staticAllowOrigin, true
	}

	if origin.value == "" {
		return "", false
	}

	if c.allowAllOrigins {
		return origin.value, true
	}

	if c.exactAllowOrigin != "" && origin.value == c.exactAllowOrigin {
		return origin.value, true
	}

	if _, ok := c.allowOrigins[origin.value]; ok {
		return origin.value, true
	}

	if len(c.subdomainOrigins) > 0 && c.matchSubdomainOrigin(origin) {
		return origin.value, true
	}

	return "", false
}

func (c *CORS) matchSubdomainOrigin(origin requestOrigin) bool {
	if origin.host == "" || len(origin.host) > 253 || strings.Contains(origin.host, ":") {
		return false
	}

	for _, pattern := range c.subdomainOrigins {
		if origin.scheme == pattern.scheme && origin.port == pattern.port &&
			strings.HasSuffix(origin.host, "."+pattern.suffix) {
			return true
		}
	}
	return false
}

// parseRequestOrigin validates a browser-serialized Origin without normalizing it.
func parseRequestOrigin(origin string) (requestOrigin, bool) {
	if origin == "" {
		return requestOrigin{}, false
	}

	if origin == "null" {
		return requestOrigin{value: origin}, true
	}

	if containsInvalidRequestOriginByte(origin) {
		return requestOrigin{}, false
	}

	scheme, hostport, ok := strings.Cut(origin, "://")
	if !ok || scheme == "" || hostport == "" || !supportedSerializedOriginScheme(scheme) {
		return requestOrigin{}, false
	}

	if strings.ContainsAny(hostport, "/?#") || strings.Contains(hostport, "@") {
		return requestOrigin{}, false
	}

	host, port, ok := parseRequestOriginHostPort(scheme, hostport)
	if !ok {
		return requestOrigin{}, false
	}

	return requestOrigin{
		value:  origin,
		scheme: scheme,
		host:   host,
		port:   port,
	}, true
}

func containsInvalidRequestOriginByte(origin string) bool {
	for i := 0; i < len(origin); i++ {
		c := origin[i]
		if c <= ' ' || c >= 0x7f {
			return true
		}
	}
	return false
}

func supportedSerializedOriginScheme(scheme string) bool {
	switch scheme {
	case "http", "https", "ws", "wss":
		return true
	default:
		return false
	}
}

func parseRequestOriginHostPort(scheme, hostport string) (host, port string, ok bool) {
	if hostport[0] == '[' {
		end := strings.IndexByte(hostport, ']')
		if end <= 1 {
			return "", "", false
		}

		host = hostport[1:end]
		if !validSerializedIPv6Host(host) {
			return "", "", false
		}

		rest := hostport[end+1:]
		if rest == "" {
			return host, "", true
		}

		if rest[0] != ':' {
			return "", "", false
		}

		port = rest[1:]
		if !validSerializedOriginPort(scheme, port) {
			return "", "", false
		}
		return host, port, true
	}

	if strings.ContainsAny(hostport, "[]") {
		return "", "", false
	}

	if i := strings.LastIndexByte(hostport, ':'); i >= 0 {
		host, port = hostport[:i], hostport[i+1:]
		if strings.Contains(host, ":") || !validSerializedOriginPort(scheme, port) {
			return "", "", false
		}
	} else {
		host = hostport
	}

	if !validSerializedOriginHost(host) {
		return "", "", false
	}
	return host, port, true
}

func validSerializedOriginHost(host string) bool {
	if host == "" {
		return false
	}

	for i := 0; i < len(host); i++ {
		c := host[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '-' || c == '.' || c == '_':
		default:
			return false
		}
	}
	return true
}

func validSerializedIPv6Host(host string) bool {
	for i := 0; i < len(host); i++ {
		c := host[i]
		if c >= 'A' && c <= 'Z' {
			return false
		}
	}

	addr, err := netip.ParseAddr(host)
	return err == nil && addr.Is6() && addr.Zone() == ""
}

func validSerializedOriginPort(scheme, port string) bool {
	p, ok := parsePort(port)
	if !ok || (len(port) > 1 && port[0] == '0') {
		return false
	}
	return !isDefaultOriginPort(scheme, p)
}
