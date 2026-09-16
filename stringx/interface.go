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

package stringx

import "reflect"

// checkNonNil rejects both nil interfaces and interfaces containing typed nils.
func checkNonNil(value any, message string) {
	if value != nil {
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
			reflect.Pointer, reflect.Slice:
			if !v.IsNil() {
				return
			}

		default:
			return
		}
	}
	panic(message)
}
