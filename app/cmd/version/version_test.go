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

package main

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name, pkg, ver string
		buildat        int64
		wantOK         bool
		checks         []string
		wantErr        string
	}{
		{"basic", "main", "v1.0.0", 1000000, true,
			[]string{`package main`, `AppVersion   = "v1.0.0"`, `AppBuildTime = 1000000`}, ""},
		{"custom_package", "version", "v2.0.0", 2000000, true,
			[]string{`package version`}, ""},
		{"empty_version", "main", "", 0, true,
			[]string{`AppVersion   = ""`}, ""},
		{"invalid_package_name", "123bad", "v1.0.0", 0, false, nil,
			`"123bad" is not an identifier`},
		{"version_contains_quote", "main", `v1."0".0`, 0, false, nil,
			`version contains '"'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := Generate(tt.pkg, tt.ver, tt.buildat)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				for _, c := range tt.checks {
					if !strings.Contains(code, c) {
						t.Errorf("missing %q", c)
					}
				}
			} else {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
				}
			}
		})
	}

	t.Run("valid_go_syntax", func(t *testing.T) {
		code, err := Generate("main", "v1.0.0", 1000000)
		if err != nil {
			t.Fatal(err)
		}
		_, err = parser.ParseFile(token.NewFileSet(), "", code, parser.AllErrors)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestGetBuildTime(t *testing.T) {
	if v := getBuildTime(); v <= 0 {
		t.Errorf("expected positive, got %d", v)
	}
}

func TestRun(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir, "v1.0.0")
		withChdir(t, dir)

		out := filepath.Join(t.TempDir(), "gen.go")
		if err := run(out, "main"); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(out)
		if !strings.Contains(string(data), `AppVersion   = "v1.0.0"`) {
			t.Error("output missing AppVersion")
		}
	})

	t.Run("git_error", func(t *testing.T) {
		dir := t.TempDir() // no git repo
		withChdir(t, dir)
		if err := run(filepath.Join(dir, "out.go"), "main"); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("invalid_package_name", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir, "v1.0.0")
		withChdir(t, dir)

		err := run(filepath.Join(dir, "out.go"), "123bad")
		if err == nil || !strings.Contains(err.Error(), "not an identifier") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestMainFunc(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir, "v1.0.0")
		withChdir(t, dir)

		*output = filepath.Join(dir, "gen.go")
		main()

		data, _ := os.ReadFile(*output)
		if !strings.Contains(string(data), `AppVersion   = "v1.0.0"`) {
			t.Error("output missing AppVersion")
		}
	})

	t.Run("exit_on_error", func(t *testing.T) {
		dir := t.TempDir() // no git repo
		withChdir(t, dir)

		var exited bool
		old := osexit
		osexit = func(int) { exited = true }
		defer func() { osexit = old }()

		*output = filepath.Join(dir, "gen.go")
		main()
		if !exited {
			t.Error("os.Exit was not called")
		}
	})
}

// -- helpers --

func initGitRepo(t *testing.T, dir, tag string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "test"},
		{"commit", "--allow-empty", "-m", "init"},
		{"tag", tag},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func withChdir(t *testing.T, dir string) {
	t.Helper()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}
