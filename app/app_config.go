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
	"flag"
	"os"
)

// SetConfigLoader replaces the config loader.
//
// It must be called before [App.Run].
func (a *App) SetConfigLoader(loader func(ctx context.Context) error) {
	if loader == nil {
		panic("app: nil config loader")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetConfigLoader")
	a.callbacks[phaseConfig] = nil
	a.registerCallbackLocked(phaseConfig, "", loader)
}

func (a *App) defaultFlagConfigLoader(context.Context) error {
	if flag.CommandLine.Parsed() {
		return nil
	}

	flag.CommandLine.Init(a.Name(), flag.ContinueOnError)
	err := flag.CommandLine.Parse(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		osexit(0)
	}

	return err
}

var osexit = os.Exit
