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

package stringx

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

type emailMaskDesensitizer struct{ prefix, suffix Desensitizer }

// NewEmailDesensitizer returns a desensitizer that masks the parts before and
// after '@' in a single email address using prefix and suffix, respectively.
// If suffix is nil, the domain is preserved. It panics if prefix is nil
// (including a typed nil), or if suffix is a typed nil.
// The returned implementation is safe for concurrent use if prefix and suffix are.
//
// Display names and comments are discarded. Empty input stays empty;
// unparseable input always becomes "****", regardless of prefix and suffix.
func NewEmailDesensitizer(prefix, suffix Desensitizer) Desensitizer {
	checkNonNil(prefix, "stringx.NewEmailDesensitizer: prefix must not be nil")
	if suffix != nil {
		checkNonNil(suffix, "stringx.NewEmailDesensitizer: suffix must not be a typed nil")
	}
	return &emailMaskDesensitizer{prefix: prefix, suffix: suffix}
}

func (d *emailMaskDesensitizer) Desensitize(s string) string {
	if s == "" {
		return ""
	}
	if !utf8.ValidString(s) {
		return "****"
	}

	address, err := mail.ParseAddress(s)
	if err != nil {
		return "****"
	}

	result := "****"

	// Use only the parsed mailbox, omitting names and comments. A quoted local
	// part may contain '@', so the final separator identifies the domain.
	at := strings.LastIndexByte(address.Address, '@')
	if at > 0 && at < len(address.Address)-1 {
		prefix := d.prefix.Desensitize(address.Address[:at])
		suffix := address.Address[at+1:]
		if d.suffix != nil {
			suffix = d.suffix.Desensitize(suffix)
		}
		result = prefix + "@" + suffix
	}

	return result
}
