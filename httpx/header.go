// Copyright 2024~2025 xgfone
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
	"slices"
	"strconv"
	"strings"
)

// Content-Disposition type values
const (
	Inline     = "inline"
	Attachment = "attachment"
)

// Headers
const (
	HeaderAccept              = "Accept"              // RFC 7231, 5.3.2
	HeaderAcceptCharset       = "Accept-Charset"      // RFC 7231, 5.3.3
	HeaderAcceptEncoding      = "Accept-Encoding"     // RFC 7231, 5.3.4
	HeaderAcceptLanguage      = "Accept-Language"     // RFC 7231, 5.3.5
	HeaderAcceptPatch         = "Accept-Patch"        // RFC 5789, 3.1
	HeaderAcceptRanges        = "Accept-Ranges"       // RFC 7233, 2.3
	HeaderAge                 = "Age"                 // RFC 7234, 5.1
	HeaderAllow               = "Allow"               // RFC 7231, 7.4.1
	HeaderAuthorization       = "Authorization"       // RFC 7235, 4.2
	HeaderCacheControl        = "Cache-Control"       // RFC 7234, 5.2
	HeaderConnection          = "Connection"          // RFC 7230, 6.1
	HeaderContentDisposition  = "Content-Disposition" // RFC 6266
	HeaderContentEncoding     = "Content-Encoding"    // RFC 7231, 3.1.2.2
	HeaderContentLanguage     = "Content-Language"    // RFC 7231, 3.1.3.2
	HeaderContentLength       = "Content-Length"      // RFC 7230, 3.3.2
	HeaderContentLocation     = "Content-Location"    // RFC 7231, 3.1.4.2
	HeaderContentRange        = "Content-Range"       // RFC 7233, 4.2
	HeaderContentType         = "Content-Type"        // RFC 7231, 3.1.1.5
	HeaderCookie              = "Cookie"              // RFC 2109, 4.3.4
	HeaderDate                = "Date"                // RFC 7231, 7.1.1.2
	HeaderETag                = "ETag"                // RFC 7232, 2.3
	HeaderExpect              = "Expect"              // RFC 7231, 5.1.1
	HeaderExpires             = "Expires"             // RFC 7234, 5.3
	HeaderFrom                = "From"                // RFC 7231, 5.5.1
	HeaderHost                = "Host"                // RFC 7230, 5.4
	HeaderIfMatch             = "If-Match"            // RFC 7232, 3.1
	HeaderIfModifiedSince     = "If-Modified-Since"   // RFC 7232, 3.3
	HeaderIfNoneMatch         = "If-None-Match"       // RFC 7232, 3.2
	HeaderIfRange             = "If-Range"            // RFC 7233, 3.2
	HeaderIfUnmodifiedSince   = "If-Unmodified-Since" // RFC 7232, 3.4
	HeaderLastModified        = "Last-Modified"       // RFC 7232, 2.2
	HeaderLink                = "Link"                // RFC 5988
	HeaderLocation            = "Location"            // RFC 7231, 7.1.2
	HeaderMaxForwards         = "Max-Forwards"        // RFC 7231, 5.1.2
	HeaderOrigin              = "Origin"              // RFC 6454
	HeaderPragma              = "Pragma"              // RFC 7234, 5.4
	HeaderProxyAuthenticate   = "Proxy-Authenticate"  // RFC 7235, 4.3
	HeaderProxyAuthorization  = "Proxy-Authorization" // RFC 7235, 4.4
	HeaderRange               = "Range"               // RFC 7233, 3.1
	HeaderReferer             = "Referer"             // RFC 7231, 5.5.2
	HeaderRetryAfter          = "Retry-After"         // RFC 7231, 7.1.3
	HeaderServer              = "Server"              // RFC 7231, 7.4.2
	HeaderSetCookie           = "Set-Cookie"          // RFC 2109, 4.2.2
	HeaderSetCookie2          = "Set-Cookie2"         // RFC 2965
	HeaderTE                  = "TE"                  // RFC 7230, 4.3
	HeaderTrailer             = "Trailer"             // RFC 7230, 4.4
	HeaderTransferEncoding    = "Transfer-Encoding"   // RFC 7230, 3.3.1
	HeaderUpgrade             = "Upgrade"             // RFC 7230, 6.7
	HeaderUserAgent           = "User-Agent"          // RFC 7231, 5.5.3
	HeaderVary                = "Vary"                // RFC 7231, 7.1.4
	HeaderVia                 = "Via"                 // RFC 7230, 5.7.1
	HeaderWarning             = "Warning"             // RFC 7234, 5.5
	HeaderWWWAuthenticate     = "WWW-Authenticate"    // RFC 7235, 4.1
	HeaderForwarded           = "Forwarded"           // RFC 7239
	HeaderXForwardedBy        = "X-Forwarded-By"      // RFC 7239, 5.1
	HeaderXForwardedFor       = "X-Forwarded-For"     // RFC 7239, 5.2
	HeaderXForwardedHost      = "X-Forwarded-Host"    // RFC 7239, 5.3
	HeaderXForwardedPort      = "X-Forwarded-Port"
	HeaderXForwardedProto     = "X-Forwarded-Proto"
	HeaderXForwardedProtocol  = "X-Forwarded-Protocol"
	HeaderXForwardedSSL       = "X-Forwarded-Ssl"
	HeaderXUrlScheme          = "X-Url-Scheme"
	HeaderXHTTPMethodOverride = "X-HTTP-Method-Override"
	HeaderXRealIP             = "X-Real-Ip"
	HeaderXServerID           = "X-Server-Id"
	HeaderXRequestID          = "X-Request-Id"
	HeaderXRequestedWith      = "X-Requested-With"

	// Access control
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials" // https://www.w3.org/TR/cors/#http-access-control-allow-credentials
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"     // https://www.w3.org/TR/cors/#http-access-control-allow-headers
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"     // https://www.w3.org/TR/cors/#http-access-control-allow-methods
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"      // https://www.w3.org/TR/cors/#http-access-control-allow-origin
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"    // https://www.w3.org/TR/cors/#http-access-control-expose-headers
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"           // https://www.w3.org/TR/cors/#http-access-control-max-age
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"   // https://www.w3.org/TR/cors/#http-access-control-request-headers
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"    // https://www.w3.org/TR/cors/#http-access-control-request-method

	// Security
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-Xss-Protection"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderXCSRFToken              = "X-Csrf-Token"
)

// MIME types
const (
	MIMETextXML                = "text/xml"
	MIMETextHTML               = "text/html"
	MIMETextPlain              = "text/plain"
	MIMEApplicationXML         = "application/xml"
	MIMEApplicationJSON        = "application/json"
	MIMEApplicationProtobuf    = "application/protobuf"
	MIMEApplicationMsgpack     = "application/msgpack"
	MIMEApplicationOctetStream = "application/octet-stream"
	MIMEApplicationForm        = "application/x-www-form-urlencoded"
	MIMEMultipartForm          = "multipart/form-data"

	MIMETextXMLCharsetUTF8         = MIMETextXML + "; charset=UTF-8"
	MIMETextHTMLCharsetUTF8        = MIMETextHTML + "; charset=UTF-8"
	MIMETextPlainCharsetUTF8       = MIMETextPlain + "; charset=UTF-8"
	MIMEApplicationXMLCharsetUTF8  = MIMEApplicationXML + "; charset=UTF-8"
	MIMEApplicationJSONCharsetUTF8 = MIMEApplicationJSON + "; charset=UTF-8"
)

// IsWebSocket reports whether the request is websocket.
func IsWebSocket(req *http.Request) bool {
	return req.Method == http.MethodGet &&
		strings.ToLower(req.Header.Get(HeaderConnection)) == "upgrade" &&
		strings.ToLower(req.Header.Get(HeaderUpgrade)) == "websocket"
}

// ContentType returns the MIME media type portion of the header "Content-Type".
func ContentType(header http.Header) (mime string) {
	mime = header.Get(HeaderContentType)
	if index := strings.IndexByte(mime, ';'); index > -1 {
		mime = strings.TrimSpace(mime[:index])
	}
	return
}

// Charset returns the charset of the request content.
//
// Return "" if there is no charset.
func Charset(header http.Header) string {
	ct := header.Get(HeaderContentType)
	for loop := len(ct) > 0; loop; {
		index := strings.IndexByte(ct, ';')
		if loop = index > -1; loop {
			ct = ct[index+1:]
		}

		if index = strings.IndexByte(ct, '='); index > -1 {
			if strings.ToLower(strings.TrimSpace(ct[:index])) == "charset" {
				return strings.TrimSpace(ct[index+1:])
			}
		}
	}
	return ""
}

// Accept returns the accepted Content-Type list from the request header
// "Accept", which are sorted by the q-factor weight from high to low.
//
// If there is no the request header "Accept", return nil.
//
// Notice:
//  1. If the value is "*/*", it will be amended as "".
//  2. If the value is "<MIME_type>/*", it will be amended as "<MIME_type>/".
//     So it can be used to match the prefix.
func Accept(header http.Header) []string {
	return accept(header.Get(HeaderAccept))
}

// AcceptEncoding is the same as Accept, but using the "Accept-Encoding" header.
func AcceptEncoding(header http.Header) []string {
	return accept(header.Get(HeaderAcceptEncoding))
}

// AcceptLanguage is the same as Accept, but using the "Accept-Language" header.
func AcceptLanguage(header http.Header) []string {
	return accept(header.Get(HeaderAcceptLanguage))
}

func accept(accept string) []string {
	if accept == "" {
		return nil
	}

	type acceptT struct {
		ct string
		q  float64
	}

	ss := strings.Split(accept, ",")
	accepts := make([]acceptT, 0, len(ss))
	for _, s := range ss {
		q := 1.0
		if k := strings.IndexByte(s, ';'); k > -1 {
			qs := s[k+1:]
			s = s[:k]

			if j := strings.IndexByte(qs, '='); j > -1 {
				if qs = qs[j+1:]; qs == "" {
					continue
				}
				if v, _ := strconv.ParseFloat(qs, 32); v > 1.0 || v <= 0.0 {
					continue
				} else {
					q = v
				}
			} else {
				continue
			}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		} else if s == "*" || s == "*/*" {
			s = ""
		} else if strings.HasSuffix(s, "/*") {
			s = s[:len(s)-1]
		}
		accepts = append(accepts, acceptT{ct: s, q: -q})
	}

	slices.SortStableFunc(accepts, func(a, b acceptT) int {
		switch {
		case a.q < b.q:
			return -1
		case a.q > b.q:
			return 1
		default:
			return 0
		}
	})

	results := make([]string, len(accepts))
	for i := range accepts {
		results[i] = accepts[i].ct
	}
	return results
}
