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

package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/xgfone/go-toolkit/httpx"
	"github.com/xgfone/go-toolkit/runtimex"
)

// Recover is an http handler middleware to recover the panic if occurring.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wrappanic(w, r)
		next.ServeHTTP(w, r)
	})
}

func wrappanic(w http.ResponseWriter, r *http.Request) {
	v := recover()
	if v == nil {
		return
	}

	stacks := runtimex.Stacks(2)
	slog.Error("wrap a panic", slog.Any("panic", v), slog.Any("stacks", stacks))

	if c := httpx.GetContext(r.Context()); c != nil {
		c.AppendError(panicerror{stacks: stacks, panicv: v})
	} else if rw, ok := w.(httpx.ResponseWriter); !ok || rw.StatusCode() == 0 {
		http.Error(w, "panic", 500)
	}
}

type panicerror struct {
	stacks []runtimex.Frame
	panicv any
}

func (e panicerror) Error() string            { return fmt.Sprintf("panic: %v", e.panicv) }
func (e panicerror) Stacks() []runtimex.Frame { return e.stacks }
