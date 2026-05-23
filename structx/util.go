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

package structx

import (
	"errors"
	"reflect"
)

func anyStructPtr(dst any) (rtype reflect.Type, root reflect.Value, err error) {
	if dst == nil {
		err = errors.New("dst is nil")
		return
	}

	rtype = reflect.TypeOf(dst)
	if rtype.Kind() != reflect.Pointer {
		err = errors.New("dst is not a pointer to struct")
		return
	}

	rtype = rtype.Elem()
	if rtype.Kind() != reflect.Struct {
		err = errors.New("dst is not a pointer to struct")
		return
	}

	root = reflect.ValueOf(dst)
	if root.IsNil() {
		err = errors.New("dst is nil")
		return
	}

	root = root.Elem()
	return
}
