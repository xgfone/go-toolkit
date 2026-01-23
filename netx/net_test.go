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

package netx

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestIPIsOn(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantOn  bool
		wantErr bool
	}{
		// Empty IP
		{"empty ip", "", false, false},

		// Invalid IP
		{"invalid ip", "abc", false, true},
		{"invalid ip format", "256.256.256.256", false, true},
		{"invalid ipv6", "gggg::", false, true},

		// Local IP (should exist)
		{"localhost ipv4", "127.0.0.1", true, false},
		{"localhost ipv6", "::1", true, false},

		// Non-existent IP
		{"non-existent ipv4", "1.2.3.4", false, false},
		{"non-existent ipv6", "2001:db8::1", false, false},

		// IPv6 full format
		{"ipv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false, false},

		// IPv6 compressed format
		{"ipv6 compressed", "2001:db8::1", false, false},

		// IPv4-mapped IPv6 address
		{"ipv4-mapped ipv6", "::ffff:192.0.2.1", false, false},

		// IPv6 with zone - netip.ParseAddr actually accepts addresses with zones
		{"ipv6 with zone", "fe80::1%eth0", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOn, err := IPIsOn(tt.ip)

			if (err != nil) != tt.wantErr {
				t.Errorf("IPIsOn(%q) error = %v, wantErr %v", tt.ip, err, tt.wantErr)
				return
			}

			if !tt.wantErr && gotOn != tt.wantOn {
				t.Errorf("IPIsOn(%q) = %v, want %v", tt.ip, gotOn, tt.wantOn)
			}
		})
	}

	// Test IP address with port (should fail to parse)
	if _, err := IPIsOn("127.0.0.1:8080"); err == nil {
		t.Error("IPIsOn('127.0.0.1:8080') should fail, but got nil error")
	}

	// Test CIDR notation (should fail to parse)
	if _, err := IPIsOn("192.168.1.1/24"); err == nil {
		t.Error("IPIsOn('192.168.1.1/24') should fail, but got nil error")
	}

	// Test IPv6 CIDR notation (should fail to parse)
	if _, err := IPIsOn("2001:db8::/32"); err == nil {
		t.Error("IPIsOn('2001:db8::/32') should fail, but got nil error")
	}
}

func TestIpIsOnInternal(t *testing.T) {
	// Test when getAddrs returns an error
	t.Run("getAddrs returns error", func(t *testing.T) {
		mockGetAddrs := func() ([]net.Addr, error) {
			return nil, errors.New("mock interface error")
		}

		on, err := ipIsOn("127.0.0.1", mockGetAddrs)
		if err == nil {
			t.Error("ipIsOn should return error when getAddrs fails")
		}
		if on {
			t.Error("ipIsOn should return false when getAddrs fails")
		}
	})

	// Test address without slash
	t.Run("address without slash", func(t *testing.T) {
		mockAddr := &mockNetAddr{network: "ip+net", address: "127.0.0.1"}
		mockGetAddrs := func() ([]net.Addr, error) {
			return []net.Addr{mockAddr}, nil
		}

		on, err := ipIsOn("127.0.0.1", mockGetAddrs)
		if err != nil {
			t.Errorf("ipIsOn returned unexpected error: %v", err)
		}
		if !on {
			t.Error("ipIsOn should return true for matching IP without slash")
		}
	})

	// Test address starting with slash (should not happen, but code should handle it)
	t.Run("address starting with slash", func(t *testing.T) {
		mockAddr := &mockNetAddr{network: "ip+net", address: "/127.0.0.1"}
		mockGetAddrs := func() ([]net.Addr, error) {
			return []net.Addr{mockAddr}, nil
		}

		on, err := ipIsOn("127.0.0.1", mockGetAddrs)
		if err != nil {
			t.Errorf("ipIsOn returned unexpected error: %v", err)
		}
		if on {
			t.Error("ipIsOn should return false for address starting with slash")
		}
	})

	// Test empty address list
	t.Run("empty address list", func(t *testing.T) {
		mockGetAddrs := func() ([]net.Addr, error) {
			return []net.Addr{}, nil
		}

		on, err := ipIsOn("127.0.0.1", mockGetAddrs)
		if err != nil {
			t.Errorf("ipIsOn returned unexpected error: %v", err)
		}
		if on {
			t.Error("ipIsOn should return false for empty address list")
		}
	})

	// Test multiple addresses with a match
	t.Run("multiple addresses with match", func(t *testing.T) {
		mockAddr1 := &mockNetAddr{network: "ip+net", address: "192.168.1.1/24"}
		mockAddr2 := &mockNetAddr{network: "ip+net", address: "127.0.0.1/8"}
		mockAddr3 := &mockNetAddr{network: "ip+net", address: "10.0.0.1/8"}
		mockGetAddrs := func() ([]net.Addr, error) {
			return []net.Addr{mockAddr1, mockAddr2, mockAddr3}, nil
		}

		on, err := ipIsOn("127.0.0.1", mockGetAddrs)
		if err != nil {
			t.Errorf("ipIsOn returned unexpected error: %v", err)
		}
		if !on {
			t.Error("ipIsOn should return true when IP is in address list")
		}
	})

	// Test multiple addresses without a match
	t.Run("multiple addresses without match", func(t *testing.T) {
		mockAddr1 := &mockNetAddr{network: "ip+net", address: "192.168.1.1/24"}
		mockAddr2 := &mockNetAddr{network: "ip+net", address: "10.0.0.1/8"}
		mockGetAddrs := func() ([]net.Addr, error) {
			return []net.Addr{mockAddr1, mockAddr2}, nil
		}

		on, err := ipIsOn("127.0.0.1", mockGetAddrs)
		if err != nil {
			t.Errorf("ipIsOn returned unexpected error: %v", err)
		}
		if on {
			t.Error("ipIsOn should return false when IP is not in address list")
		}
	})
}

// mockNetAddr implements net.Addr interface for testing
type mockNetAddr struct {
	network string
	address string
}

func (m *mockNetAddr) Network() string {
	return m.network
}

func (m *mockNetAddr) String() string {
	return m.address
}

func TestIsTimeout(t *testing.T) {
	if IsTimeout(errors.New("false")) {
		t.Error("expect false, but got true")
	}

	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", time.Microsecond)
	if err == nil {
		_ = conn.Close()
		t.Error("expect an error, but got nil")
	} else if !IsTimeout(err) {
		t.Errorf("expect a timeout error, but got '%s'", err.Error())
	}
}

func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		input        string
		expectedHost string
		expectedPort string
	}{
		// Empty string
		{"", "", ""},

		// IPv4 addresses
		{"1.2.3.4", "1.2.3.4", ""},
		{"1.2.3.4:80", "1.2.3.4", "80"},
		{"1.2.3.4:8080", "1.2.3.4", "8080"},

		// IPv6 addresses
		{"ff00::", "ff00:", ""},
		{"[ff00::]", "ff00::", ""},
		{"[ff00::]:80", "ff00::", "80"},
		{"[ff00::]:8080", "ff00::", "8080"},
		{"[2001:db8::1]", "2001:db8::1", ""},
		{"[2001:db8::1]:443", "2001:db8::1", "443"},

		// Hostnames
		{"localhost", "localhost", ""},
		{"localhost:80", "localhost", "80"},
		{"example.com", "example.com", ""},
		{"example.com:443", "example.com", "443"},

		// Incomplete IPv6 addresses
		{"[abc", "[abc", ""},
		{"[abc]", "abc", ""},
		{"[abc]:80", "abc", "80"},

		// Invalid port numbers (contain non-digit characters)
		{"1.2.3.4:80a", "1.2.3.4:80a", ""},
		{"localhost:8.0", "localhost:8.0", ""},
		{"[ff00::]:80x", "[ff00::]:80x", ""},
		{"example.com:443abc", "example.com:443abc", ""},

		// Empty port numbers
		{"1.2.3.4:", "1.2.3.4", ""},
		{"localhost:", "localhost", ""},
		{"[ff00::]:", "ff00::", ""},

		// Multiple colons (take the last one)
		{"host:port:80", "host:port", "80"},
		{"host::80", "host:", "80"},

		// Special characters
		{"host-name:8080", "host-name", "8080"},
		{"host_name:8080", "host_name", "8080"},
		{"host.name:8080", "host.name", "8080"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			host, port := SplitHostPort(test.input)
			if host != test.expectedHost {
				t.Errorf("input=%q: expected host=%q, got %q", test.input, test.expectedHost, host)
			}
			if port != test.expectedPort {
				t.Errorf("input=%q: expected port=%q, got %q", test.input, test.expectedPort, port)
			}
		})
	}
}
