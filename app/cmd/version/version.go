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

// Command version generates a Go source file that sets the app version and
// build time. It is intended to be invoked via //go:generate.
//
// It supports the following flags:
//
//	-output   The output Go source file name (default "main_version.go").
//	-package  The package name of the output file (default "main").
//
// Example go:generate directives:
//
//	//go:generate go run github.com/xgfone/go-toolkit/app/cmd/version
//	//go:generate go run github.com/xgfone/go-toolkit/app/cmd/version -output=version.go -package=version
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/token"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	output  = flag.String("output", "main_version.go", "The output file of version.")
	pkgname = flag.String("package", "main", "The package name.")

	// Overridable for testing.
	osexit = os.Exit
)

func main() {
	if err := run(*output, *pkgname); err != nil {
		fmt.Println(err)
		osexit(1)
	}
}

func run(outfile, pkg string) error {
	version, err := getVersion()
	if err != nil {
		return err
	}

	data, err := Generate(pkg, version, getBuildTime())
	if err != nil {
		return err
	}

	return os.WriteFile(outfile, []byte(data), 0600)
}

func getBuildTime() int64 {
	return time.Now().Unix()
}

func getVersion() (version string, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(8)
	cmd := exec.Command("git", "describe", "--tags", "--match", "v*")
	cmd.Stderr = io.Discard
	cmd.Stdout = buf
	err = cmd.Run()
	version = strings.TrimSpace(buf.String())
	return
}

// Generate returns the generated Go source code content
// with the given package name, version and build time.
func Generate(pkgname, version string, buildat int64) (string, error) {
	if !token.IsIdentifier(pkgname) {
		return "", fmt.Errorf("package name %q is not an identifier", pkgname)
	}
	if strings.Contains(version, `"`) {
		return "", fmt.Errorf(`version contains '"'`)
	}
	return fmt.Sprintf(_VersionFileTmpl, pkgname, version, buildat), nil
}

const _VersionFileTmpl = `package %s

import (
	"time"

	"github.com/xgfone/go-toolkit/app"
)

const (
	AppVersion   = "%s"
	AppBuildTime = %d
)

func init() {
	app.DefaultApp.SetBuildTime(time.Unix(AppBuildTime, 0).Local())
	app.DefaultApp.SetVersion(AppVersion)
}
`
