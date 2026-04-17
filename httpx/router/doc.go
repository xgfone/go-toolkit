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

// Package router provides an HTTP server router with the middleware support.
//
// Example
//
//	package main
//
//	import (
//		"log/slog"
//		"net/http"
//
//		"github.com/xgfone/go-toolkit/httpx"
//		"github.com/xgfone/go-toolkit/httpx/router"
//	)
//
//	func main() {
//		// 1. New a router.
//		router := router.New()
//
//		// 2. Add the global middlewares.
//		router.Use(_logreq, _recover)
//
//		// 3. Add the routes.
//		router1 := router.Group("/group1")
//		router1.Path("/ok").Get(httpx.Handler200)
//
//		router2 := router.Group("/group2")
//		router3 := router2.Group("/group3").Auth(_auth)
//		router3.Path("/create").Post(httpx.Handler201)
//
//		// 4. Start the http server with the router.
//		_ = http.ListenAndServe("127.0.0.1:8000", router)
//	}
//
//	var (
//		_auth = httpx.MiddlewareFunc(func(next http.Handler) http.Handler {
//			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				// TODO:
//				next.ServeHTTP(w, r)
//			})
//		})
//
//		_recover = httpx.MiddlewareFunc(func(next http.Handler) http.Handler {
//			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				defer func() {
//					if r := recover(); r != nil {
//						slog.Error("wrap a panic", "err", r)
//						http.Error(w, "panic", 500)
//					}
//				}()
//				next.ServeHTTP(w, r)
//			})
//		})
//
//		_logreq = httpx.MiddlewareFunc(func(next http.Handler) http.Handler {
//			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//				next.ServeHTTP(w, r)
//				slog.Info("receive a request", "method", r.Method, "path", r.URL.Path)
//			})
//		})
//	)
package router
