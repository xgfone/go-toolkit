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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Test data structures
type testRequest struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type testResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// Mock HTTP client
type mockClient struct {
	doFunc func(*http.Request) (*http.Response, error)
}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"id":"123","name":"test","value":42}`)),
	}, nil
}

// Helper type: Reader that simulates read errors
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func TestGet(t *testing.T) {
	SetClient(&mockClient{doFunc: func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}})

	Get(context.Background(), "http://127.0.0.1", nil)
}

// Test Post function - basic success case
func TestPost_Success(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Verify request method
			if req.Method != http.MethodPost {
				t.Errorf("expected method POST, got %s", req.Method)
			}

			// Verify Content-Type header
			contentType := req.Header.Get("Content-Type")
			if contentType != MIMEApplicationJSON {
				t.Errorf("expected Content-Type %s, got %s", MIMEApplicationJSON, contentType)
			}

			// Read request body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			// Verify request body
			var reqBody testRequest
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}

			if reqBody.Name != "test" || reqBody.Value != 42 {
				t.Errorf("unexpected request body: %+v", reqBody)
			}

			// Return success response
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"123","name":"test","value":42}`)),
			}, nil
		},
	}

	// Set mock client
	SetClient(client)
	defer SetClient(http.DefaultClient) // Restore default client

	// Prepare request and response data
	reqData := testRequest{Name: "test", Value: 42}
	var respData testResponse

	// Call Post function
	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, reqData)

	// Verify results
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if respData.ID != "123" || respData.Name != "test" || respData.Value != 42 {
		t.Errorf("unexpected response data: %+v", respData)
	}
}

// Test Post function - req parameter is nil
func TestPost_RequestNil(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Verify Content-Type header is not set
			contentType := req.Header.Get("Content-Type")
			if contentType != "" {
				t.Errorf("expected empty Content-Type for nil request, got %s", contentType)
			}

			// Verify request body is empty (don't read if it might be nil)
			if req.Body != nil {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("failed to read request body: %v", err)
				}
				if len(body) != 0 {
					t.Errorf("expected empty body for nil request, got %s", string(body))
				}
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"456"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var respData testResponse
	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, nil)

	if err != nil {
		t.Fatalf("Post with nil request failed: %v", err)
	}

	if respData.ID != "456" {
		t.Errorf("unexpected response data: %+v", respData)
	}
}

// Test Post function - req parameter is io.Reader
func TestPost_RequestReader(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Verify Content-Type header
			contentType := req.Header.Get("Content-Type")
			if contentType != MIMEApplicationJSON {
				t.Errorf("expected Content-Type %s, got %s", MIMEApplicationJSON, contentType)
			}

			// Read request body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			// Verify request body content
			expectedBody := `{"name":"reader","value":99}`
			if string(body) != expectedBody {
				t.Errorf("expected body %s, got %s", expectedBody, string(body))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"789"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	// Create io.Reader request body
	reqBody := strings.NewReader(`{"name":"reader","value":99}`)
	var respData testResponse

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, reqBody)

	if err != nil {
		t.Fatalf("Post with io.Reader failed: %v", err)
	}

	if respData.ID != "789" {
		t.Errorf("unexpected response data: %+v", respData)
	}
}

// Test Post function - response status code is not 200
func TestPost_Non200StatusCode(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"error":"bad request"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqData := testRequest{Name: "test", Value: 42}
	var respData testResponse

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, reqData)

	if err == nil {
		t.Fatal("expected error for non-200 status code")
	}

	expectedErr := "statuscode=400, body={\"error\":\"bad request\"}"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

// Test Post function - HTTP request fails
func TestPost_HttpRequestFailed(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqData := testRequest{Name: "test", Value: 42}
	var respData testResponse

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, reqData)

	if err == nil {
		t.Fatal("expected error for failed HTTP request")
	}

	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("expected error containing 'network error', got %v", err)
	}
}

// Test Post function - reading response body fails
func TestPost_ReadResponseBodyFailed(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(&errorReader{}),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqData := testRequest{Name: "test", Value: 42}
	var respData testResponse

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, reqData)

	if err == nil {
		t.Fatal("expected error for failed response body reading")
	}

	if !strings.Contains(err.Error(), "statuscode=200") {
		t.Errorf("expected error containing 'statuscode=200', got %v", err)
	}
}

// Test Post function - JSON decoding fails
func TestPost_JsonDecodeFailed(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`invalid json`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqData := testRequest{Name: "test", Value: 42}
	var respData testResponse

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, reqData)

	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}

	if !strings.Contains(err.Error(), "fail to decode the response body") {
		t.Errorf("expected error containing 'fail to decode the response body', got %v", err)
	}
}

// Test Post function - resp parameter is nil
func TestPost_ResponseNil(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"123"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqData := testRequest{Name: "test", Value: 42}

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", nil, reqData)

	if err != nil {
		t.Fatalf("Post with nil response failed: %v", err)
	}
}

// Test Post function - resp parameter is a function type
func TestPost_ResponseFunction(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`custom response`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqData := testRequest{Name: "test", Value: 42}
	called := false

	respFunc := func(rsp *http.Response) error {
		called = true
		if rsp.StatusCode != http.StatusCreated {
			return fmt.Errorf("expected status 201, got %d", rsp.StatusCode)
		}
		return nil
	}

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", respFunc, reqData)

	if err != nil {
		t.Fatalf("Post with response function failed: %v", err)
	}

	if !called {
		t.Fatal("response function was not called")
	}
}

// Test Post function - request creation fails (invalid URL)
func TestPost_InvalidURL(t *testing.T) {
	// Save original client
	originalClient := GetClient()
	defer SetClient(originalClient)

	// Use default client
	SetClient(http.DefaultClient)

	reqData := testRequest{Name: "test", Value: 42}
	var respData testResponse

	ctx := context.Background()
	err := Post(ctx, "://invalid-url", &respData, reqData)

	if err == nil {
		t.Fatal("expected error for invalid URL")
	}

	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected parse error, got %v", err)
	}
}

// Test Post function - JSON encoding fails (non-encodable type)
func TestPost_JsonEncodeFailed(t *testing.T) {
	// Save original client
	originalClient := GetClient()
	defer SetClient(originalClient)

	// Use default client (request won't actually be sent)
	SetClient(http.DefaultClient)

	// Create a value that cannot be JSON encoded (function type)
	invalidReq := func() {}

	var respData testResponse

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, invalidReq)

	if err == nil {
		t.Fatal("expected error for JSON encoding failure")
	}

	if !strings.Contains(err.Error(), "fail to encode request body") {
		t.Errorf("expected error containing 'fail to encode request body', got %v", err)
	}
}

// Test Post function - response function returns error
func TestPost_ResponseFunctionError(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":"test"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqData := testRequest{Name: "test", Value: 42}
	expectedError := errors.New("response function error")

	respFunc := func(rsp *http.Response) error {
		return expectedError
	}

	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", respFunc, reqData)

	if err == nil {
		t.Fatal("expected error from response function")
	}

	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}

// Test Post function - bytes.Buffer as request body
func TestPost_BytesBufferRequest(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}

			// Verify request body
			expectedBody := `{"data":"bytes buffer"}`
			if string(body) != expectedBody {
				t.Errorf("expected body %s, got %s", expectedBody, string(body))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	// Use bytes.Buffer as request body
	var buf bytes.Buffer
	buf.WriteString(`{"data":"bytes buffer"}`)

	var respData map[string]any
	ctx := context.Background()
	err := Post(ctx, "http://example.com/api", &respData, &buf)

	if err != nil {
		t.Fatalf("Post with bytes.Buffer failed: %v", err)
	}

	if success, ok := respData["success"].(bool); !ok || !success {
		t.Errorf("unexpected response: %+v", respData)
	}
}

// Test Post function - various status code error messages
func TestPost_VariousStatusCodes(t *testing.T) {
	testCases := []struct {
		name        string
		statusCode  int
		body        string
		expectErr   bool
		errContains string
	}{
		{
			name:        "201 Created",
			statusCode:  http.StatusCreated,
			body:        `{"id":"123"}`,
			expectErr:   true,
			errContains: "statuscode=201",
		},
		{
			name:        "204 No Content",
			statusCode:  http.StatusNoContent,
			body:        "",
			expectErr:   true,
			errContains: "statuscode=204",
		},
		{
			name:        "404 Not Found",
			statusCode:  http.StatusNotFound,
			body:        `{"error":"not found"}`,
			expectErr:   true,
			errContains: "statuscode=404",
		},
		{
			name:        "500 Internal Server Error",
			statusCode:  http.StatusInternalServerError,
			body:        `{"error":"server error"}`,
			expectErr:   true,
			errContains: "statuscode=500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &mockClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tc.statusCode,
						Body:       io.NopCloser(strings.NewReader(tc.body)),
					}, nil
				},
			}

			SetClient(client)
			defer SetClient(http.DefaultClient)

			reqData := testRequest{Name: "test", Value: 42}
			var respData testResponse

			ctx := context.Background()
			err := Post(ctx, "http://example.com/api", &respData, reqData)

			if tc.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error containing %q, got %v", tc.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Test _ClientError methods
func TestClientError(t *testing.T) {
	// Create a mock request and response
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	rsp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader("error body")),
	}

	// Test basic error creation
	err := newClientError(req, rsp)
	if err.StatusCode() != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, err.StatusCode())
	}

	if err.Request() != req {
		t.Error("Request() should return the original request")
	}

	if err.Response() != rsp {
		t.Error("Response() should return the original response")
	}

	if err.ResponseBody() != "" {
		t.Errorf("expected empty response body, got %q", err.ResponseBody())
	}

	if err.Unwrap() != nil {
		t.Error("Unwrap() should return nil when no error is set")
	}

	// Test WithBody method
	body := []byte("test body")
	errWithBody := err.WithBody(body)
	if errWithBody.ResponseBody() != "test body" {
		t.Errorf("expected response body %q, got %q", "test body", errWithBody.ResponseBody())
	}

	// Test WithError method
	testErr := errors.New("test error")
	errWithError := err.WithError(testErr)
	if errWithError.Unwrap() != testErr {
		t.Error("Unwrap() should return the wrapped error")
	}

	// Test Error() method with different combinations
	t.Run("Error method combinations", func(t *testing.T) {
		// Case 1: Only status code (no body, no error)
		err1 := newClientError(req, &http.Response{StatusCode: 404})
		expected1 := "statuscode=404"
		if err1.Error() != expected1 {
			t.Errorf("expected %q, got %q", expected1, err1.Error())
		}

		// Case 2: Status code with body
		err2 := newClientError(req, &http.Response{StatusCode: 400}).WithBody([]byte("bad request"))
		expected2 := "statuscode=400, body=bad request"
		if err2.Error() != expected2 {
			t.Errorf("expected %q, got %q", expected2, err2.Error())
		}

		// Case 3: Status code with error
		err3 := newClientError(req, &http.Response{StatusCode: 500}).WithError(errors.New("server error"))
		expected3 := "statuscode=500, err=server error"
		if err3.Error() != expected3 {
			t.Errorf("expected %q, got %q", expected3, err3.Error())
		}

		// Case 4: Status code with both body and error
		err4 := newClientError(req, &http.Response{StatusCode: 503}).
			WithBody([]byte("service unavailable")).
			WithError(errors.New("timeout"))
		expected4 := "statuscode=503, body=service unavailable, err=timeout"
		if err4.Error() != expected4 {
			t.Errorf("expected %q, got %q", expected4, err4.Error())
		}
	})

	// Test that WithBody and WithError return new instances (immutability)
	t.Run("Immutability", func(t *testing.T) {
		baseErr := newClientError(req, &http.Response{StatusCode: 400})

		// Apply WithBody
		err1 := baseErr.WithBody([]byte("body1"))
		if baseErr.ResponseBody() != "" {
			t.Error("base error should not be modified by WithBody")
		}
		_ = err1 // Use the variable

		// Apply WithError
		err2 := baseErr.WithError(errors.New("error1"))
		if baseErr.Unwrap() != nil {
			t.Error("base error should not be modified by WithError")
		}
		_ = err2 // Use the variable

		// Chain methods
		err3 := baseErr.WithBody([]byte("body2")).WithError(errors.New("error2"))
		if baseErr.ResponseBody() != "" || baseErr.Unwrap() != nil {
			t.Error("base error should not be modified by chained methods")
		}
		if err3.ResponseBody() != "body2" || err3.Unwrap().Error() != "error2" {
			t.Error("chained methods should work correctly")
		}
	})
}
