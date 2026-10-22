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

package app_test

import (
	"context"
	"fmt"

	"github.com/xgfone/go-toolkit/app"
)

func ExampleApp_On() {
	a := app.New()
	a.SetName("example")
	a.SetSignals()
	a.SetConfigLoader(func(context.Context) error { return nil })
	a.On(app.StageInit, "initialize", func(ctx context.Context) error {
		// Capture the App; use the supplied context for this execution.
		fmt.Println(a.Name(), "initialized:", ctx.Err() == nil)
		return nil
	})
	a.On(app.StageReady, "stop", func(context.Context) error {
		a.Stop()
		return nil
	})
	a.On(app.StageCleanup, "cleanup", func(ctx context.Context) error {
		fmt.Println("stopping:", a.Context().Err() != nil)
		fmt.Println("cleanup context active:", ctx.Err() == nil)
		return nil
	})

	if err := a.Run(context.Background()); err != nil {
		panic(err)
	}

	<-a.Done()
	fmt.Println("shutdown finished")

	// Output:
	// example initialized: true
	// stopping: true
	// cleanup context active: true
	// shutdown finished
}

func ExampleApp_Go() {
	a := app.New()
	a.SetSignals()
	a.SetConfigLoader(func(context.Context) error { return nil })
	workerExited := make(chan struct{})
	a.On(app.StageStart, "start-worker", func(context.Context) error {
		// Capture the owning App to start a task managed by its lifecycle.
		a.Go("worker", func(ctx context.Context) error {
			defer close(workerExited)
			<-ctx.Done()
			return nil
		})
		return nil
	})
	a.On(app.StageReady, "stop", func(context.Context) error {
		a.Stop()
		return nil
	})

	if err := a.Run(context.Background()); err != nil {
		panic(err)
	}

	<-workerExited
	fmt.Println("background task stopped before Run returned")

	// Output:
	// background task stopped before Run returned
}
