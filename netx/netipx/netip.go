// Copyright 2024~2026 xgfone
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

// Package netipx provides some convenient netip functions.
package netipx

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/xgfone/go-toolkit/internal/netx"
)

// AddrFromNetAddr converts a net.Addr to netip.Addr.
func AddrFromNetAddr(netaddr net.Addr) (addr netip.Addr, err error) {
	switch v := netaddr.(type) {
	case nil:
		return addr, fmt.Errorf("nil net.Addr")

	case *net.TCPAddr:
		if v != nil {
			return addrFromIP(v.IP, v.Zone)
		}

	case *net.UDPAddr:
		if v != nil {
			return addrFromIP(v.IP, v.Zone)
		}

	case *net.IPAddr:
		if v != nil {
			host, _ := netx.SplitHostPort(v.String())
			return netip.ParseAddr(host)
		}

	default:
		host, _ := netx.SplitHostPort(v.String())
		return netip.ParseAddr(host)
	}

	return addr, fmt.Errorf("nil %T", netaddr)
}

func addrFromIP(ip net.IP, zone string) (netip.Addr, error) {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return netip.Addr{}, fmt.Errorf("invalid IP address %q", ip)
	}
	return addr.WithZone(zone), nil
}
