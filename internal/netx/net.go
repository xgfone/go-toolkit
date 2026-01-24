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

// Package netx provides some convenient net functions.
package netx

import "strings"

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
	i := strings.LastIndexByte(hostport, ':')
	if i < 0 {
		host = hostport
		return
	}

	if hostport[0] != '[' {
		host = hostport[:i]
		port = hostport[i+1:]
		return
	}

	end := strings.IndexByte(hostport, ']')
	if end < 0 {
		host = hostport
		return
	}

	switch end + 1 {
	case len(hostport):
		host = hostport[1:end]
		return

	case i:
		host = hostport[1:end]
		port = hostport[i+1:]

	default:
		host = hostport
	}

	return
}
