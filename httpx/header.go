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

package httpx

import (
	"net/http"
	"strings"
)

// Headers
const (
	HeaderUpgrade       = "Upgrade"       // RFC 7230, 6.7
	HeaderUserAgent     = "User-Agent"    // RFC 7231, 5.5.3
	HeaderContentType   = "Content-Type"  // RFC 7231, 3.1.1.5
	HeaderAuthorization = "Authorization" // RFC 7235, 4.2
)

// MIME types
const (
	MIMEApplicationXML         = "application/xml"
	MIMEApplicationJSON        = "application/json"
	MIMEApplicationForm        = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf    = "application/protobuf"
	MIMEApplicationOctetStream = "application/octet-stream"
	MIMEMultipartForm          = "multipart/form-data"
)

// ContentType returns the MIME media type portion of the header "Content-Type".
func ContentType(header http.Header) (mime string) {
	mime = header.Get("Content-Type")
	if index := strings.IndexByte(mime, ';'); index > -1 {
		mime = strings.TrimSpace(mime[:index])
	}
	return
}
