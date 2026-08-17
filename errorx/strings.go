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

package errorx

import "strings"

// StringsError is an error that contains a list of strings.
type StringsError interface {
	Strings() []string
	error
}

type stringsError struct {
	strings []string
}

func (e *stringsError) Error() string {
	return strings.Join(e.strings, "; ")
}

func (e *stringsError) Strings() []string {
	return e.strings
}

// Strings returns an error containing vs.
func Strings(vs []string) error {
	return &stringsError{strings: vs}
}
