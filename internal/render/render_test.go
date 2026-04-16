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
	"bytes"
	"io"
	"net/http"
	"testing"
)

func TestWriter(t *testing.T) {
	err := write(&mockResponseWriter{}, bytes.NewBufferString("abc"))
	if err != io.ErrShortWrite {
		t.Errorf("expect an io.ErrShortWrite, but got %v", err)
	}
}

type mockResponseWriter struct{}

func (m *mockResponseWriter) WriteHeader(statusCode int)  {}
func (m *mockResponseWriter) Write(p []byte) (int, error) { return 0, nil }
func (m *mockResponseWriter) Header() http.Header         { return nil }
