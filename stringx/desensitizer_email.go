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

type emailMaskDesensitizer struct{ local MaskDesensitizer }

// NewEmailDesensitizer returns a desensitizer that masks the local part of a
// single email address using a copy of local, preserving the domain. The
// returned implementation is safe for concurrent use. A zero-value local
// replaces the entire local part with "****".
//
// Display names and comments are discarded. Empty input stays empty;
// unparseable input always becomes "****", regardless of local's configuration.
func NewEmailDesensitizer(local MaskDesensitizer) Desensitizer {
	return &emailMaskDesensitizer{local: local}
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

	// Use only the parsed mailbox, omitting names and comments. A quoted local
	// part may contain '@', so the final separator identifies the domain.
	at := strings.LastIndexByte(address.Address, '@')
	if at <= 0 || at == len(address.Address)-1 {
		return "****"
	}

	return d.local.Desensitize(address.Address[:at]) + address.Address[at:]
}
