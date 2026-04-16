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

package render

import (
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	if err := JSON(rec, 400, nil); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s != "" {
		t.Errorf("expect response body '%s', but got '%s'", "", s)
	}

	rec = httptest.NewRecorder()
	if err := JSON(rec, 400, map[string]string{"a": "b"}); err != nil {
		t.Fatal(err)
	} else if s := rec.Body.String(); s != `{"a":"b"}`+"\n" {
		t.Errorf("expect response body '%s', but got '%s'", `{"a":"b"}`, s)
	}

	if rec.Code != 400 {
		t.Errorf("expect status code %d, but got %d", 400, rec.Code)
	}

	expectbody := `{"a":"b"}` + "\n"
	if body := rec.Body.String(); body != expectbody {
		t.Errorf("expect response body '%s', but got '%s'", expectbody, body)
	}
}
