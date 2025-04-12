// Copyright 2025 xgfone
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

package jsonx

import (
	"strings"
	"testing"
)

func TestMarshal(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expect a panic, but got not")
			}
		}()

		SetMarshalWriterFunc(nil)
	}()

	data, err := Marshal("http://localhost/path?a=b&c=d")
	if err != nil {
		t.Fatal(err)
	}

	const expect = `"http://localhost/path?a=b&c=d"`
	if s := strings.TrimSpace(string(data)); s != expect {
		t.Errorf("expected '%s', but got '%s'", expect, s)
	}
}

func TestUnmarshal(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expect a panic, but got not")
			}
		}()

		SetUnmarshalReaderFunc(nil)
	}()

	var url string
	err := Unmarshal([]byte(`"http://localhost/path?a=b&c=d"`), &url)
	if err != nil {
		t.Fatal(err)
	}

	const expect = "http://localhost/path?a=b&c=d"
	if url != expect {
		t.Errorf("expected '%s', but got '%s'", expect, url)
	}
}

func TestUnmarshalBytes(t *testing.T) {
	data := []byte(`"abc"`)

	var s string
	if err := UnmarshalBytes(data, &s); err != nil {
		t.Fatal(err)
	} else if s != "abc" {
		t.Errorf("expected '%s', but got '%s'", "abc", s)
	}
}

func TestUnmarshalString(t *testing.T) {
	data := `"abc"`

	var s string
	if err := UnmarshalString(data, &s); err != nil {
		t.Fatal(err)
	} else if s != "abc" {
		t.Errorf("expected '%s', but got '%s'", "abc", s)
	}
}

func TestMarshalBytes(t *testing.T) {
	if data, err := MarshalBytes("abc"); err != nil {
		t.Fatal(err)
	} else if string(data) != `"abc"` {
		t.Errorf("expected '%s', but got '%s'", `"abc"`, string(data))
	}
}

func TestMarshalString(t *testing.T) {
	if s, err := MarshalString("abc"); err != nil {
		t.Fatal(err)
	} else if s != `"abc"` {
		t.Errorf("expected '%s', but got '%s'", `"abc"`, s)
	}
}
