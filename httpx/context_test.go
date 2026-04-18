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
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xgfone/go-toolkit/codeint"
	"github.com/xgfone/go-toolkit/result"
)

func TestContext_AppendError(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test appending first error
	err1 := errors.New("error1")
	ctx.AppendError(err1)
	if ctx.Error != err1 {
		t.Errorf("expected error1, got %v", ctx.Error)
	}

	// Test appending second error
	err2 := errors.New("error2")
	ctx.AppendError(err2)
	if !errors.Is(ctx.Error, err1) || !errors.Is(ctx.Error, err2) {
		t.Errorf("expected joined error containing both errors, got %v", ctx.Error)
	}

	// Test appending nil error
	ctx.AppendError(nil)
	if !errors.Is(ctx.Error, err1) || !errors.Is(ctx.Error, err2) {
		t.Errorf("error should not change when appending nil, got %v", ctx.Error)
	}
}

func TestContext_SetContentType(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	ctx.SetContentType("application/json")
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
	}
}

func TestContext_SetContentDisposition(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test inline disposition
	ctx.SetContentDisposition("inline", "")
	if disp := rec.Header().Get("Content-Disposition"); disp != "Content-Disposition: inline" {
		t.Errorf("expected 'Content-Disposition: inline', got '%s'", disp)
	}

	// Test attachment disposition without filename
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	ctx.SetContentDisposition("attachment", "")
	if disp := rec.Header().Get("Content-Disposition"); disp != "Content-Disposition: attachment" {
		t.Errorf("expected 'Content-Disposition: attachment', got '%s'", disp)
	}

	// Test attachment disposition with filename
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	ctx.SetContentDisposition("attachment", "test.jpg")
	if disp := rec.Header().Get("Content-Disposition"); disp != "attachment; filename=test.jpg" {
		t.Errorf("expected 'attachment; filename=test.jpg', got '%s'", disp)
	}

	// Test attachment disposition with filename containing special characters
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	ctx.SetContentDisposition("attachment", "test file with spaces.jpg")
	if disp := rec.Header().Get("Content-Disposition"); disp != "attachment; filename=\"test file with spaces.jpg\"" {
		t.Errorf("expected 'attachment; filename=\"test file with spaces.jpg\"', got '%s'", disp)
	}

	// Test panic with invalid disposition type
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid disposition type")
		}
	}()
	ctx.SetContentDisposition("invalid", "")
}

func TestContext_Redirect(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test valid redirect
	ctx.Redirect(301, "https://example.com")
	if loc := rec.Header().Get("Location"); loc != "https://example.com" {
		t.Errorf("expected Location 'https://example.com', got '%s'", loc)
	}
	if rec.Code != 301 {
		t.Errorf("expected status code 301, got %d", rec.Code)
	}

	// Test panic with invalid status code (too low)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid redirect status code 200")
		}
	}()
	ctx.Redirect(200, "https://example.com")

	// Test panic with invalid status code (too high)
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid redirect status code 400")
		}
	}()
	ctx.Redirect(400, "https://example.com")

	// Test valid redirect status codes
	validCodes := []int{300, 301, 302, 303, 304, 305, 306, 307, 308}
	for _, code := range validCodes {
		rec = httptest.NewRecorder()
		ctx.ResponseWriter = rec
		ctx.Redirect(code, "https://example.com")
		if rec.Code != code {
			t.Errorf("expected status code %d, got %d", code, rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "https://example.com" {
			t.Errorf("expected Location 'https://example.com', got '%s'", loc)
		}
	}
}

func TestContext_NoContent(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	ctx.NoContent(204)
	if rec.Code != 204 {
		t.Errorf("expected status code 204, got %d", rec.Code)
	}
}

func TestContext_JSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test successful JSON response
	data := map[string]string{"message": "test"}
	ctx.JSON(200, data)
	if rec.Code != 200 {
		t.Errorf("expected status code 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type containing 'application/json', got '%s'", ct)
	}
	if !strings.Contains(rec.Body.String(), "test") {
		t.Errorf("expected body containing 'test', got '%s'", rec.Body.String())
	}
}

func TestContext_Stream(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test streaming response
	data := "streaming data"
	reader := strings.NewReader(data)
	ctx.Stream(200, reader)
	if rec.Code != 200 {
		t.Errorf("expected status code 200, got %d", rec.Code)
	}
	if rec.Body.String() != data {
		t.Errorf("expected body '%s', got '%s'", data, rec.Body.String())
	}
}

func TestContext_Success(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test Success method
	data := "success data"
	ctx.Success(data)
	if rec.Code != 200 {
		t.Errorf("expected status code 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type containing 'application/json', got '%s'", ct)
	}
}

func TestContext_Respond(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test Respond with data
	response := result.Ok("test data")
	ctx.Respond(response)
	if rec.Code != 200 {
		t.Errorf("expected status code 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type containing 'application/json', got '%s'", ct)
	}

	// Test Respond with error
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	response = result.Err(codeint.NewError(404))
	ctx.Respond(response)
	if rec.Code != 404 {
		t.Errorf("expected status code 404, got %d", rec.Code)
	}

	// Test Respond with no content
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	response = result.Response{}
	ctx.Respond(response)
	if rec.Code != 200 {
		t.Errorf("expected status code 200, got %d", rec.Code)
	}
}

func TestSetRespond(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test custom respond function
	called := false
	customRespond := func(c *Context, r result.Response) {
		called = true
		c.NoContent(201)
	}

	SetRespond(customRespond)
	defer func() {
		SetRespond(defaultRespond)
	}()

	ctx.Respond(result.Ok("test"))
	if !called {
		t.Errorf("custom respond function not called")
	}
	if rec.Code != 201 {
		t.Errorf("expected status code 201 from custom respond, got %d", rec.Code)
	}

	// Test panic when setting nil respond function
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when setting nil respond function")
		}
	}()
	SetRespond(nil)
}

func TestDefaultRespond(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test default respond with data
	response := result.Ok("test data")
	defaultRespond(ctx, response)
	if rec.Code != 200 {
		t.Errorf("expected status code 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected Content-Type containing 'application/json', got '%s'", ct)
	}

	// Test default respond with error
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	response = result.Err(errors.New("test error"))
	defaultRespond(ctx, response)
	if rec.Code != 500 {
		t.Errorf("expected status code 500 for generic error, got %d", rec.Code)
	}

	// Test default respond with no content
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	response = result.Response{}
	defaultRespond(ctx, response)
	if rec.Code != 200 {
		t.Errorf("expected status code 200 for no content, got %d", rec.Code)
	}
}

func TestRespondError(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Test with codeint.Error
	response := result.Response{Error: codeint.NewError(404)}
	respondError(ctx, response)
	if rec.Code != 404 {
		t.Errorf("expected status code 404, got %d", rec.Code)
	}

	// Test with *codeint.Error
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	err := codeint.NewError(400)
	response = result.Response{Error: &err}
	respondError(ctx, response)
	if rec.Code != 400 {
		t.Errorf("expected status code 400, got %d", rec.Code)
	}

	// Test with error implementing StatusCode()
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	customErr := &statusCodeError{code: 403}
	response = result.Response{Error: customErr}
	respondError(ctx, response)
	if rec.Code != 403 {
		t.Errorf("expected status code 403, got %d", rec.Code)
	}
	// Verify that the error was wrapped as codeint.ErrInternalServerError
	// The response body should contain the wrapped error
	body := rec.Body.String()
	if !strings.Contains(body, "Internal Server Error") {
		t.Errorf("expected error to be wrapped as internal server error, got body: %s", body)
	}

	// Test with generic error
	rec = httptest.NewRecorder()
	ctx.ResponseWriter = rec
	response = result.Response{Error: errors.New("generic error")}
	respondError(ctx, response)
	if rec.Code != 500 {
		t.Errorf("expected status code 500 for generic error, got %d", rec.Code)
	}
}

type statusCodeError struct {
	code int
}

func (e *statusCodeError) Error() string {
	return "status code error"
}

func (e *statusCodeError) StatusCode() int {
	return e.code
}

// Test that errors with StatusCode() method are properly wrapped
func TestRespondError_Wrapping(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	ctx := &Context{
		ResponseWriter: rec,
		Request:        req,
	}

	// Create a custom error type that implements both error and StatusCode()
	type customError struct {
		msg  string
		code int
	}

	// Test with a concrete type that implements the interface
	err := &customErrorWithStatusCode{
		msg:  "custom error message",
		code: 418,
	}

	response := result.Response{Error: err}
	respondError(ctx, response)

	// Should use the status code from the error
	if rec.Code != 418 {
		t.Errorf("expected status code 418, got %d", rec.Code)
	}

	// The error should be wrapped as codeint.ErrInternalServerError
	// Check that the response contains the wrapped error
	body := rec.Body.String()
	if !strings.Contains(body, "Internal Server Error") {
		t.Errorf("expected error to be wrapped as internal server error, got body: %s", body)
	}
}

type customErrorWithStatusCode struct {
	msg  string
	code int
}

func (e *customErrorWithStatusCode) Error() string {
	return e.msg
}

func (e *customErrorWithStatusCode) StatusCode() int {
	return e.code
}
