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

package validation

import (
	"context"
	"io"

	"github.com/xgfone/go-toolkit/jsonx"
)

// BindJSONBytes unmarshals the JSON bytes into out and validates it.
func BindJSONBytes(ctx context.Context, in []byte, out any) (err error) {
	if err = jsonx.UnmarshalBytes(in, out); err == nil {
		err = Validate(ctx, out)
	}
	return
}

// BindJSONString unmarshals the JSON string into out and validates it.
func BindJSONString(ctx context.Context, in string, out any) (err error) {
	if err = jsonx.UnmarshalString(in, out); err == nil {
		err = Validate(ctx, out)
	}
	return
}

// BindJSONReader reads JSON from the io.Reader, unmarshals it into out, and validates it.
func BindJSONReader(ctx context.Context, in io.Reader, out any) (err error) {
	if err = jsonx.UnmarshalReader(out, in); err == nil {
		err = Validate(ctx, out)
	}
	return
}
