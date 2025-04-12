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

package httpx

import (
	"net/http"
	"testing"
)

func TestDoer(t *testing.T) {
	var client Doer

	client = DoFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 201}, nil
	})

	resp, err := client.Do(nil)
	if err != nil {
		t.Error(err)
	} else if resp.StatusCode != 201 {
		t.Errorf("expected status code 201, got %d", resp.StatusCode)
	}
}
