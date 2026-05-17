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
	"testing"
	"time"
)

func TestSafeRun(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var err error

		app := New()
		app.On(StageStart, func(_ context.Context, app *App) error {
			app.Go(func(context.Context) error { return nil })
			return nil
		})

		go func() {
			err = app.Run(context.Background())
		}()

		time.Sleep(time.Millisecond * 100)
		app.Stop()
		app.Wait()

		if err != nil {
			t.Errorf("expect nil, but got an error: %s", err.Error())
		}
	})

	t.Run("panic with error", func(t *testing.T) {
		panicerr := errors.New("test")
		var err error

		app := New()
		app.On(StageStart, func(_ context.Context, app *App) error {
			app.Go(func(context.Context) error { panic(panicerr) })
			return nil
		})

		go func() {
			err = app.Run(context.Background())
		}()

		time.Sleep(time.Millisecond * 100)
		app.Stop()
		app.Wait()

		if err == nil {
			t.Fatal("expect an error, but got nil")
		}
		if !errors.Is(err, panicerr) {
			t.Fatalf("expect a panic error, but got a different error: %s", err.Error())
		}

		const expect = `app: background task "": panic: test`
		if s := err.Error(); s != expect {
			t.Fatalf("expect error message '%s', but got '%s'", expect, s)
		}
	})

	t.Run("panic without error", func(t *testing.T) {
		var err error

		app := New()
		app.On(StageStart, func(_ context.Context, app *App) error {
			app.Go(func(context.Context) error { panic("test") })
			return nil
		})

		go func() {
			err = app.Run(context.Background())
		}()

		time.Sleep(time.Millisecond * 100)
		app.Stop()
		app.Wait()

		if err == nil {
			t.Fatal("expect an error, but got nil")
		}

		const expect = `app: background task "": panic: test`
		if s := err.Error(); s != expect {
			t.Fatalf("expect error message '%s', but got '%s'", expect, s)
		}
	})
}
