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
	"fmt"
	"reflect"

	"github.com/xgfone/go-toolkit/internal/structs"
)

var (
	errDefaultNilPointer = errors.New("SetDefault: v is nil")
	errDefaultNotStruct  = errors.New("SetDefault: v is not a pointer to struct")
)

// SetDefault sets the default values of the struct fields tagged with "default".
//
// If a field has a "default" tag and its current value is the zero value
// of its type, SetDefault will set it to the value parsed from the tag.
// Otherwise, the field is left unchanged.
func SetDefault[Struct any](v *Struct) (err error) {
	if v == nil {
		return errDefaultNilPointer
	}

	rtype := reflect.TypeFor[Struct]()
	if rtype.Kind() != reflect.Struct {
		return errDefaultNotStruct
	}

	root := reflect.ValueOf(v).Elem()
	for _, f := range structs.Parse(rtype, "").Fields {
		if f.Default == "" {
			continue
		}

		rtype, rvalue, err := f.GetField(root)
		if err == nil && rvalue.IsZero() {
			err = f.SetField(rtype, rvalue, f.Default)
		}
		if err != nil {
			return fmt.Errorf("%q: %w", f.Name, err)
		}
	}

	return
}
