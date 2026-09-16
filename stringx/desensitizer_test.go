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

package stringx

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestDesensitizer(t *testing.T) {
	dephoner := NewDesensitizer(3, 4).WithChars("")

	if s := dephoner.Desensitize(""); s != "" {
		t.Errorf("expect an empty string, but got '%s'", s)
	}

	if s := dephoner.Desensitize("123"); s != "****" {
		t.Errorf("expect '%s', but got '%s'", "****", s)
	}

	if s := dephoner.Desensitize("1234567"); s != "****" {
		t.Errorf("expect '%s', but got '%s'", "****", s)
	}

	if s := dephoner.Desensitize("12345678"); s != "123****5678" {
		t.Errorf("expect '%s', but got '%s'", "123****5678", s)
	}

	if s := dephoner.Desensitize("1234567890"); s != "123****7890" {
		t.Errorf("expect '%s', but got '%s'", "123****7890", s)
	}

	denamer1 := dephoner.WithLeft(1).WithRight(0).WithChars("**")
	if s := denamer1.Desensitize("谢1谢2"); s != "谢**" {
		t.Errorf("expect '%s', but got '%s'", "谢**", s)
	}

	denamer2 := dephoner.WithLeft(0).WithRight(1).WithChars("**")
	if s := denamer2.Desensitize("1谢2谢"); s != "**谢" {
		t.Errorf("expect '%s', but got '%s'", "**谢", s)
	}

	denamer3 := dephoner.WithLeft(0).WithRight(0).WithChars("**")
	if s := denamer3.Desensitize("1谢2谢"); s != "**" {
		t.Errorf("expect '%s', but got '%s'", "**", s)
	}
}

func TestDesensitizerConfiguration(t *testing.T) {
	original := NewDesensitizer(3, 4)
	changed := original.WithLeft(1).WithRight(2).WithChars("#")
	if original.Left() != 3 || original.Right() != 4 || original.Chars() != "****" {
		t.Fatal("With methods modified the original")
	}
	if changed.Left() != 1 || changed.Right() != 2 || changed.Chars() != "#" {
		t.Fatal("unexpected changed configuration")
	}
	if got := changed.Desensitize("甲乙丙丁戊"); got != "甲#丁戊" {
		t.Fatalf("Desensitize() = %q", got)
	}
	for _, f := range []func(){
		func() { NewDesensitizer(8, -10) },
		func() { NewDesensitizer(-10, 8) },
		func() { original.WithLeft(-1) },
		func() { original.WithRight(-1) },
	} {
		mustPanic(t, f)
	}
}

func TestDesensitizerBoundaries(t *testing.T) {
	const maxInt = int(^uint(0) >> 1)
	for _, tt := range []struct {
		name        string
		d           Desensitizer
		input, want string
	}{
		{"overflow", NewDesensitizer(maxInt, maxInt), "secret", "****"},
		{"large_left", NewDesensitizer(maxInt, 0), "secret", "****"},
		{"large_right", NewDesensitizer(0, maxInt), "secret", "****"},
		{"exact_length", NewDesensitizer(2, 2), "甲乙丙丁", "****"},
		{"no_retained_runes", NewDesensitizer(0, 0), "secret", "****"},
		{"unicode_left", NewDesensitizer(1, 0), "😀甲乙", "😀****"},
		{"unicode_right", NewDesensitizer(0, 1), "甲乙😀", "****😀"},
		{"unicode_both", NewDesensitizer(1, 1), "😀甲乙😀", "😀****😀"},
		{"empty_input", NewDesensitizer(1, 1), "", "****"},
		{"empty_chars", NewDesensitizer(1, 1).WithChars(""), "abc", "a****c"},
		{"both_empty", NewDesensitizer(1, 1).WithChars(""), "", ""},
		{"zero_empty", MaskDesensitizer{}, "", ""},
		{"zero_nonempty", MaskDesensitizer{}, "secret", "****"},
		{"phone", PhoneDesensitizer(), "13812345678", "138****5678"},
		{"short", ShortDesensitizer(), "123456", "12****56"},
		{"default", DefaultDesensitizer(), "1234567890", "1234****7890"},
		{"password", PasswordDesensitizer(), "secret", "********"},
		{"password_empty", PasswordDesensitizer(), "", "********"},
		{"custom", DesensitizerFunc(strings.ToUpper), "abc", "ABC"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.Desensitize(tt.input); got != tt.want {
				t.Errorf("Desensitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDesensitizerSetters(t *testing.T) {
	for _, tt := range []struct {
		name string
		get  func() Desensitizer
		set  func(Desensitizer)
	}{
		{"phone", PhoneDesensitizer, SetPhoneDesensitizer},
		{"email", EmailDesensitizer, SetEmailDesensitizer},
		{"short", ShortDesensitizer, SetShortDesensitizer},
		{"default", DefaultDesensitizer, SetDefaultDesensitizer},
		{"password", PasswordDesensitizer, SetPasswordDesensitizer},
	} {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.get()
			t.Cleanup(func() { tt.set(original) })
			if reflect.TypeOf(original).Kind() != reflect.Pointer {
				t.Fatalf("initial implementation is %T, want a pointer", original)
			}

			pointer := new(NewDesensitizer(0, 0).WithChars("[pointer]"))
			value := NewDesensitizer(0, 0).WithChars("[value]")
			function := DesensitizerFunc(strings.ToUpper)

			tt.set(pointer)
			snapshot := tt.get()
			if snapshot != pointer {
				t.Fatal("getter did not retain the supplied pointer")
			}

			tt.set(value)
			if got := tt.get().Desensitize("secret"); got != "[value]" {
				t.Fatalf("value implementation returned %q", got)
			}

			tt.set(function)
			if got := tt.get().Desensitize("secret"); got != "SECRET" {
				t.Fatalf("function implementation returned %q", got)
			}
			if got := snapshot.Desensitize("secret"); got != "[pointer]" {
				t.Fatalf("replacement changed the previous instance: %q", got)
			}

			var nilPointer *MaskDesensitizer
			var nilEmail *emailMaskDesensitizer
			var nilFunc DesensitizerFunc
			for _, d := range []Desensitizer{nil, nilPointer, nilEmail, nilFunc} {
				mustPanic(t, func() { tt.set(d) })
				if got := tt.get().Desensitize("secret"); got != "SECRET" {
					t.Fatalf("failed setter changed the current implementation: %q", got)
				}
			}

			// Concurrent replacement must support different concrete types.
			implementations := []Desensitizer{pointer, value, function}
			var wg sync.WaitGroup
			for range 8 {
				wg.Go(func() {
					for i := 0; i < 100; i++ {
						tt.set(implementations[i%len(implementations)])
						switch got := tt.get().Desensitize("secret"); got {
						case "[pointer]", "[value]", "SECRET":
						default:
							t.Errorf("unexpected concurrent result %q", got)
						}
					}
				})
			}
			wg.Wait()
		})
	}
}

func ExampleNewDesensitizer() {
	d := NewDesensitizer(3, 4).WithChars("***")
	fmt.Println(d.Desensitize("13812345678"))
	fmt.Println(PhoneDesensitizer().Desensitize("13812345678"))
	// Output:
	// 138***5678
	// 138****5678
}
