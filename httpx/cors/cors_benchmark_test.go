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

package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	benchmarkCORSStatus             int
	benchmarkCORSHeader             int
	benchmarkCORSVaryAcceptEncoding = []string{"Accept-Encoding"}
)

func BenchmarkCORSServeHTTP(b *testing.B) {
	defaultConfig := NewDefaultConfig()

	benchmarks := []struct {
		name        string
		config      Config
		request     *http.Request
		setupHeader func(http.Header)
	}{
		{
			name:    "actual_static_wildcard_no_origin",
			config:  Config{AllowOrigins: []string{"*"}},
			request: benchmarkCORSRequest(http.MethodGet, nil),
		},
		{
			name:   "actual_static_wildcard_with_origin",
			config: Config{AllowOrigins: []string{"*"}},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://example.com",
			}),
		},
		{
			name: "actual_static_wildcard_with_expose_headers",
			config: Config{
				AllowOrigins:  []string{"*"},
				ExposeHeaders: []string{"X-Request-Id", "X-Trace-Id"},
			},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://example.com",
			}),
		},
		{
			name:   "actual_exact_origin",
			config: Config{AllowOrigins: []string{"https://example.com"}},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://example.com",
			}),
		},
		{
			name: "actual_multiple_exact_origins",
			config: Config{AllowOrigins: []string{
				"https://one.example",
				"https://two.example",
				"https://three.example",
				"https://four.example",
				"https://five.example",
				"https://six.example",
				"https://seven.example",
				"https://eight.example",
			}},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://seven.example",
			}),
		},
		{
			name: "actual_credentialed_wildcard_reflect_origin",
			config: Config{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
			},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://example.com",
			}),
		},
		{
			name:   "actual_subdomain_pattern",
			config: Config{AllowOrigins: []string{"https://*.example.com"}},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://api.example.com",
			}),
		},
		{
			name:   "actual_subdomain_pattern_with_port",
			config: Config{AllowOrigins: []string{"https://*.example.com:8443"}},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://api.example.com:8443",
			}),
		},
		{
			name: "actual_disallowed_origin_pass_through",
			config: Config{AllowOrigins: []string{
				"https://allowed.example",
				"https://other.example",
			}},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://denied.example",
			}),
		},
		{
			name:    "actual_no_origin_dynamic_config",
			config:  Config{AllowOrigins: []string{"https://example.com"}},
			request: benchmarkCORSRequest(http.MethodGet, nil),
		},
		{
			name:   "actual_vary_merge_existing",
			config: Config{AllowOrigins: []string{"https://example.com"}},
			request: benchmarkCORSRequest(http.MethodGet, map[string]string{
				"Origin": "https://example.com",
			}),
			setupHeader: func(h http.Header) {
				h["Vary"] = benchmarkCORSVaryAcceptEncoding
			},
		},
		{
			name:   "options_non_preflight",
			config: Config{AllowOrigins: []string{"https://example.com"}},
			request: benchmarkCORSRequest(http.MethodOptions, map[string]string{
				"Origin": "https://example.com",
			}),
		},
		{
			name:   "preflight_static_wildcard_default",
			config: defaultConfig,
			request: benchmarkCORSRequest(http.MethodOptions, map[string]string{
				"Origin":                        "https://example.com",
				"Access-Control-Request-Method": http.MethodPut,
			}),
		},
		{
			name: "preflight_explicit_headers",
			config: Config{
				AllowOrigins: []string{"https://example.com"},
				AllowMethods: []string{http.MethodPut},
				AllowHeaders: []string{"X-Token", "X-Trace"},
			},
			request: benchmarkCORSRequest(http.MethodOptions, map[string]string{
				"Origin":                         "https://example.com",
				"Access-Control-Request-Method":  http.MethodPut,
				"Access-Control-Request-Headers": "X-Token, X-Trace",
			}),
		},
		{
			name: "preflight_default_reflect_headers",
			config: Config{
				AllowOrigins: []string{"https://example.com"},
				AllowMethods: []string{http.MethodPut},
			},
			request: benchmarkCORSRequest(http.MethodOptions, map[string]string{
				"Origin":                         "https://example.com",
				"Access-Control-Request-Method":  http.MethodPut,
				"Access-Control-Request-Headers": "X-One, X-Two, X-Three, X-Four",
			}),
		},
		{
			name: "preflight_credentialed_wildcard_reflect",
			config: Config{
				AllowOrigins:     []string{"*"},
				AllowCredentials: true,
				AllowMethods:     []string{"*"},
				AllowHeaders:     []string{"*"},
			},
			request: benchmarkCORSRequest(http.MethodOptions, map[string]string{
				"Origin":                         "https://example.com",
				"Access-Control-Request-Method":  http.MethodPatch,
				"Access-Control-Request-Headers": "Authorization, X-Token",
			}),
		},
		{
			name: "preflight_disallowed_origin",
			config: Config{
				AllowOrigins: []string{"https://allowed.example"},
				AllowMethods: []string{http.MethodPut},
			},
			request: benchmarkCORSRequest(http.MethodOptions, map[string]string{
				"Origin":                        "https://denied.example",
				"Access-Control-Request-Method": http.MethodPut,
			}),
		},
		{
			name: "preflight_disallowed_header",
			config: Config{
				AllowOrigins: []string{"https://example.com"},
				AllowMethods: []string{http.MethodPut},
				AllowHeaders: []string{"X-Allowed"},
			},
			request: benchmarkCORSRequest(http.MethodOptions, map[string]string{
				"Origin":                         "https://example.com",
				"Access-Control-Request-Method":  http.MethodPut,
				"Access-Control-Request-Headers": "X-Denied",
			}),
		},
		{
			name: "preflight_invalid_method_token",
			config: Config{
				AllowOrigins: []string{"https://example.com"},
				AllowMethods: []string{http.MethodPut},
			},
			request: benchmarkCORSRequest(http.MethodOptions, map[string]string{
				"Origin":                        "https://example.com",
				"Access-Control-Request-Method": "BAD METHOD",
			}),
		},
	}

	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			handler := bm.config.CORS(0).HTTPHandler(next)
			w := newBenchmarkCORSResponseWriter()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				w.reset()
				if bm.setupHeader != nil {
					bm.setupHeader(w.Header())
				}
				handler.ServeHTTP(w, bm.request)
			}

			benchmarkCORSStatus = w.status
			benchmarkCORSHeader = len(w.header)
		})
	}
}

func benchmarkCORSRequest(method string, headers map[string]string) *http.Request {
	r := httptest.NewRequest(method, "/", nil)
	for key, value := range headers {
		r.Header.Set(key, value)
	}
	return r
}

type benchmarkCORSResponseWriter struct {
	header http.Header
	status int
}

func newBenchmarkCORSResponseWriter() *benchmarkCORSResponseWriter {
	return &benchmarkCORSResponseWriter{header: make(http.Header, 8)}
}

func (w *benchmarkCORSResponseWriter) Header() http.Header {
	return w.header
}

func (w *benchmarkCORSResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func (w *benchmarkCORSResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *benchmarkCORSResponseWriter) reset() {
	clear(w.header)
	w.status = 0
}
