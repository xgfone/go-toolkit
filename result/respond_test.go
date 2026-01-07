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
	"errors"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSetRespondFunc(t *testing.T) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("panic: expect nil, but got %v", r)
			}
		}()

		SetRespondFunc(defaultRespond)
	}()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expect a panic, but got nil")
			}
		}()

		SetRespondFunc(nil)
	}()

	if GetRespondFunc() == nil {
		t.Error("not expect nil")
	}
}

func TestDefaultRepond(t *testing.T) {
	SetRespondFunc(defaultRespond)

	Success(_NoopResponder{t: t}, nil)
	Success(_NoopJSONer{t: t}, nil)
	Success(_NoopJSONer{t: t}, 123)
	Failure(_NoopJSONer{t: t}, errors.New("test"))

	rec1 := httptest.NewRecorder()
	Success(rec1, nil)
	if rec1.Code != 200 {
		t.Errorf("expect status code %d, but got %d", 200, rec1.Code)
	} else if body := rec1.Body.String(); body != "" {
		t.Errorf("expect empty body, but got %s", body)
	}

	rec2 := httptest.NewRecorder()
	Respond(rec2, Response{Data: 123, Error: _Error{code: 400, msg: "test"}})
	if rec2.Code != 400 {
		t.Errorf("expect status code %d, but got %d", 400, rec2.Code)
	} else if body := rec2.Body.String(); body != `{"Error":"test","Data":123}`+"\n" {
		t.Errorf("expect body %s, but got %s", `{"Error":"test","Data":123}`+"\n", body)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expect a panic, but got nil")
		}
	}()
	Respond(123, Response{})
}

type (
	_NoopResponder struct{ t *testing.T }
	_NoopJSONer    struct{ t *testing.T }
)

func (r _NoopResponder) Respond(_ Response) {}
func (r _NoopJSONer) JSON(code int, value any) {
	resp, ok := value.(Response)
	if !ok && code != 200 && value != nil {
		r.t.Errorf("expect a Response, but got %T", value)
		return
	}

	if code == 200 {
		if resp.Data != nil && !reflect.DeepEqual(resp.Data, 123) {
			r.t.Errorf("expect %v, but got %v", 123, resp.Data)
		}
	} else {
		if s := resp.Error.Error(); s != "test" {
			r.t.Errorf("expect %s, but got %s", "test", s)
		}
	}
}
