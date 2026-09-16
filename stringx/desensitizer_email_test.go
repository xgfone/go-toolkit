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

package stringx_test

import (
	"fmt"
	"testing"

	"github.com/xgfone/go-toolkit/stringx"
)

func TestEmailDesensitizer(t *testing.T) {
	d := stringx.EmailDesensitizer()
	for _, tt := range []struct {
		name, input, want string
	}{
		{"ordinary", "alice@example.com", "a****@example.com"},
		{"two_runes", "ab@example.com", "a****@example.com"},
		{"one_rune", "a@example.com", "****@example.com"},
		{"unicode_local", "用户@example.com", "用****@example.com"},
		{"one_unicode_rune", "用@example.com", "****@example.com"},
		{"unicode_domain", "用户@例子.公司", "用****@例子.公司"},
		{"alias", "alice+orders@example.com", "a****@example.com"},
		{"preserve_domain", "Alice@Mail.Example.COM", "A****@Mail.Example.COM"},
		{"display_name", "Alice Smith <alice@example.com>", "a****@example.com"},
		{"comment", "alice@example.com (Alice Smith)", "a****@example.com"},
		{"quoted_at", `"alice@work"@example.com`, "a****@example.com"},
		{"empty", "", ""},
		{"whitespace", " ", "****"},
		{"no_separator", "alice", "****"},
		{"no_local", "@example.com", "****"},
		{"no_domain", "alice@", "****"},
		{"extra_separator", "alice@@example.com", "****"},
		{"address_list", "alice@example.com, bob@example.com", "****"},
		{"invalid_utf8", "alice@\xff.com", "****"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.Desensitize(tt.input); got != tt.want {
				t.Errorf("Desensitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewEmailDesensitizerConfiguration(t *testing.T) {
	local := stringx.NewDesensitizer(2, 1).WithChars("#")
	d := stringx.NewEmailDesensitizer(local)
	if got := d.Desensitize("abcdef@example.com"); got != "ab#f@example.com" {
		t.Fatalf("custom configuration returned %q", got)
	}
	if got := d.Desensitize("甲乙丙丁@example.com"); got != "甲乙#丁@example.com" {
		t.Fatalf("Unicode local part returned %q", got)
	}

	local = local.WithLeft(0).WithRight(0).WithChars("hidden")
	changed := stringx.NewEmailDesensitizer(local)
	if got := changed.Desensitize("abcdef@example.com"); got != "hidden@example.com" {
		t.Fatalf("updated configuration returned %q", got)
	}
	if got := d.Desensitize("abcdef@example.com"); got != "ab#f@example.com" {
		t.Fatalf("changing the supplied configuration modified the original instance: %q", got)
	}
	if got := changed.Desensitize("invalid"); got != "****" {
		t.Fatalf("invalid input returned %q", got)
	}

	previous := stringx.EmailDesensitizer()
	t.Cleanup(func() { stringx.SetEmailDesensitizer(previous) })
	stringx.SetEmailDesensitizer(d)
	if got := stringx.EmailDesensitizer().Desensitize("abcdef@example.com"); got != "ab#f@example.com" {
		t.Fatalf("custom default returned %q", got)
	}
}

func TestNewEmailDesensitizerZeroMask(t *testing.T) {
	d := stringx.NewEmailDesensitizer(stringx.MaskDesensitizer{})
	if got := d.Desensitize("alice@example.com"); got != "****@example.com" {
		t.Fatalf("zero value returned %q", got)
	}
	if got := d.Desensitize(""); got != "" {
		t.Fatalf("empty input returned %q", got)
	}
}

func ExampleEmailDesensitizer() {
	d := stringx.EmailDesensitizer()
	fmt.Println(d.Desensitize("alice@example.com"))
	fmt.Println(d.Desensitize("a@example.com"))
	fmt.Println(d.Desensitize("用户@example.com"))
	// Output:
	// a****@example.com
	// ****@example.com
	// 用****@example.com
}

func ExampleNewEmailDesensitizer() {
	d := stringx.NewEmailDesensitizer(
		stringx.NewDesensitizer(2, 1).WithChars("***"),
	)
	fmt.Println(d.Desensitize("abcdef@example.com"))
	// Output: ab***f@example.com
}
