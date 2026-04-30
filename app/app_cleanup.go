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

import (
	"context"
	"errors"
	"fmt"
)

type namedCleanup struct {
	name string
	fn   CleanupFunc
}

// CleanupFunc is a cleanup function registered by DeferCleanup.
//
// Cleanups are executed in reverse registration order.
type CleanupFunc func(ctx context.Context) error

// DeferCleanup registers a cleanup function.
//
// Cleanups are executed in reverse registration order.
// It can be called before Run or during running lifecycle,
// such as Module.Init.
//
// It cannot be called during or after shutdown.
func (a *App) DeferCleanup(name string, fn CleanupFunc) {
	if name == "" {
		panic("app: empty cleanup name")
	}

	if fn == nil {
		panic("app: nil cleanup func")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state == stateStopping || a.state == stateExited {
		panic("app: DeferCleanup cannot be called during or after shutdown")
	}

	a.cleanups = append(a.cleanups, namedCleanup{name: name, fn: fn})
}

func (a *App) runCleanups(ctx context.Context) error {
	a.mu.Lock()
	items := append([]namedCleanup(nil), a.cleanups...)
	a.cleanups = nil // Clear cleanups to avoid accidental repeated execution.
	a.mu.Unlock()

	var errs []error

	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]

		if err := item.fn(ctx); err != nil {
			errs = append(errs, fmt.Errorf("app: cleanup %q: %w", item.name, err))
		}
	}

	return errors.Join(errs...)
}
