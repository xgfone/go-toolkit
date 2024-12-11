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
	"bytes"
	"context"
	"crypto/md5"
	"io"
	"testing"
)

func TestUploader(t *testing.T) {
	uploader := UploaderFunc(func(_ context.Context, r io.Reader) (url string, err error) {
		data, err := io.ReadAll(r)
		if err != nil {
			return
		}
		sum := md5.Sum(data)
		return string(sum[:]), nil
	})

	data := []byte("test")
	sums := md5.Sum(data)
	sum := string(sums[:])

	url, err := uploader.Upload(context.Background(), bytes.NewReader(data))
	if err != nil {
		t.Error(err)
	} else if url != sum {
		t.Errorf("expect url sum '%s', but got '%s'", sum, url)
	}
}
