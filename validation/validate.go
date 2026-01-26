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

// Package validation provides some validation functions.
package validation

import "context"

var _validate func(context.Context, any) error = _default

func _default(ctx context.Context, value any) error {
	if v, ok := value.(_Validator); ok {
		return v.Validate(ctx)
	}
	return nil
}

type _Validator interface {
	Validate(context.Context) error
}

// Validate validates whether the value is the valid,
// which can be overrided by SetValidateFunc.
func Validate(ctx context.Context, value any) error {
	return _validate(ctx, value)
}

// SetValidateFunc resets the global validation function,
// which will be used by Validate.
//
// If f is nil, it will panic.
func SetValidateFunc(f func(ctx context.Context, value any) error) {
	if f == nil {
		panic("SetValidateFunc: the validate function must not be nil")
	}
	_validate = f
}
