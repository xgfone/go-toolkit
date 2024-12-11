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

package iox

import (
	"context"
	"io"
)

// Uploader is used to upload the data and returns the access url.
type Uploader interface {
	Upload(context.Context, io.Reader) (url string, err error)
}

// UploaderFunc is a data uploader function.
type UploaderFunc func(context.Context, io.Reader) (url string, err error)

// Upload implements the interface Uploader.
func (f UploaderFunc) Upload(c context.Context, r io.Reader) (url string, err error) {
	return f(c, r)
}
