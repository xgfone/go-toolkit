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

package codeint

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/jsonx"
)

func TestErrorDecode(t *testing.T) {
	var err Error

	if e := err.Decode(func(a any) error {
		return jsonx.UnmarshalString(`{"Code":401}`, a)
	}); e != nil {
		t.Errorf("got an error: %s", e.Error())
	} else if err.Code != 401 {
		t.Errorf("expect code %d, but got %d", 401, err.Code)
	}

	if e := err.DecodeJSON(strings.NewReader(`{"Code":402}`)); e != nil {
		t.Errorf("got an error: %s", e.Error())
	} else if err.Code != 402 {
		t.Errorf("expect code %d, but got %d", 402, err.Code)
	}

	if e := err.DecodeJSONBytes([]byte(`{"Code":403}`)); e != nil {
		t.Errorf("got an error: %s", e.Error())
	} else if err.Code != 403 {
		t.Errorf("expect code %d, but got %d", 403, err.Code)
	}

	if e := err.DecodeJSONString(`{"Code":404}`); e != nil {
		t.Errorf("got an error: %s", e.Error())
	} else if err.Code != 404 {
		t.Errorf("expect code %d, but got %d", 404, err.Code)
	}
}

func TestError(t *testing.T) {
	err := NewError(400).WithCode(401).WithData("abc").WithStatus(501).
		WithMessagef("message %s", "1").WithReasonf("reason %s", "2")

	if e := err.Unwrap(); e != nil {
		t.Errorf("expect nil, but got an error: %s", e.Error())
	}
	if code := err.GetCode(); code != 401 {
		t.Errorf("expect code %d, but got %d", 401, code)
	}
	if code := err.StatusCode(); code != 501 {
		t.Errorf("expect status code %d, but got %d", 501, code)
	}
	if code := err.WithStatus(600).StatusCode(); code != 500 {
		t.Errorf("expect status code %d, but got %d", 500, code)
	}

	if s := err.String(); s != "code=401, msg=message 1, reason=reason 2, data=abc" {
		t.Errorf("expect string '%s', but got '%s'", "code=401, msg=message 1, reason=reason 2, data=abc", s)
	}

	if s := err.Error(); s != "reason 2" {
		t.Errorf("expect error '%s', but got '%s'", "reason 2", s)
	}
	if s := err.WithReason("").Error(); s != "message 1" {
		t.Errorf("expect error '%s', but got '%s'", "message 1", s)
	}
	if s := err.WithReason("").WithMessage("").Error(); s != "status=501, code=401" {
		t.Errorf("expect error '%s', but got '%s'", "status=501, code=401", s)
	}
	if s := err.WithReason("").WithMessage("").WithStatus(0).Error(); s != "code=401" {
		t.Errorf("expect error '%s', but got '%s'", "code=401", s)
	}

	err = err.WithErrorf("error %s", "3")
	if s := err.Error(); s != "error 3" {
		t.Errorf("expect error '%s', but got '%s'", "error 3", s)
	}

	if e := err.WithError(nil); e.Err != nil {
		t.Errorf("expect inner error is nil, but got '%s'", e.Err)
	}

	if e := err.TryError(nil); e != nil {
		t.Errorf("expect nil, but got an error: %s", e.Error())
	}
	if e := err.TryError(NewError(429)); e == nil {
		t.Errorf("expect an error, but got nil")
	} else if _e, ok := e.(Error); !ok {
		t.Errorf("expect an Error, but got %T", e)
	} else if _e.GetCode() != 429 {
		t.Errorf("expect code %d, but got %d", 429, _e.GetCode())
	}
	if e := err.TryError(errors.New("error")); e == nil {
		t.Errorf("expect an error, but got nil")
	} else if _e, ok := e.(Error); !ok {
		t.Errorf("expect an Error, but got %T", e)
	} else if code := _e.GetCode(); code != 401 {
		t.Errorf("expect code %d, but got %d", 401, code)
	} else if s := _e.Error(); s != "error" {
		t.Errorf("expect error '%s', but got '%s'", "error", s)
	}

	if !err.Is(NewError(401)) {
		t.Error("expect error 401, but got not")
	}
	if !err.Is(_TestError{Code: 401}) {
		t.Error("expect error 401, but got not")
	}
	if err.WithCode(500).WithError(NewError(403)).Is(errors.New("test")) {
		t.Error("unexpected error")
	}
}

type _TestError struct {
	Code int
}

func (e _TestError) Error() string {
	return fmt.Sprintf("code=%d", e.Code)
}

func (e _TestError) GetCode() int {
	return e.Code
}
