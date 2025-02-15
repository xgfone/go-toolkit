// Copyright 2024 xgfone
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

import (
	"runtime"
	"strconv"
	"strings"
)

var trimPrefixes = []string{"/pkg/mod/", "/src/"}

// TrimPkgFile trims the package path prefix of the file.
func TrimPkgFile(file string) string {
	for _, mark := range trimPrefixes {
		if index := strings.Index(file, mark); index > -1 {
			file = file[index+len(mark):]
			break
		}
	}
	return file
}

// Frame represents a call stack frame.
type Frame struct {
	File string `json:",omitempty"`
	Func string `json:",omitempty"`
	Line int    `json:",omitempty"`
}

// String formats the frame to a string.
func (f Frame) String() string {
	var b strings.Builder
	b.Grow(len(f.File) + len(f.Func) + 8)
	b.WriteString(f.File)

	if f.Func != "" {
		b.WriteByte(':')
		b.WriteString(f.Func)
	}

	if f.Line > 0 {
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(int64(f.Line), 10))
	}

	return b.String()
}

// Caller returns the stack frame of caller.
func Caller(skip int) Frame {
	pcs := make([]uintptr, 1)
	if n := runtime.Callers(skip+2, pcs); n > 0 {
		frame, _ := runtime.CallersFrames(pcs).Next()
		if frame.PC != 0 {
			return Frame{
				File: TrimPkgFile(frame.File),
				Func: extractfuncname(frame.Function),
				Line: frame.Line,
			}
		}
	}
	return Frame{File: "???"}
}

// Stacks returns the frames of the current call stacks.
func Stacks(skip int) []Frame {
	var pcs [64]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	if n == 0 {
		return nil
	}

	stacks := make([]Frame, 0, n)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		stacks = append(stacks, Frame{
			File: TrimPkgFile(frame.File),
			Func: extractfuncname(frame.Function),
			Line: frame.Line,
		})
	}

	return stacks
}

func extractfuncname(function string) string {
	if index := strings.LastIndexByte(function, '.'); index > -1 {
		function = function[index+1:]
	}
	return function
}
