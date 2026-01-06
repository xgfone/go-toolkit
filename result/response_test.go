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

package result

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/jsonx"
)

type _Responser struct {
	t *testing.T
	s string
}

func (r _Responser) Respond(resp Response) {
	if s, err := jsonx.MarshalString(resp); err != nil {
		r.t.Fatalf("expect nil, but got %v", err)
	} else if s != r.s {
		r.t.Fatalf("expect '%s', but got '%s'", r.s, s)
	}
}

func TestResponse(t *testing.T) {
	resp := NewResponse(nil, nil)
	if code := resp.StatusCode(); code != 200 {
		t.Errorf("expect status code %d, but got %d", 200, code)
	}

	resp = resp.WithError(errors.New("test"))
	if code := resp.StatusCode(); code != 500 {
		t.Errorf("expect status code %d, but got %d", 500, code)
	}

	resp = resp.WithError(_Error{400, "test"})
	if code := resp.StatusCode(); code != 400 {
		t.Errorf("expect status code %d, but got %d", 400, code)
	}

	const s = `{"Error":"test","Data":123}`

	resp = resp.WithData(123)
	resp.Respond(_Responser{t: t, s: s})

	resp = resp.WithData(float64(123))
	_resp1 := Response{Error: new(_Error)}
	if err := _resp1.DecodeJSONBytes([]byte(s)); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(_resp1.Data, resp.Data) {
		t.Errorf("expect data %v, but got %v", resp.Data, _resp1.Data)
	} else if !_errequal(_resp1.Error, resp.Error) {
		t.Errorf("expect error %v, but got %v", resp.Error, _resp1.Error)
	}

	_resp2 := Response{Error: new(_Error)}
	if err := _resp2.DecodeJSON(strings.NewReader(s)); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(_resp2.Data, resp.Data) {
		t.Errorf("expect data %v, but got %v", resp.Data, _resp2.Data)
	} else if !_errequal(_resp2.Error, resp.Error) {
		t.Errorf("expect error %v, but got %v", resp.Error, _resp2.Error)
	}

	_resp3 := Response{Error: new(_Error)}
	decode := func(v any) error { return jsonx.UnmarshalBytes([]byte(s), v) }
	if err := _resp3.Decode(decode); err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(_resp3.Data, resp.Data) {
		t.Errorf("expect data %v, but got %v", resp.Data, _resp3.Data)
	} else if !_errequal(_resp3.Error, resp.Error) {
		t.Errorf("expect error %v, but got %v", resp.Error, _resp3.Error)
	}
}

func _errequal(e1, e2 error) bool {
	if e1 == nil && e2 == nil {
		return true
	} else if e1 == nil || e2 == nil {
		return false
	}

	return e1.Error() == e2.Error()
}

type _Error struct {
	code int
	msg  string
}

func (e _Error) Error() string   { return e.msg }
func (e _Error) StatusCode() int { return e.code }

func (e _Error) MarshalJSON() ([]byte, error) {
	return jsonx.MarshalBytes(e.msg)
}
func (e *_Error) UnmarshalJSON(data []byte) error {
	return jsonx.UnmarshalReader(&e.msg, bytes.NewReader(data))
}
