// Copyright 2024 xgfone
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

// Package netx provides some convenient net functions.
package netx

import (
	"errors"
	"net"
	"net/netip"
	"strings"
)

type timeoutError interface {
	Timeout() bool // Is the error a timeout?
	error
}

// IsTimeout reports whether the error is timeout.
func IsTimeout(err error) bool {
	var timeoutErr timeoutError
	return errors.As(err, &timeoutErr) && timeoutErr.Timeout()
}

// IPIsOn reports whether the ip is configured on a certain network interface.
//
// If ip is empty, return (false, nil).
func IPIsOn(ip string) (on bool, err error) {
	if ip == "" {
		return
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	ip = addr.String()
	for _, addr := range addrs {
		_ip := addr.String()
		if index := strings.IndexByte(_ip, '/'); index > 0 {
			_ip = _ip[:index]
		}
		if _ip == ip {
			return true, nil
		}
	}

	return false, nil
}

// SplitHostPort separates host and port. If the port is not valid, it returns
// the entire input as host, and it doesn't check the validity of the host.
// Unlike net.SplitHostPort, but per RFC 3986, it requires ports to be numeric.
func SplitHostPort(hostport string) (host, port string) {
	host = hostport

	colon := strings.LastIndexByte(host, ':')
	if colon != -1 && validOptionalPort(host[colon:]) {
		host, port = host[:colon], host[colon+1:]
	}

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	return
}

// validOptionalPort reports whether port is either an empty string
// or matches /^:\d*$/
func validOptionalPort(port string) bool {
	if port == "" {
		return true
	}
	if port[0] != ':' {
		return false
	}
	for _, b := range port[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}
