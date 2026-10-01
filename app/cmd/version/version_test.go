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

func TestGetVersion(t *testing.T) {
	commit := []string{"commit", "--allow-empty", "-m", "next"}
	tests := []struct {
		name     string
		tag      string
		prefix   string
		want     string
		commands [][]string
	}{
		{"at_tag", "v1.0.0", "v", "v1.0.0", nil},
		{"one_commit", "v1.0.0", "v", "v1.0.0-1", [][]string{commit}},
		{"multiple_commits", "v1.0.0", "v", "v1.0.0-2", [][]string{commit, commit}},
		{"annotated_tag", "", "v", "v1.0.0-2", [][]string{{"tag", "-a", "v1.0.0", "-m", "release"}, commit, commit}},
		{"custom_prefix", "release-1.0.0", "release-", "release-1.0.0-1", [][]string{commit}},
		{"highest_version_tag", "v1.10.0", "v", "v1.10.0-2", [][]string{commit, commit, {"tag", "v1.9.0"}}},
		{"branch_with_same_name", "v1.0.0", "v", "v1.0.0-1", [][]string{commit, {"branch", "v1.0.0"}}},
		{"merge_history", "v1.0.0", "v", "v1.0.0-3", [][]string{
			{"checkout", "-b", "feature"},
			{"commit", "--allow-empty", "-m", "feature"},
			{"checkout", "-"}, commit,
			{"merge", "--no-ff", "-m", "merge feature", "feature"},
		}},
		{"no_tag", "", "v", "", nil},
		{"no_matching_tag", "release-1.0.0", "v", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			initGitRepo(t, dir, tt.tag)
			for _, args := range tt.commands {
				cmd := exec.Command("git", args...)
				cmd.Dir = dir
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %v: %v\n%s", args, err, out)
				}
			}

			withChdir(t, dir)
			got, err := getVersion(tt.prefix)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("getVersion(%q) = %q, want %q", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestGetVersionWithoutHead(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir, "v1.0.0")

	// Keep the tag, but move to a branch with no commits: listing tags succeeds
	// while counting commits relative to HEAD fails.
	cmd := exec.Command("git", "checkout", "--orphan", "unborn")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create orphan branch: %v\n%s", err, out)
	}
	withChdir(t, dir)

	got, err := getVersion("v")
	if err == nil {
		t.Fatal("expected an error for HEAD without a commit")
	}
	if got != "" {
		t.Errorf("getVersion returned %q on error, want an empty version", got)
	}
}

func TestRun(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir, "v1.0.0")
		withChdir(t, dir)

		out := filepath.Join(t.TempDir(), "gen.go")
		if err := run(out, "main", "v"); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(out)
		if !strings.Contains(string(data), `AppVersion   = "v1.0.0"`) {
			t.Error("output missing AppVersion")
		}
	})

	for _, tag := range []string{"", "1.0.0"} {
		name := "no_tag"
		if tag != "" {
			name = "no_v_tag"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			initGitRepo(t, dir, tag)
			withChdir(t, dir)

			out := filepath.Join(t.TempDir(), "gen.go")
			if err := run(out, "main", "v"); err != nil {
				t.Fatal(err)
			}
			data, _ := os.ReadFile(out)
			if !strings.Contains(string(data), `AppVersion   = ""`) {
				t.Error("output AppVersion is not empty")
			}
		})
	}

	t.Run("custom_tag_prefix", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir, "release-1.0.0")
		withChdir(t, dir)

		out := filepath.Join(t.TempDir(), "gen.go")
		if err := run(out, "main", "release-"); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(out)
		if !strings.Contains(string(data), `AppVersion   = "release-1.0.0"`) {
			t.Error("output missing custom-prefix AppVersion")
		}
	})

	t.Run("git_error", func(t *testing.T) {
		dir := t.TempDir() // no git repo
		withChdir(t, dir)
		if err := run(filepath.Join(dir, "out.go"), "main", "v"); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("invalid_package_name", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir, "v1.0.0")
		withChdir(t, dir)

		err := run(filepath.Join(dir, "out.go"), "123bad", "v")
		if err == nil || !strings.Contains(err.Error(), "not an identifier") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("write_error", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir, "v1.0.0")
		withChdir(t, dir)

		out := filepath.Join(dir, "missing", "out.go")
		if err := run(out, "main", "v"); !os.IsNotExist(err) {
			t.Fatalf("expected a missing output directory error, got %v", err)
		}
	})
}

func TestMainFunc(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepo(t, dir, "v1.0.0")
		withChdir(t, dir)

		*output = filepath.Join(dir, "gen.go")
		*tagPrefix = "v"
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
		*tagPrefix = "v"
		main()
		if !exited {
			t.Error("os.Exit was not called")
		}
	})
}

// -- helpers --

func initGitRepo(t *testing.T, dir, tag string) {
	t.Helper()
	commands := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "test"},
		{"commit", "--allow-empty", "-m", "init"},
	}
	if tag != "" {
		commands = append(commands, []string{"tag", tag})
	}
	for _, args := range commands {
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
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestCommandLineFlags(t *testing.T) {
	// Calling main from a test would inherit testing's already-parsed flags.
	// Execute the actual command to exercise flag parsing as users invoke it.
	binary := filepath.Join(t.TempDir(), "version.exe")
	output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build version command: %v\n%s", err, output)
	}

	dir := t.TempDir()
	initGitRepo(t, dir, "v1.0.0")
	cmd := exec.Command("git", "tag", "release-2.0.0")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create release tag: %v\n%s", err, output)
	}

	cmd = exec.Command(binary, "-output=chosen.go", "-package=version", "-tag-prefix=release-")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run version command: %v\n%s", err, output)
	}

	data, err := os.ReadFile(filepath.Join(dir, "chosen.go"))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"package version", `AppVersion   = "release-2.0.0"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("generated source does not contain %q: %s", want, data)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "main_version.go")); !os.IsNotExist(err) {
		t.Fatalf("default output must not be written when -output is set: %v", err)
	}
}
