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

package netx

import "testing"

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

func TestValidOptionalPort(t *testing.T) {
	tests := []struct {
		port     string
		expected bool
	}{
		// Empty string
		{"", true},

		// Valid ports
		{":", true},
		{":0", true},
		{":80", true},
		{":8080", true},
		{":65535", true},

		// Invalid ports (not starting with colon)
		{"80", false},
		{"8080", false},
		{"port", false},

		// Invalid ports (contain non-digit characters)
		{":80a", false},
		{":8.0", false},
		{":80-", false},
		{":80+", false},
		{": 80", false},
		{":80 ", false},

		// Edge cases
		{":999999", true}, // Although exceeds 65535, still numeric
		{":-1", false},    // Contains minus sign
		{":+1", false},    // Contains plus sign
	}

	for _, test := range tests {
		t.Run(test.port, func(t *testing.T) {
			result := validOptionalPort(test.port)
			if result != test.expected {
				t.Errorf("validOptionalPort(%q) = %v, expected %v", test.port, result, test.expected)
			}
		})
	}
}
