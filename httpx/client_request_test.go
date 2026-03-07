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
	"log/slog"
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

// Request builder interfaces for testing
type simpleRequestBuilder struct {
	customHeader string
}

func (b *simpleRequestBuilder) NewRequest(ctx context.Context, method string, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	if b.customHeader != "" {
		req.Header.Set("X-Custom", b.customHeader)
	}
	return req, nil
}

type cleanupRequestBuilder struct {
	customHeader  string
	cleanupCalled bool
}

func (b *cleanupRequestBuilder) NewRequest(ctx context.Context, method string, url string) (*http.Request, func(), error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, nil, err
	}
	if b.customHeader != "" {
		req.Header.Set("X-Custom", b.customHeader)
	}
	return req, func() { b.cleanupCalled = true }, nil
}

type errorRequestBuilder struct{}

func (b *errorRequestBuilder) NewRequest(ctx context.Context, method string, url string) (*http.Request, error) {
	return nil, errors.New("builder error")
}

type errorCleanupRequestBuilder struct {
	cleanupCalled bool
}

func (b *errorCleanupRequestBuilder) NewRequest(ctx context.Context, method string, url string) (*http.Request, func(), error) {
	cleanup := func() {
		b.cleanupCalled = true
	}
	return nil, cleanup, errors.New("builder error")
}

type bodyRequestBuilder struct{}

func (b *bodyRequestBuilder) NewRequest(ctx context.Context, method string, url string) (*http.Request, error) {
	body := bytes.NewBufferString(`{"name":"builder","value":100}`)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", MIMEApplicationJSON)
	return req, nil
}

// Mock HTTP client
type mockClient struct {
	doFunc func(*http.Request) (*http.Response, error)
}

func (m *mockClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))}, nil
}

// Error reader for testing
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

// Test Request function with nil request
func TestRequest_NilRequest(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Errorf("expected method GET, got %s", req.Method)
			}
			if req.Body != nil {
				t.Error("expected nil body for nil request")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"123","name":"test","value":42}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodGet, "http://example.com/api", &resp, nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.ID != "123" || resp.Name != "test" || resp.Value != 42 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

// Test Request function with io.Reader request
func TestRequest_ReaderRequest(t *testing.T) {
	requestBody := `{"name":"reader","value":99}`
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Errorf("expected method POST, got %s", req.Method)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			if string(body) != requestBody {
				t.Errorf("expected body %q, got %q", requestBody, string(body))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"456","name":"reader","value":99}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var resp testResponse
	ctx := context.Background()
	reader := strings.NewReader(requestBody)
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, reader)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.ID != "456" || resp.Name != "reader" || resp.Value != 99 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

// Test Request function with struct request (JSON encoded)
func TestRequest_StructRequest(t *testing.T) {
	reqData := testRequest{Name: "struct", Value: 77}
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPut {
				t.Errorf("expected method PUT, got %s", req.Method)
			}

			contentType := req.Header.Get("Content-Type")
			if contentType != MIMEApplicationJSON {
				t.Errorf("expected Content-Type %s, got %s", MIMEApplicationJSON, contentType)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			var reqBody testRequest
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if reqBody.Name != "struct" || reqBody.Value != 77 {
				t.Errorf("unexpected request body: %+v", reqBody)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"789","name":"struct","value":77}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodPut, "http://example.com/api", &resp, reqData)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.ID != "789" || resp.Name != "struct" || resp.Value != 77 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

// Test Request function with request function
func TestRequest_RequestFunction(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodDelete {
				t.Errorf("expected method DELETE, got %s", req.Method)
			}
			if req.Header.Get("X-Custom") != "custom-value" {
				t.Error("expected X-Custom header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"status":"deleted"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqFunc := func(ctx context.Context, method string, url string) (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Custom", "custom-value")
		return req, nil
	}

	var resp map[string]string
	ctx := context.Background()
	err := Request(ctx, http.MethodDelete, "http://example.com/api", &resp, reqFunc)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp["status"] != "deleted" {
		t.Errorf("unexpected response: %v", resp)
	}
}

// Test Request function with request function that returns cleanup
func TestRequest_RequestFunctionWithCleanup(t *testing.T) {
	cleanupCalled := false
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	reqFunc := func(ctx context.Context, method string, url string) (*http.Request, func(), error) {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, nil, err
		}
		return req, func() { cleanupCalled = true }, nil
	}

	var resp map[string]string
	ctx := context.Background()
	err := Request(ctx, http.MethodPatch, "http://example.com/api", &resp, reqFunc)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if !cleanupCalled {
		t.Error("cleanup function should have been called")
	}
	if resp["status"] != "ok" {
		t.Errorf("unexpected response: %v", resp)
	}
}

// Test Request function with request builder interface
func TestRequest_RequestBuilderInterface(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Custom") != "builder-header" {
				t.Error("expected X-Custom header from builder")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"builder"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	builder := &simpleRequestBuilder{customHeader: "builder-header"}
	var resp map[string]string
	ctx := context.Background()
	err := Request(ctx, http.MethodHead, "http://example.com/api", &resp, builder)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp["id"] != "builder" {
		t.Errorf("unexpected response: %v", resp)
	}
}

// Test Request function with request builder interface that returns cleanup
func TestRequest_RequestBuilderInterfaceWithCleanup(t *testing.T) {
	builder := &cleanupRequestBuilder{customHeader: "cleanup-builder"}
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Custom") != "cleanup-builder" {
				t.Error("expected X-Custom header from builder")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"cleanup"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var resp map[string]string
	ctx := context.Background()
	err := Request(ctx, http.MethodOptions, "http://example.com/api", &resp, builder)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if !builder.cleanupCalled {
		t.Error("cleanup should have been called")
	}
	if resp["id"] != "cleanup" {
		t.Errorf("unexpected response: %v", resp)
	}
}

// Test Request function with response function
func TestRequest_ResponseFunction(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"custom":"response"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var capturedResponse *http.Response
	respFunc := func(rsp *http.Response) error {
		capturedResponse = rsp
		body, err := io.ReadAll(rsp.Body)
		if err != nil {
			return err
		}
		if string(body) != `{"custom":"response"}` {
			return fmt.Errorf("unexpected body: %s", string(body))
		}
		return nil
	}

	ctx := context.Background()
	err := Request(ctx, http.MethodGet, "http://example.com/api", respFunc, nil)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if capturedResponse == nil {
		t.Error("response function should have received the response")
	}
}

// Test Request function with non-200 status code
func TestRequest_Non200StatusCode(t *testing.T) {
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

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, nil)

	if err == nil {
		t.Fatal("expected error for non-200 status code")
	}

	// Check that error contains status code
	if !strings.Contains(err.Error(), "statuscode=400") {
		t.Errorf("error should contain 'statuscode=400', got: %v", err)
	}
}

// Test Request function with HTTP request failure
func TestRequest_HttpRequestFailed(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, nil)

	if err == nil {
		t.Fatal("expected error for HTTP request failure")
	}
	if !strings.Contains(err.Error(), "network error") {
		t.Errorf("error should contain 'network error', got: %v", err)
	}
}

// Test Request function with invalid URL
func TestRequest_InvalidURL(t *testing.T) {
	SetClient(&mockClient{})
	defer SetClient(http.DefaultClient)

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodGet, "://invalid-url", &resp, nil)

	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// Test Request function with JSON encode failure
func TestRequest_JsonEncodeFailed(t *testing.T) {
	// Create a value that cannot be JSON encoded
	unencodable := func() {}
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP request should not be made")
			return nil, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, unencodable)

	if err == nil {
		t.Fatal("expected error for JSON encode failure")
	}
	if !strings.Contains(err.Error(), "fail to encode request body") {
		t.Errorf("error should contain 'fail to encode request body', got: %v", err)
	}
}

// Test Request function with JSON decode failure
func TestRequest_JsonDecodeFailed(t *testing.T) {
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

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, nil)

	if err == nil {
		t.Fatal("expected error for JSON decode failure")
	}
	if !strings.Contains(err.Error(), "fail to decode the response body") {
		t.Errorf("error should contain 'fail to decode the response body', got: %v", err)
	}
}

// Test Request function with read response body failure
func TestRequest_ReadResponseBodyFailed(t *testing.T) {
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

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, nil)

	if err == nil {
		t.Fatal("expected error for read response body failure")
	}
	if !strings.Contains(err.Error(), "fail to read the response body") {
		t.Errorf("error should contain 'fail to read the response body', got: %v", err)
	}
}

// Test Request function with nil response
func TestRequest_NilResponse(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"test"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	ctx := context.Background()
	err := Request(ctx, http.MethodGet, "http://example.com/api", nil, nil)

	if err != nil {
		t.Fatalf("Request with nil response failed: %v", err)
	}
}

// Test Request function with empty response body
func TestRequest_EmptyResponseBody(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodGet, "http://example.com/api", &resp, nil)

	if err != nil {
		t.Fatalf("Request with empty response body failed: %v", err)
	}
	// Response should remain zero-valued
	if resp.ID != "" || resp.Name != "" || resp.Value != 0 {
		t.Errorf("expected zero-valued response, got: %+v", resp)
	}
}

// Test Request function with request builder that has body
func TestRequest_RequestBuilderWithBody(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			contentType := req.Header.Get("Content-Type")
			if contentType != MIMEApplicationJSON {
				t.Errorf("expected Content-Type %s, got %s", MIMEApplicationJSON, contentType)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			if string(body) != `{"name":"builder","value":100}` {
				t.Errorf("unexpected request body: %s", string(body))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"body-builder"}`)),
			}, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	builder := &bodyRequestBuilder{}
	var resp map[string]string
	ctx := context.Background()
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, builder)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp["id"] != "body-builder" {
		t.Errorf("unexpected response: %v", resp)
	}
}

// Test Request function with error request builder
func TestRequest_ErrorRequestBuilder(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP request should not be made")
			return nil, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	builder := &errorRequestBuilder{}
	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, builder)

	if err == nil {
		t.Fatal("expected error for builder error")
	}
	if !strings.Contains(err.Error(), "builder error") {
		t.Errorf("error should contain 'builder error', got: %v", err)
	}
}

// Test Request function with error cleanup request builder
func TestRequest_ErrorCleanupRequestBuilder(t *testing.T) {
	client := &mockClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP request should not be made")
			return nil, nil
		},
	}

	SetClient(client)
	defer SetClient(http.DefaultClient)

	builder := &errorCleanupRequestBuilder{}
	var resp testResponse
	ctx := context.Background()
	err := Request(ctx, http.MethodPost, "http://example.com/api", &resp, builder)

	if err == nil {
		t.Fatal("expected error for builder error")
	}
	if !strings.Contains(err.Error(), "builder error") {
		t.Errorf("error should contain 'builder error', got: %v", err)
	}
}

// Test Request function with response function error
func TestRequest_ResponseFunctionError(t *testing.T) {
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

	expectedError := errors.New("response function error")
	respFunc := func(rsp *http.Response) error {
		return expectedError
	}

	ctx := context.Background()
	err := Request(ctx, http.MethodGet, "http://example.com/api", respFunc, nil)

	if err == nil {
		t.Fatal("expected error from response function")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}
}

// Test Request function with various HTTP methods
func TestRequest_VariousMethods(t *testing.T) {
	testCases := []struct {
		method string
	}{
		{http.MethodGet},
		{http.MethodPost},
		{http.MethodPut},
		{http.MethodDelete},
		{http.MethodPatch},
		{http.MethodHead},
		{http.MethodOptions},
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			client := &mockClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					if req.Method != tc.method {
						t.Errorf("expected method %s, got %s", tc.method, req.Method)
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"method":"` + tc.method + `"}`)),
					}, nil
				},
			}

			SetClient(client)
			defer SetClient(http.DefaultClient)

			var resp map[string]string
			ctx := context.Background()
			err := Request(ctx, tc.method, "http://example.com/api", &resp, nil)

			if err != nil {
				t.Fatalf("Request with method %s failed: %v", tc.method, err)
			}
			if resp["method"] != tc.method {
				t.Errorf("expected response method %s, got %s", tc.method, resp["method"])
			}
		})
	}
}

// Test Request function with the debug logging
func TestRequest_DebugLogging(t *testing.T) {
	origLogger := slog.Default()
	defer slog.SetDefault(origLogger)

	buf := new(bytes.Buffer)
	logOptions := slog.HandlerOptions{Level: slog.LevelDebug}
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &logOptions)))

	SetClient(DoFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`test`)),
		}, nil
	}))

	err := Request(context.Background(), http.MethodGet, "http://127.0.0.1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if s := buf.String(); !strings.Contains(s, "respbody=test") {
		t.Errorf("log: expect to contain '%s', but got '%s'", "respbody=test", s)
	}
}
