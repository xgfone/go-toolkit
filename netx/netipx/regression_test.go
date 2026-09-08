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

package netipx

import (
	"net"
	"net/netip"
	"testing"
)

func TestRegressionIPv6Zone(t *testing.T) {
	src := &net.TCPAddr{IP: net.ParseIP("fe80::1"), Zone: "en0", Port: 80}
	got, err := AddrFromNetAddr(src)
	if err != nil || got.Zone() != src.Zone {
		t.Fatalf("zone lost: addr=%s zone=%q err=%v", got, got.Zone(), err)
	}
}

func TestRegressionInvalidIP(t *testing.T) {
	got, err := AddrFromNetAddr(&net.UDPAddr{IP: net.IP{1}})
	if err == nil {
		t.Fatalf("invalid IP returned success: %v", got)
	}
}

func TestRegressionIPAddrIPv6(t *testing.T) {
	got, err := AddrFromNetAddr(&net.IPAddr{IP: net.ParseIP("2001:db8::1")})
	if err != nil || got.String() != "2001:db8::1" {
		t.Fatalf("net.IPAddr corrupted by port splitting: %v %v", got, err)
	}
}

func TestRegressionNetAddrIPRepresentation(t *testing.T) {
	for _, tc := range []struct {
		name string
		ip   net.IP
		zone string
		want string
	}{
		{name: "parsed IPv4", ip: net.ParseIP("192.0.2.1"), want: "192.0.2.1"},
		{name: "constructed IPv4", ip: net.IPv4(192, 0, 2, 1), want: "192.0.2.1"},
		{name: "four-byte IPv4", ip: net.IP{192, 0, 2, 1}, want: "192.0.2.1"},
		{name: "mapped IPv4", ip: net.ParseIP("::ffff:192.0.2.1"), want: "192.0.2.1"},
		{name: "IPv6", ip: net.ParseIP("2001:db8::1"), want: "2001:db8::1"},
		{name: "IPv6 zone", ip: net.ParseIP("fe80::1"), zone: "en0", want: "fe80::1%en0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := netip.MustParseAddr(tc.want)
			got, err := AddrFromNetAddr(&net.IPAddr{IP: tc.ip, Zone: tc.zone})
			if err != nil || got != want {
				t.Errorf("IPAddr = %v, %v; want %v", got, err, want)
			}

			// TCPAddr and UDPAddr already preserved the input's byte representation.
			if len(tc.ip) == net.IPv6len && want.Is4() {
				want = netip.MustParseAddr("::ffff:" + tc.want)
			}

			for _, src := range []net.Addr{
				&net.TCPAddr{IP: tc.ip, Zone: tc.zone, Port: 80},
				&net.UDPAddr{IP: tc.ip, Zone: tc.zone, Port: 80},
			} {
				got, err := AddrFromNetAddr(src)
				if err != nil || got != want {
					t.Errorf("%T = %v, %v; want %v", src, got, err, want)
				}
			}
		})
	}
}
