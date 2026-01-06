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

import "fmt"

var respond func(responder any, response Response) = defaultRespond

// Respond sends the response via the given responder.
//
// The actual logic is delegated to the global respond function,
// which can be overridden by calling SetRespondFunc.
//
// By default, the responder must implement one of:
//
//	interface{ Respond(Response) }
//	interface{ JSON(code int, value any) }
func Respond(responder any, response Response) {
	respond(responder, response)
}

// GetRespondFunc returns the global response-sending function used by Respond.
func GetRespondFunc() func(responder any, response Response) {
	return respond
}

// SetRespondFunc replaces the global response-sending function used by Respond.
//
// It panics if f is nil.
func SetRespondFunc(f func(responder any, response Response)) {
	if f == nil {
		panic("result.SetRespondFunc: the function cannot be nil")
	}

	respond = f
}

func defaultRespond(responder any, response Response) {
	switch resp := responder.(type) {
	case interface{ Respond(Response) }:
		resp.Respond(response)

	case interface{ JSON(code int, value any) }:
		if response.IsZero() {
			resp.JSON(response.StatusCode(), nil)
		} else {
			resp.JSON(response.StatusCode(), response)
		}

	default:
		panic(fmt.Errorf("result.DefaultRespond: unknown responder type %T", responder))
	}
}
