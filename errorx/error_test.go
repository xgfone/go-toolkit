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

package errorx

import (
	"errors"
	"fmt"
	"testing"
)

func TestSensitive(t *testing.T) {
	err := errors.New("sensitive data")
	se := Sensitive(err, "safe")
	if se == nil {
		t.Fatal("Sensitive(...) = nil, want non-nil")
	}
	if se.Error() != "safe" {
		t.Fatalf("Error() = %q, want %q", se.Error(), "safe")
	}
	if se.SensitiveError() != err {
		t.Fatalf("SensitiveSource() = %#v, want %#v", se.SensitiveError(), err)
	}
}

func TestNewSensitiveError(t *testing.T) {
	raw := errors.New("dsn=mysql://root:pass@127.0.0.1/db")

	t.Run("nil", func(t *testing.T) {
		if got := NewSensitiveError(nil, "safe"); got != nil {
			t.Fatalf("NewSensitiveError(nil, ...) = %#v, want nil", got)
		}
	})

	t.Run("normal", func(t *testing.T) {
		got := NewSensitiveError(raw, "open db failed")
		if got == nil {
			t.Fatal("NewSensitiveError(...) = nil, want non-nil")
		}
		if got.Error() != "open db failed" {
			t.Fatalf("Error() = %q, want %q", got.Error(), "open db failed")
		}
		if got.SensitiveError() != raw {
			t.Fatalf("SensitiveError() = %#v, want %#v", got.SensitiveError(), raw)
		}
	})

	t.Run("trim and default", func(t *testing.T) {
		got := NewSensitiveError(raw, "   ")
		if got == nil {
			t.Fatal("NewSensitiveError(...) = nil, want non-nil")
		}
		if got.Error() != defaultSensitiveMessage {
			t.Fatalf("Error() = %q, want %q", got.Error(), defaultSensitiveMessage)
		}
	})
}

func TestSensitiveErrorNilReceiver(t *testing.T) {
	var err *SensitiveError

	if got := err.Error(); got != "<nil>" {
		t.Fatalf("(*SensitiveError)(nil).Error() = %q, want %q", got, "<nil>")
	}
	if got := err.Unwrap(); got != nil {
		t.Fatalf("(*SensitiveError)(nil).Unwrap() = %#v, want nil", got)
	}
	if got := err.SensitiveError(); got != nil {
		t.Fatalf("(*SensitiveError)(nil).SensitiveSource() = %#v, want nil", got)
	}
}

func TestErrorsIsAndAs(t *testing.T) {
	raw := errors.New("raw detail")
	err := fmt.Errorf("service: %w", NewSensitiveError(raw, "safe detail"))

	if !errors.Is(err, raw) {
		t.Fatal("errors.Is(err, raw) = false, want true")
	}

	var se *SensitiveError
	if !errors.As(err, &se) {
		t.Fatal("errors.As(err, SensitiveError) = false, want true")
	}
	if se.Error() != "safe detail" {
		t.Fatalf("se.Error() = %q, want %q", se.Error(), "safe detail")
	}
	if se.SensitiveError() != raw {
		t.Fatalf("se.SensitiveError() = %#v, want %#v", se.SensitiveError(), raw)
	}
}

func TestDirectSensitiveError(t *testing.T) {
	raw := errors.New("secret")
	err := NewSensitiveError(raw, "safe")

	var se *SensitiveError
	if !errors.As(err, &se) {
		t.Fatal("errors.As(err, SensitiveError) = false, want true")
	}
	if se.Error() != "safe" {
		t.Fatalf("se.Error() = %q, want %q", se.Error(), "safe")
	}
	if se.SensitiveError() != raw {
		t.Fatalf("se.SensitiveError() = %#v, want %#v", se.SensitiveError(), raw)
	}
}
