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

	"github.com/xgfone/go-toolkit/internal/netx"
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
	return ipIsOn(ip, net.InterfaceAddrs)
}

// ipIsOn is the internal implementation of IPIsOn that accepts a function
// to get interface addresses, making it testable.
func ipIsOn(ip string, getAddrs func() ([]net.Addr, error)) (on bool, err error) {
	if ip == "" {
		return
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return
	}

	addrs, err := getAddrs()
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

// SplitHostPort separates host and port from a string in "host:port",
// "ipv4:port" or "[ipv6]:port" format.
//
// The function doesn't validate the host or port format. For IPv6 addresses
// without brackets, the last colon is treated as the port separator.
//
// Examples:
//
//	"example.com:80"      -> host="example.com", port="80"
//	"1.2.3.4:80"          -> host="1.2.3.4", port="80"
//	"1.2.3.4"             -> host="1.2.3.4", port=""
//	"[ff00::1]:80"        -> host="ff00::1", port="80"
//	"[ff00::]"            -> host="ff00::", port=""
//	"ff00::"              -> host="ff00:", port=""
func SplitHostPort(hostport string) (host, port string) {
	return netx.SplitHostPort(hostport)
}
