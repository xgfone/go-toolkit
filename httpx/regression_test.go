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

package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/codeint"
	"github.com/xgfone/go-toolkit/errorx"
	"github.com/xgfone/go-toolkit/result"
)

func TestWrappedCodeintResponse(t *testing.T) {
	base := codeint.ErrNotFound.WithCode(400004).WithData("public data")
	for name, err := range map[string]error{
		"value":   fmt.Errorf("lookup: %w", base),
		"pointer": fmt.Errorf("lookup: %w", &base),
		"joined":  errors.Join(errors.New("lookup"), base),
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c := newContext(rec, httptest.NewRequest("GET", "/", nil))
			c.Failure(err)

			var response struct{ Error codeint.Error }
			if e := json.Unmarshal(rec.Body.Bytes(), &response); e != nil {
				t.Fatal(e)
			}

			if rec.Code != 404 || response.Error.Code != base.Code ||
				response.Error.Data != base.Data ||
				response.Error.Reason != err.Error() {
				t.Fatalf("wrapped error lost its status or fields: %d %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestWrappedSensitiveResponse(t *testing.T) {
	secret := codeint.ErrNotFound.WithMessage("secret message").WithData("secret data").WithReason("secret reason")
	err := fmt.Errorf("lookup: %w", errorx.Sensitive(secret, "unavailable"))

	rec := httptest.NewRecorder()
	c := newContext(rec, httptest.NewRequest("GET", "/", nil))

	c.Failure(err)
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), "lookup: unavailable") ||
		strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("sensitive response exposed underlying fields or lost status: %d %s", rec.Code, rec.Body)
	}
}

func TestRegressionWrappedErrorStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	c := AcquireContext()
	defer ReleaseContext(c)

	c.Reset(rec, req)
	DefaultRespond(c, result.Err(fmt.Errorf("lookup: %w", codeint.ErrNotFound)))
	if rec.Code != 404 {
		t.Fatalf("wrapped ErrNotFound returned %d with %s", rec.Code, rec.Body)
	}
}
