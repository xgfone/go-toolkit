package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func ExampleGet() {
	// Mock Client
	SetClient(DoFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"Name":"Aaron","Id":123}`)),
		}, nil
	}))

	// Struct
	{
		var resp struct {
			Name string
			Id   int64
		}

		err := Get(context.Background(), "http://127.0.0.1", &resp)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Id=%d, Name=%s\n", resp.Id, resp.Name)
		}
	}

	// Function
	{
		var resp struct {
			Name string
			Id   int64
		}

		respf := func(r *http.Response) error {
			return json.NewDecoder(r.Body).Decode(&resp)
		}

		err := Get(context.Background(), "http://127.0.0.1", respf)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("Id=%d, Name=%s\n", resp.Id, resp.Name)
		}
	}

	// Output:
	// Id=123, Name=Aaron
	// Id=123, Name=Aaron
}

func ExamplePost() {
	// Mock Client
	SetClient(&mockClient{doFunc: func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"Status":"Success"}`)),
		}, nil
	}})

	type Request struct {
		Action string
		Name   string
	}

	var resp struct {
		Status string
	}

	req := Request{Action: "Update", Name: "Aaron"}
	err := Post(context.Background(), "http://127.0.0.1", &resp, req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Status: %s\n", resp.Status)
	}

	// Output:
	// Status: Success
}
