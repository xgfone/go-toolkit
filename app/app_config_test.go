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
	"slices"
	"testing"
)

func TestSetConfigLoaderReplacesPreviousLoader(t *testing.T) {
	a := newTestRunApp()
	var events []string
	a.SetConfigLoader(func(context.Context) error {
		t.Error("replaced loader must not run")
		return nil
	})
	a.SetConfigLoader(func(_ context.Context) error {
		events = append(events, "config")
		// Configuration may register even the first public stage.
		a.On(StageInit, "", func(context.Context) error {
			events = append(events, "init")
			return nil
		})
		return nil
	})

	a.On(StageReady, "", func(_ context.Context) error { a.Stop(); return nil })
	if err := a.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"config", "init"}; !slices.Equal(events, want) {
		t.Fatalf("startup = %v, want %v", events, want)
	}
}

func TestDefaultFlagConfigLoader(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		parsed   bool
		wantErr  error
		wantExit []int
	}{
		{name: "parse"},
		{name: "already parsed", parsed: true, args: []string{"--ignored"}},
		{name: "help", args: []string{"-h"}, wantErr: flag.ErrHelp, wantExit: []int{0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			originalFlags := flag.CommandLine
			originalArgs := os.Args
			originalExit := osexit
			t.Cleanup(func() {
				flag.CommandLine = originalFlags
				os.Args = originalArgs
				osexit = originalExit
			})
			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
			if tc.parsed {
				if err := flag.CommandLine.Parse(nil); err != nil {
					t.Fatal(err)
				}
			}
			os.Args = append([]string{"test"}, tc.args...)
			var exitCodes []int
			osexit = func(code int) { exitCodes = append(exitCodes, code) }
			a := New()
			a.SetSignals()
			a.On(StageReady, "stop", func(context.Context) error { a.Stop(); return nil })
			if err := a.Run(context.Background()); !errors.Is(err, tc.wantErr) {
				t.Fatalf("Run error = %v, want %v", err, tc.wantErr)
			}
			if !flag.CommandLine.Parsed() || !slices.Equal(exitCodes, tc.wantExit) {
				t.Fatalf("parsed=%v, exit codes=%v, want %v", flag.CommandLine.Parsed(), exitCodes, tc.wantExit)
			}
		})
	}
}
