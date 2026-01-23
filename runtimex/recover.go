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

package runtimex

import (
	"context"
	"log/slog"
)

// Recover is a convenient function to wrap and recover the panic if occurring.
//
// NOTICE: It must be called after defer, like
//
//	defer Recover(context.Background())
func Recover(ctx context.Context, logargs ...any) {
	if r := recover(); r != nil {
		logargs = append(logargs,
			slog.Any("panic", r),
			slog.Any("stacks", Stacks(2)),
		)
		slog.Error("wrap a panic", logargs...)
	}
}
