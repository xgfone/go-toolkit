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
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDoRequestDecodeTarget(t *testing.T) {
	old := GetClient()
	t.Cleanup(func() { SetClient(old) })

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	do := func(body string, dst any) error {
		SetClient(DoFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}))
		return DoRequest(ctx, req, dst)
	}

	t.Run("struct pointer", func(t *testing.T) {
		var dst testResponse
		if err := do(`{"value":42}`, &dst); err != nil {
			t.Fatal(err)
		}
		if dst.Value != 42 {
			t.Fatalf("value = %d, want 42", dst.Value)
		}
	})

	t.Run("allocate through pointer", func(t *testing.T) {
		var dst *testResponse
		if err := do(`{"value":42}`, &dst); err != nil {
			t.Fatal(err)
		}
		if dst == nil || dst.Value != 42 {
			t.Fatalf("unexpected target: %+v", dst)
		}
	})

	t.Run("null map", func(t *testing.T) {
		dst := map[string]int{"value": 42}
		if err := do(`null`, &dst); err != nil {
			t.Fatal(err)
		}
		if dst != nil {
			t.Fatalf("null did not clear map: %v", dst)
		}
	})

	t.Run("null slice", func(t *testing.T) {
		dst := []int{42}
		if err := do(`null`, &dst); err != nil {
			t.Fatal(err)
		}
		if dst != nil {
			t.Fatalf("null did not clear slice: %v", dst)
		}
	})

	t.Run("null pointer", func(t *testing.T) {
		dst := &testResponse{Value: 42}
		if err := do(`null`, &dst); err != nil {
			t.Fatal(err)
		}
		if dst != nil {
			t.Fatalf("null did not clear pointer: %+v", dst)
		}
	})

	t.Run("null interface", func(t *testing.T) {
		var dst any = "previous"
		if err := do(`null`, &dst); err != nil {
			t.Fatal(err)
		}
		if dst != nil {
			t.Fatalf("null did not clear interface: %v", dst)
		}
	})

	for _, tc := range []struct {
		name string
		dst  any
	}{
		{"non-pointer", testResponse{}},
		{"typed nil", (*testResponse)(nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := do(`{"value":42}`, tc.dst)
			var invalid *json.InvalidUnmarshalError
			if !errors.As(err, &invalid) {
				t.Fatalf("expected InvalidUnmarshalError, got %v", err)
			}
		})
	}
}
