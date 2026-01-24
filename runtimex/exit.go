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

package runtimex

import "os"

var _exit func(code int) = os.Exit

// Exit terminates the program with the given exit code.
// By default, Exit is equivalent to os.Exit(code).
//
// The exit behavior can be customized via SetExitFunc
// to perform some cleanup operations before exit.
func Exit(code int) {
	_exit(code)
}

// GetExitFunc returns the current exit function.
func GetExitFunc() func(code int) {
	return _exit
}

// SetExitFunc sets the exit function to be called by Exit.
//
// The provided function exit should typically call os.Exit(code)
// to ensure proper program termination. However, in some scenarios,
// such as testing, exit may choose not to call os.Exit to avoid
// terminating the test process.
//
// The function exit must not be nil, otherwise panic.
func SetExitFunc(exit func(code int)) {
	if exit == nil {
		panic("runtimex.SetExitFunc: exit function must not be nil")
	}
	_exit = exit
}
