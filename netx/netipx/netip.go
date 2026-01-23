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

// Package netipx provides some convenient netip functions.
package netipx

import (
	"net"
	"net/netip"

	"github.com/xgfone/go-toolkit/internal/netx"
)

// AddrFromNetAddr converts a net.Addr to netip.Addr.
func AddrFromNetAddr(netaddr net.Addr) (addr netip.Addr, err error) {
	switch v := netaddr.(type) {
	case *net.TCPAddr:
		addr, _ = netip.AddrFromSlice(v.IP)

	case *net.UDPAddr:
		addr, _ = netip.AddrFromSlice(v.IP)

	default:
		host, _ := netx.SplitHostPort(v.String())
		addr, err = netip.ParseAddr(host)
	}

	return
}
