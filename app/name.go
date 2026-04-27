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
	"os"
	"path/filepath"
	"strings"
)

var appname string

func init() {
	if len(os.Args) > 0 {
		appname = filepath.Base(os.Args[0])
		appname = strings.TrimSuffix(appname, ".exe")
	}
}

// GetName returns the name of the application.
//
// Default: the package name of the application, that's,
// filepath.Base(os.Args[0]), but not contain the suffix ".exe"
func GetName() string {
	return appname
}

// SetName sets the name of the application.
func SetName(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		panic("app name is empty")
	}
	appname = name
}
