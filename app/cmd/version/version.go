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
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var output = flag.String("output", "main_version.go", "The output file of version.")
var pkgname = flag.String("package", "main", "The package name.")

func main() {
	buf := bytes.NewBuffer(nil)
	buf.Grow(8)

	cmd := exec.Command("git", "describe", "--tags", "--match", "v*")
	cmd.Stderr = io.Discard
	cmd.Stdout = buf

	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	version := strings.TrimSpace(buf.String())
	if strings.Contains(version, `"`) {
		fmt.Println(`version contain '"'`)
		os.Exit(1)
	}

	buildat := time.Now().Unix()
	data := fmt.Sprintf(_VersionFileTmpl, *pkgname, version, buildat)

	if err := os.WriteFile(*output, []byte(data), 0600); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
