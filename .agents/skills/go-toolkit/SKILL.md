---
name: go-toolkit
description: Use github.com/xgfone/go-toolkit packages in Go applications for collection transforms, HTTP services and clients, application lifecycle, struct binding, masking, and utility tasks, or migrate v0.30.2 and v1 prereleases toward v1. Apply only to this module's usage and upgrades.
---

# Using and migrating go-toolkit

Work with `github.com/xgfone/go-toolkit`. Import the required subpackage,
for example `github.com/xgfone/go-toolkit/iterx`; the module root is not a
Go API package. Current v1 development requires Go 1.27 and has no third-party
module dependencies. Check the application's effective revision before using
the current API guidance below.

## Resolve the version and source

Run in the consuming application's module directory:

```shell
go list -m -json github.com/xgfone/go-toolkit
```

Inspect `Version`, `Replace`, and `Dir` together with `go.mod` and any active
`go.work`. `Dir` points to the effective source, including local replacements.
Workspace or replacement source may have no release version; inspect its actual
revision. For vendored builds, also inspect `vendor/modules.txt` and the vendored
package. If the module is not installed, use the requested version rather than
silently selecting the latest one.

For migration, identify the exact source and target tags or commits. To inspect
a target not installed in the application, use a checkout of that revision or
`go mod download -json github.com/xgfone/go-toolkit@VERSION` and read its `Dir`.
This inspection is separate from upgrading the application's dependency. Do not
assume a final `v1.0.0` release exists or that a beta includes changes from the
development branch. If a requested revision is unavailable, report that fact.

All module paths and documentation paths in the following table are relative
to the selected **go-toolkit source/documentation root**, not the consuming
project or installed skill. Read that version's package comments, signatures,
and examples; use `go doc` for the relevant package or symbol. If local documents
are missing, retrieve them from the matching repository revision. Read only
the packages needed for the task.

## Select the package

| Task                                                                            | Packages and documentation                                                                                                                           |
| ------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| Lazy iterator transforms and aggregation                                        | `iterx/doc.go`, `iterx/transform.go`, `iterx/func.go`, `iterx/example_test.go`                                                                       |
| Slice conversion, grouping, deduplication, or map conversion                    | `slicex/slice.go`, `mapx/map.go`, `mapx/smap.go`; use standard `slices`/`maps` where documented                                                      |
| String masking, email masking, or string generation                             | `stringx/README.md`, `stringx/doc.go`                                                                                                                |
| HTTP clients, request binding, context handlers, headers, or JSON/XML responses | `httpx/client_request.go`, `httpx/client_request_example_test.go`, `httpx/binder.go`, `httpx/handler.go`, `httpx/context.go`, `httpx/header.go`      |
| Routes, middleware, CORS, or HTTP logging                                       | `httpx/router/doc.go`, `httpx/router/router.go`, `httpx/middleware/`, `httpx/cors/cors.go`, `httpx/logger/logger.go`, `httpx/logger/example_test.go` |
| Application lifecycle, modules, background tasks, or build version command      | `app/app.go`, `app/app_hook.go`, `app/app_goroutine.go`, `app/app_module.go`, `app/module/httpserver.go`, `app/cmd/version/version.go`               |
| Struct binding, defaults, or validation                                         | `structx/binder.go`, `structx/default.go`, `structx/validate.go`, `validation/validate.go`                                                           |
| Error codes, wrapped errors, or result responses                                | `codeint/error.go`, `codeint/error_http.go`, `errorx/sensitive.go`, `errorx/strings.go`, `result/response.go`, `result/respond.go`                   |
| Clocks, cancellable waits, random values, or integer-list parsing               | `timex/time.go`, `timex/sleep.go`, `random/int.go`, `random/string.go`, `strconvx/slice.go`                                                          |
| Networking, comments, I/O, reflection, runtime, or zero-copy conversions        | `netx/`, `netx/netipx/`, `bytex/comment.go`, `iox/closer.go`, `reflectx/implement.go`, `runtimex/`, `unsafex/`                                       |
| Experimental streams or redaction                                               | `exp/README.md`, `exp/iterx/doc.go`, `exp/redact/doc.go`; these packages have no compatibility guarantees even in v1                                 |

## Apply the relevant usage contracts

- Iterator adapters are lazy, preserve input order, stop when the consumer stops,
  and do not cache values or add concurrency. Repeatability depends on the input.
  A nil iterator is not an empty sequence. Distinguish a transform's callback
  (`To`) from transform-and-filter (`FilterTo`), and one-value from two-value
  iterators. Use standard collectors when appropriate; inspect nilness, capacity,
  duplicate-key, and slice-sharing requirements before substituting a collector.
- `slicex.Unique` preserves first-occurrence order and nilness; `GroupBy`
  preserves order inside each group. Map iteration order is unspecified.
  `mapx.SMap[T]` is a map type with helpers, not a synchronized map.
- HTTP request binders accept destination struct pointers, bind the requested
  source, then apply defaults and validation. Use `path`, `query`, `header`, or
  `form` tags as appropriate. `BindPath` reads `Request.PathValue`; a router or
  caller must populate it. Missing and empty flat string values are both ignored,
  so required values need validation. `structx.BindValues` uses leaf names for
  expanded fields; `BindMap` supports path-aware nested binding.
- `httpx.ContextHandler` needs a context installed by middleware such as
  `middleware.Context`. That context is pooled and released after handling;
  do not retain it for asynchronous work. Configure router routes and middleware
  before serving: the backend is initialized once. The default backend routes
  unmatched paths and methods to its fallback, unless an explicit catch-all
  takes precedence.
- HTTP client helpers such as `Get`/`Post` take a context and a response
  destination pointer. Automatic handling requires status 200; a
  `func(*http.Response) error` callback handles other statuses or formats.
  `DoRequest` closes the response body after the callback. Non-reader request
  bodies are JSON encoded, and a non-nil request body sets JSON content type.
- Use `logger.Config.Middleware` and `cors.Config.Middleware`. Logger
  `GetLogAttrs` replaces the default attribute callback; wrap it to retain
  default fields when adding custom ones. The default callback includes original
  sensitive errors when present; choose attributes appropriate to the log's
  audience. Header/query/body logging requires explicit custom attributes.
- Configure an `app.App` before `Run`; it runs only once. Register lifecycle
  modules and hooks, and use `GoNamed` for managed background work.
  `SetSignals()` disables OS-signal handling. `module.HttpServer` belongs to a
  running application's lifecycle. `runtimex.ExitContext` is separate: `Exit`
  cancels it before the exit function but does not wait for cleanup.
- String masks count runes; generators count bytes including affixes.
  `WithXxx` mask/affix configuration returns independent values. Preset getters
  and replacement setters are synchronized, but custom implementations and shared
  output buffers need their own concurrency discipline.
- `timex.Now` defaults to `time.Now` without location conversion. Prefer
  `time.Now` unless a replaceable clock is needed. Change `SetNowFunc` before
  concurrent use and restore `time.Now` after tests. `timex.Sleep` observes
  cancellation, including already-cancelled contexts with a non-positive duration.
- `strconvx.ParseInt64Slice`/`ParseUint64Slice` return nil for empty input and
  fail without partial results for invalid elements. `random.IntN`/`Int64N`
  require a positive bound; string helpers take an explicit charset.
  `unsafex.String` and `unsafex.Bytes` share memory: do not mutate the backing
  bytes of the resulting string or the bytes obtained from a string.
- `exp/iterx` uses methods such as `Map` and `FilterMap`, whereas stable
  `iterx` uses `To` and `FilterTo`. Keep these import paths and APIs distinct;
  experimental packages share this module's version and have no stability promise.

For example, lazily parse and filter integers, then collect the first two:

```go
numbers := iterx.FilterTo(slices.Values([]string{"10", "bad", "20", "30"}), func(s string) (int, bool) {
    n, err := strconv.Atoi(s)
    return n, err == nil
})
out := slices.Collect(iterx.Take(numbers, 2))
// out: []int{10, 20}
```

Import standard `slices` and `strconv`, plus
`github.com/xgfone/go-toolkit/iterx`. For other packages, use that revision's
package comments and example tests instead of inventing analogies from this API.

## Migrate and verify

For v0.30.2-to-v1 or v1 prerelease upgrades, read
[references/migration-v0.30.2-to-v1.md](references/migration-v0.30.2-to-v1.md)
relative to this **skill directory**, then the target module's relevant migration
guides and source. The bundled index identifies a checked source revision; it
does not override the application's exact target.

After updating used call sites, run the consuming application's relevant tests.
Check collection nilness/order/sharing, JSON behavior, destination binding,
HTTP statuses and fallback routes, custom log fields, time locations, and
startup/cancellation/shutdown where used. Report the resulting dependency
revision, changes, checks run, and unverified behavior. If modifying this library,
use `go test ./...`; its CI also runs the suite with the race detector.
