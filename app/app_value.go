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

package app

import "fmt"

// ValueFunc is a dynamic value provider.
//
// Values registered by SetFunc will be called every time Get is called.
type ValueFunc func() any

// Get is the similar to App.Get, but uses the package-level DefaultApp
// and asserts the value to T.
func Get[T any](key string) (T, bool) {
	var zero T

	v, ok := DefaultApp.Get(key)
	if !ok {
		return zero, false
	}

	t, ok := v.(T)
	if !ok {
		return zero, false
	}

	return t, ok
}

// MustGet is like Get, but panics if the value is not found or type mismatch.
func MustGet[T any](key string) T {
	v, ok := DefaultApp.Get(key)
	if !ok {
		panic(fmt.Sprintf("app: value %q not found", key))
	}

	t, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("app: value %q type mismatch: got %T", key, v))
	}

	return t
}

// Set stores a static value.
//
// If value is a function and should be executed on Get, use SetFunc instead.
func (a *App) Set(key string, value any) {
	if key == "" {
		panic("app: empty value key")
	}

	if value == nil {
		panic("app: nil value")
	}

	a.values.Store(key, value)
}

// SetFunc stores a dynamic value provider.
//
// The provider is called every time Get is called.
func (a *App) SetFunc(key string, fn func() any) {
	if key == "" {
		panic("app: empty value key")
	}

	if fn == nil {
		panic("app: nil value func")
	}

	a.values.Store(key, ValueFunc(fn))
}

// Get returns a stored value.
//
// If the value was registered by SetFunc, Get calls the provider and returns
// its current result.
func (a *App) Get(key string) (any, bool) {
	if key == "" {
		panic("app: empty value key")
	}

	v, ok := a.values.Load(key)
	if !ok {
		return nil, false
	}

	if fn, ok := v.(ValueFunc); ok {
		v = fn()
	}

	return v, true
}
