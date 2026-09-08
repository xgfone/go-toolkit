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

package structx

import (
	"strings"
	"testing"
)

type regressionMutatingText string

func (v *regressionMutatingText) UnmarshalText(b []byte) error {
	if len(b) > 0 {
		b[0] = 'X'
	}

	*v = regressionMutatingText(b)
	return nil
}

func TestRegressionTextInputOwnership(t *testing.T) {
	s := strings.Clone("abc")
	src := map[string]string{"Value": s}

	var dst struct{ Value regressionMutatingText }
	if err := BindStringMap(&dst, src, ""); err != nil {
		t.Fatal(err)
	}

	if src["Value"] != "abc" {
		t.Fatalf("UnmarshalText modified immutable source string: %q", src["Value"])
	}
}

type regressionMutatingStruct struct{ hidden string }

func (v *regressionMutatingStruct) UnmarshalText(b []byte) error {
	if len(b) > 0 {
		b[0] = 'X'
	}

	v.hidden = string(b)
	return nil
}

func TestRegressionMapTextInputOwnership(t *testing.T) {
	s := strings.Clone("abc")
	src := map[string]any{"Value": s}

	var dst struct{ Value regressionMutatingStruct }
	if err := BindMap(&dst, src, ""); err != nil {
		t.Fatal(err)
	}

	if src["Value"] != "abc" {
		t.Fatalf("BindMap UnmarshalText modified immutable string: %q", src["Value"])
	}
}

func TestDefaultTextUnmarshalerMayModifyInput(t *testing.T) {
	var dst struct {
		Value regressionMutatingText `default:"abc"`
	}

	if err := SetDefault(&dst); err != nil {
		t.Fatal(err)
	}
	if dst.Value != "Xbc" {
		t.Fatalf("unexpected parsed default: %q", dst.Value)
	}
}
