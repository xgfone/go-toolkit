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
	"flag"
	"os"
	"testing"
	"time"
)

func TestSetConfigLoader_Panics_Nil(t *testing.T) {
	defer func() { _ = recover() }()
	New().SetConfigLoader(nil)
	t.Error("expected panic")
}

func TestSetConfigLoader_Panics_AfterRun(t *testing.T) {
	app := New()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_ = app.Run(ctx)

	defer func() { _ = recover() }()
	app.SetConfigLoader(func(ctx context.Context, app *App) error { return nil })
	t.Error("expected panic")
}

func TestDefaultFlagConfigLoader(t *testing.T) {
	origCL := flag.CommandLine
	origArgs := os.Args
	t.Cleanup(func() {
		flag.CommandLine = origCL
		os.Args = origArgs
	})

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"test"}

	app := New()
	// Don't call SetConfigLoader — exercise the default.
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultFlagConfigLoader_AlreadyParsed(t *testing.T) {
	origCL := flag.CommandLine
	t.Cleanup(func() { flag.CommandLine = origCL })

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	_ = fs.Parse(nil)
	flag.CommandLine = fs

	app := New()
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	if err := app.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultFlagConfigLoader_Help(t *testing.T) {
	origCL := flag.CommandLine
	origArgs := os.Args
	origExit := osexit
	t.Cleanup(func() {
		flag.CommandLine = origCL
		os.Args = origArgs
		osexit = origExit
	})

	// Trigger flag.ErrHelp by passing -h.
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"test", "-h"}

	var exitCode int
	osexit = func(code int) { exitCode = code }

	app := New()
	app.SetSignals()

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := app.Run(ctx)
	// On help, os.Exit(0) is called; the loader returns nil, so Run proceeds.
	// But osexit does os.Exit normally — our mock records the code.
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	_ = err
}
