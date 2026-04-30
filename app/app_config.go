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

// ConfigLoader is used to load or parse app configuration before app starts.
type ConfigLoader func(ctx context.Context, app *App) error

// SetConfigLoader replaces the default ConfigLoader.
//
// It must be called before Run.
func (a *App) SetConfigLoader(loader ConfigLoader) {
	if loader == nil {
		panic("app: nil ConfigLoader")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetConfigLoader")
	a.configLoader = loader
}

func defaultFlagConfigLoader(ctx context.Context, app *App) (err error) {
	if flag.CommandLine.Parsed() {
		return
	}

	flag.CommandLine.Init(app.Name(), flag.ContinueOnError)
	if err = flag.CommandLine.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			osexit(0)
		}
	}

	return
}

var osexit = os.Exit
