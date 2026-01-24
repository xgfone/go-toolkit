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
		{"[abc]", "[abc]", ""},
		{"[abc]:80", "abc", "80"},

		// Invalid port numbers (contain non-digit characters)
		{"1.2.3.4:80a", "1.2.3.4", "80a"},
		{"localhost:8.0", "localhost", "8.0"},
		{"[ff00::]:80x", "ff00::", "80x"},
		{"example.com:443abc", "example.com", "443abc"},

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

		// Edge cases for IPv6 with brackets
		{"[abc]xyz", "[abc]xyz", ""},
		{"[abc]:80:extra", "[abc]:80:extra", ""},
		{"[ff00::]extra", "[ff00::]extra", ""},
		{"[ff00::]:80:extra", "[ff00::]:80:extra", ""},
		{"[]", "[]", ""},
		{"[]:", "", ""},
		{"[]:80", "", "80"},

		// More edge cases
		{"[", "[", ""},
		{"]", "]", ""},
		{"[:", "[:", ""},
		{"]:", "]", ""},
		{"[::]", "::", ""},
		{"[::]:", "::", ""},
		{"[::]:80", "::", "80"},
		{"[2001:db8::1:2:3:4]", "2001:db8::1:2:3:4", ""},
		{"[2001:db8::1:2:3:4]:8080", "2001:db8::1:2:3:4", "8080"},
		// IPv6 with multiple colons without brackets
		{"2001:db8::1:2:3:4", "2001:db8::1:2:3", "4"},
		{"2001:db8::1:2:3:4:80", "2001:db8::1:2:3:4", "80"},
		// Single character cases
		{":", "", ""},
		{"a:", "a", ""},
		{":a", "", "a"},
		{"[a", "[a", ""},
		{"a]", "a]", ""},
		// Multiple colons edge cases
		{":::80", "::", "80"},
		{"::::", ":::", ""},
		// Mixed cases
		{"[a:b]", "a:b", ""},
		{"[a:b]:80", "a:b", "80"},
		{"[a:b]:", "a:b", ""},
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
