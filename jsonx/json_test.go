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
	"bytes"
	"strings"
	"testing"
)

func TestMarshal(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	err := Marshal(buf, "http://localhost/path?a=b&c=d")
	if err != nil {
		t.Fatal(err)
	}

	const expect = `"http://localhost/path?a=b&c=d"`
	if s := strings.TrimSpace(buf.String()); s != expect {
		t.Errorf("expected '%s', but got '%s'", expect, s)
	}
}

func TestUnmarshal(t *testing.T) {
	var url string
	err := Unmarshal(&url, bytes.NewBufferString(`"http://localhost/path?a=b&c=d"`))
	if err != nil {
		t.Fatal(err)
	}

	const expect = "http://localhost/path?a=b&c=d"
	if url != expect {
		t.Errorf("expected '%s', but got '%s'", expect, url)
	}
}
