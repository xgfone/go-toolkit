# Migration: v0.30.2 to v1

This guide compares tag `v0.30.2` with the v1 implementation in this source
tree. Read it together with the guides from the v1 tag or revision you select.
Upgrade the toolchain from Go 1.24 to Go 1.27. The module path remains
`github.com/xgfone/go-toolkit`; import its subpackages.

## Package guides

Review the shared JSON, HTTP, and utility changes below, then read the guides
for the packages your application uses. Use the guides from the selected v1
tag or revision throughout the migration.

| Packages                               | Migration guide                                                        |
| -------------------------------------- | ---------------------------------------------------------------------- |
| `app`, `app/module`, `app/cmd/version` | [Lifecycle, dependencies, tasks, and build metadata](app/MIGRATION.md) |
| `iterx`                                | [Iterator APIs](iterx/MIGRATION.md)                                    |
| `slicex`                               | [Slice APIs and storage behavior](slicex/MIGRATION.md)                 |
| `mapx`                                 | [Map APIs and collectors](mapx/MIGRATION.md)                           |
| `stringx`                              | [Masks, generators, and presets](stringx/README.md#migration)          |

The sections below cover changes outside those guides and JSON behavior shared
by several packages. Examples omit imports and application-specific values.

## Removed APIs and imports

| v0.30.2 API or import                                                  | v1 replacement                                                                                                                          |
| ---------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `jsonx` and its global codec setters                                   | `encoding/json/v2` or application-owned codecs; see the JSON section below.                                                             |
| `iox.Uploader`, `iox.UploaderFunc`                                     | Define the required upload interface and function adapter in the consuming application.                                                 |
| `github.com/xgfone/go-toolkit/redact`                                  | `github.com/xgfone/go-toolkit/exp/redact`; update imports wherever its `Level` and `Redactor` types are used.                           |
| `router.DefaultRouter`                                                 | `router.Default()`; use `router.New()` for an independently owned router. The default instance can no longer be reassigned.             |
| `logger.Config.Logger(priority)`                                       | The existing `logger.Config.Middleware(priority)` method.                                                                               |
| `cors.Config.CORS(priority)`                                           | The existing `cors.Config.Middleware(priority)` method; CORS configuration otherwise retains its behavior.                              |
| `logger.Config.GetRequestBody`, `GetResponse`                          | `GetLogAttrs(w, r) []slog.Attr`; migrate selected fields explicitly.                                                                    |
| Assignable `httpx.Handler200`, `Handler404`, and other status handlers | Constants implementing `http.Handler`; declare application-owned handler variables with an explicit interface type when replacing them. |
| `random.DefaultCharset` variable                                       | `random.DefaultCharset()` and `random.SetDefaultCharset(value)`.                                                                        |
| `bytex.CommentHash`, `CommentSlashes`                                  | Application-owned `[]byte("#")` and `[]byte("//")`.                                                                                     |
| `netx.IPIsOn`                                                          | `netx.IsLocalIP`; the local-interface lookup contract is retained.                                                                      |
| `runtimex.TrimPkgFile`                                                 | `runtimex.TrimSourcePath`; the trimming policy is retained.                                                                             |
| `codeint.Error.TryError(err)`                                          | The existing `codeint.Error.Wrap(err)` method.                                                                                          |
| Decoding methods on `codeint.Error` and `result.Response`              | Call the decoder directly; see the [JSON replacements](#json-encoding-and-decoding) below.                                              |

The status-handler change also affects local type inference. For example,
`handler := httpx.Handler404` now infers the concrete constant's type. Use an
interface when the variable must later hold another implementation:

```go
var handler http.Handler = httpx.Handler404
handler = http.NotFoundHandler()
```

The moved redaction APIs retain their signatures and behavior. All packages
under `exp/`, including the new chainable `exp/iterx`, have no compatibility
guarantee in v1; see the [experimental package policy](exp/README.md).

## JSON encoding and decoding

In v0.30.2, `jsonx` supplied configurable wrappers around `encoding/json`.
Its default writer used an `Encoder` with HTML escaping disabled and appended
a newline. Its default reader used `Decoder.Decode` once, without requiring EOF.
The `[]byte`/`string` marshal helpers removed the writer's trailing newline.

In v1, HTTP JSON rendering and request encoding, and JSON body binding use
`encoding/json/v2` directly. The decoding convenience methods on `codeint.Error`
and `result.Response` are removed; decode into these types with standard-library
functions or application-owned codecs. The global codec setters no longer
intercept toolkit JSON operations.

| Removed v0.30.2 call                          | v1 standard-library replacement                                                           |
| --------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `jsonx.Marshal`, `MarshalBytes`               | `json.Marshal(value)`                                                                     |
| `jsonx.MarshalWriter(writer, value)`          | `json.MarshalWrite(writer, value)`                                                        |
| `jsonx.MarshalString`, `MarshalStringWithCap` | Marshal to bytes and convert to string, or own the writer/buffer.                         |
| `jsonx.Unmarshal`, `UnmarshalBytes`           | `json.Unmarshal(data, dst)`                                                               |
| `jsonx.UnmarshalReader(dst, reader)`          | `json.UnmarshalRead(reader, dst)`; argument order changes.                                |
| `jsonx.UnmarshalString`                       | `json.Unmarshal([]byte(text), dst)` or `json.UnmarshalRead(strings.NewReader(text), dst)` |

The following seven methods from v0.30.2 are also removed:

| v0.30.2 method                                                     | v1 replacement                                                        |
| ------------------------------------------------------------------ | --------------------------------------------------------------------- |
| `codeint.Error.Decode`, `result.Response.Decode`                   | `decode(dst)`; pass the destination pointer directly to the callback. |
| `codeint.Error.DecodeJSON`, `result.Response.DecodeJSON`           | `json.UnmarshalRead(reader, dst)`                                     |
| `codeint.Error.DecodeJSONBytes`, `result.Response.DecodeJSONBytes` | `json.Unmarshal(data, dst)`                                           |
| `codeint.Error.DecodeJSONString`                                   | `json.UnmarshalRead(strings.NewReader(text), dst)`                    |

Here `json` denotes `encoding/json/v2`, and `dst` is a non-`nil` pointer to the
destination, such as `&codedErr` for a `codeint.Error` or `&response` for a
`result.Response`. The standard-library calls also accept `json.Options`.
For a non-`null` JSON error, initialize `result.Response.Error` with a concrete
error pointer before decoding, for example
`response := result.Response{Error: new(codeint.Error)}`; its `error` interface
does not select a concrete type automatically.

The JSON replacements change defaults:

| Case                                                     | v0.30.2 default                         | Current v1 default                                                                                               |
| -------------------------------------------------------- | --------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Writer output                                            | A trailing newline                      | No appended newline, including HTTP responses and JSON request bodies                                            |
| `nil` `[]int` / `map` without omission tags              | `null`                                  | `[]` / `{}`                                                                                                      |
| Struct-field name matching                               | Case-insensitive fallback               | Case-sensitive; an unmatched name may be ignored without an error                                                |
| Duplicate JSON member names                              | Accepted                                | Rejected                                                                                                         |
| Invalid UTF-8 in JSON strings                            | Replaced with the replacement rune      | Rejected                                                                                                         |
| Trailing non-whitespace or a second top-level JSON value | Default reader accepted the first value | `Unmarshal` and `UnmarshalRead` reject the input                                                                 |
| Reader completion                                        | Decode the next value                   | `UnmarshalRead` reads to EOF; use an explicit streaming decoder for a sequence of values                         |
| `omitempty`                                              | Go-value emptiness                      | JSON-value emptiness; zero numbers and `false` are retained, while empty values behind interfaces can be omitted |
| `[N]byte`                                                | JSON array of numbers                   | Base64 JSON string                                                                                               |
| Ordinary Go array decoding                               | Shorter/longer JSON arrays accepted     | Exact array length required                                                                                      |
| `time.Duration`                                          | Number of nanoseconds                   | No default representation; define an integer/string wire field or a custom codec                                 |
| Map member order                                         | Deterministic key ordering              | Unspecified by default                                                                                           |
| JSON `null` into an existing scalar                      | Could leave its old value intact        | Resets it to zero                                                                                                |
| `,string` on `bool`/`string` fields                      | Supported                               | Unsupported; the option applies to numeric types                                                                 |
| Nonempty struct containing only unexported fields        | Encoded as `{}`                         | Encoding fails, including ordinary errors such as `errors.New(...)`                                              |
| Pointer-receiver JSON methods on non-addressable values  | Not consistently called                 | Called consistently; review custom marshaling and merge behavior                                                 |

HTML characters `<`, `>`, and `&` were already unescaped by the old default
`jsonx` writer. Do not substitute the default behavior of plain
`encoding/json.Marshal` when determining the v0.30.2 wire format.

Case sensitivity applies to toolkit types too: `{"code":401}` no longer
populates `codeint.Error.Code`, whose JSON name is `Code`. An ignored member can
leave the destination zero while decoding returns `nil`. Match field names
exactly or decode through an application-owned DTO/codec.

The omission changes affect toolkit DTOs too. Ignoring the old writer newline:

- `codeint.Error{}` changes from `{}` to `{"Code":0}`.
- `result.Ok("")` changes from `{"Data":""}` to `{}`; empty collections in `Data`
  can also disappear under `omitempty`.
- `result.Err(errors.New("failure"))` can now fail JSON encoding instead of
  producing `{"Error":{}}`. Supply a serializable error DTO, such as an
  appropriate `codeint.Error`, and check rendering errors.

Some convenience responders, including `result.Respond` with a raw
`http.ResponseWriter`, do not return rendering errors. Validate their payloads or
use an application-owned responder when those errors must be handled.

When retaining a previous wire contract, use explicit DTO fields and tags
supported by the target codec, or own the encoder/decoder. For example, this
application-owned writer retains the old default writer's newline and HTML
policy using `encoding/json/v2`:

```go
func writeLegacyJSON(w http.ResponseWriter, status int, value any) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    data = append(data, '\n')
    w.Header().Set("Content-Type", "application/json; charset=UTF-8")
    w.WriteHeader(status)
    _, err = w.Write(data)
    return err
}
```

Installing such a writer in your application does not alter toolkit JSON body
binding or decoding automatically. Migrate those call sites separately. In
particular, changing the removed global `jsonx` hooks has no v1 equivalent.

## HTTP behavior

### Binding and clients

`httpx.BindPath` and `Context.BindPath` are additions. They bind `path` tags from
`Request.PathValue`, then apply defaults and validation. Populate wildcard values
through the router or `Request.SetPathValue`. Missing and empty values are ignored;
query parameters and bodies are not read. Other binders retain their signatures,
with JSON decoding subject to the changes above. The `Context` binder methods
previously guarded by Go 1.27 are available with the new minimum toolchain.

Automatic HTTP response decoding now passes the supplied destination directly
to the JSON decoder, instead of the address of the interface holding it. Pass a
non-`nil` pointer. This corrects the old behavior that could appear to accept an
invalid destination or fail to update its intended value. Status 200 is still
required for automatic handling; a response callback still owns status and body
interpretation, and the helper closes the response body after it returns.

JSON request bodies now own their backing bytes throughout asynchronous body
closure and `GetBody` replays. This fixes corruption possible with the old pooled
buffer, including redirects that replay the body. Default debug logs now contain
method, host, path, and status; full URLs, headers, and bodies are no longer logged.
Use `WrapClient` for application-specific logging.

### Header parsing and response status

- `IsWebSocket` now recognizes case-insensitive, comma-separated `Connection` and
  `Upgrade` tokens across all header values, such as `keep-alive, Upgrade`.
- `ContentType` normalizes the MIME media type's case and whitespace; `Charset`
  parses and unquotes MIME parameters instead of returning their raw suffix.
  For example, `Application/JSON; Charset="utf-8"; boundary=x` now yields
  `application/json` and `utf-8`. This also affects body content-type selection.
- `Accept`, `AcceptEncoding`, and `AcceptLanguage` now read all header values and
  honor quoted parameter values, including embedded commas. Parameters other
  than `q` no longer become mistaken weights. Invalid weights, including `NaN`,
  and weights outside `(0, 1]` are excluded; weights use 64-bit precision.
  Stable descending-weight order and existing wildcard normalization are retained.
- The `Context` response writer no longer commits final status on informational
  1xx responses other than 101. A later final response is forwarded correctly;
  invalid status values outside `[100, 999]` panic.
- `codeint.Error.WithStatus` and `StatusCode` now normalize invalid nonzero HTTP
  statuses outside `[100, 599]` to 500. Zero still clears an explicit status and
  allows a valid `Code` to supply it. This also makes `NewError(42).ServeHTTP` safe.

### Router and request logging

The default `ServeMux` backend installs the fallback unless a successfully
registered, host-independent and method-independent catch-all covers it. A
method-only or host-only root route no longer suppresses fallback for other
requests. Unmatched methods reach the fallback instead of `ServeMux`'s 405
response. Handler-generated 404 responses remain handler responses.

`Router.Routes` returns a copied slice rather than its internal slice. Mutating
that slice no longer changes registration; configure the router before serving.
Take a fresh snapshot to observe subsequent registration and `Online` updates.
Failed registration now clears a caller-supplied stale `Online` flag.

Logger configuration replaces `GetRequestBody` and `GetResponse` with `GetLogAttrs`.
The built-in request fields retain `raddr`, `method`, `host`, `path`, `cost`, and the
configured request ID. Query, request/response headers, and bodies are no longer
included automatically. `NewDefaultConfig`'s attribute callback provides `code`, `err`,
and `sensitive_error` when applicable. It runs after the handler and before `PostHandle`;
wrapping it preserves those fields when adding application attributes:

```go
config := logger.NewDefaultConfig()
baseAttrs := config.GetLogAttrs
config.GetLogAttrs = func(w http.ResponseWriter, r *http.Request) []slog.Attr {
    attrs := baseAttrs(w, r)
    return append(attrs, slog.String("query", r.URL.RawQuery))
}
```

An empty `Config` has no default response attributes. Original sensitive error
messages may still be included by `NewDefaultConfig`; select attributes for the
intended log audience when replacing the callback.

## Time and utility behavior

| Area               | v0.30.2 to v1 change and migration action                                                                                                                                                                                                                                              |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `timex`            | `Format`, `Formats`, `GetFormat`, `SetFormat`, `GetFormats`, `SetFormats`, `Location`, `GetLocation`, `SetLocation`, and `Unix` are removed. Keep policies in the application and use `time.Unix` plus an explicit location.                                                           |
| Default clock      | `timex.Now` defaults to `time.Now` instead of converting to `timex.Location` (UTC by default). `Today` and time-based string generators follow that clock's location. `App.BuildTime` now uses `time.Local`. Request UTC explicitly where required.                                    |
| Zero checks        | `IsZero` now treats typed `nil` values with an `IsZero` method as zero without invoking that method. The new `EqualZero` uses `==`; an interface containing a typed `nil` is not a `nil` interface.                                                                                    |
| `Stacks`           | `runtimex.Stacks` now includes the final frame previously dropped. Update stack snapshots or consumers that assumed its old length.                                                                                                                                                    |
| Network conversion | `netipx.AddrFromNetAddr` returns errors for `nil`/typed-`nil` TCP, UDP, and IP addresses and malformed IP bytes; old paths could panic or return an invalid address with `nil` error. IPv6 zones are retained, and raw IPv6 custom addresses can be parsed before host/port splitting. |
| Random integers    | `IntN`/`Int64N` retain the `[0, n)` contract and panic for non-positive bounds. Cryptographic sampling no longer logs an error and falls back to `math/rand`; failure of `crypto/rand.Read` terminates rather than producing weaker random output.                                     |
| Default charset    | The getter/setter are safe concurrently; the setter rejects an empty string. `String` and `Bytes` still require an explicit charset.                                                                                                                                                   |
| Text binding       | `structx` string sources now reach `UnmarshalText` through a writable byte copy, preventing source-string mutation or faults from the old unsafe alias. Existing byte-slice sources can still be passed through; copy data retained by a custom unmarshaler.                           |
| Comments           | `RemoveLineComments` now respects escaped double quotes and comment markers inside double-quoted text while removing a later real comment. Supply the removed marker variables as local byte slices.                                                                                   |

The public `structx` binder/default/validation signatures and core field-selection
rules are retained. The changes above do not turn flat `BindValues` into nested
path binding; `BindMap` remains the path-aware operation. `runtimex.ExitContext`
still only notifies on `runtimex.Exit` and does not wait for cleanup or observe
the independent app lifecycle.

## Additions available after migration

Alongside additions documented in the package guides, v1 adds:

- `strconvx.ParseInt64Slice` and `ParseUint64Slice` for comma-separated base-10
  integers. Whitespace-only input returns `nil`; invalid, empty, or out-of-range
  elements return a `nil` slice and an error wrapping `strconv.NumError` with a
  one-based element position. The unsigned parser rejects either sign.
- `timex.Sleep(ctx, duration)`, which observes cancellation even for non-positive
  durations, and `runtimex.EqualZero` for direct comparable zero checks.
- `stringx.DesensitizerFunc`, `NewEmailDesensitizer`, and an email preset; these
  have no v0.30.2 predecessor. See the string guide for stricter generator validation.
- `exp/iterx.Stream` and `Stream2`, constructed with `FromSeq` and `FromSeq2`,
  for optional chainable transformations. Stable `iterx` migrations follow its
  own guide; the experimental streams are a separate choice.

After updating call sites, verify representative JSON payloads and response
bytes, destination binding, logs, routing fallbacks, collection storage and
nilness, time locations, and application startup/shutdown. Compilation alone
does not establish wire-format or resource-lifetime compatibility.

## AI-assisted migration

The [go-toolkit skill](.agents/skills/go-toolkit/SKILL.md) supports migration
from v0.30.2 to v1. Its
[migration index](.agents/skills/go-toolkit/references/migration-v0.30.2-to-v1.md)
provides a companion checklist using the same baseline. See the
[skill setup instructions](README.md#ai-agent-skill) for installation and updates.

Resolve the application's effective dependency version and identify the exact
target tag or commit before migrating. Read that target's package documentation
and migration guides; its source and APIs remain authoritative.

Invoke the installed skill with the intended scope and target:

```text
$go-toolkit Migrate this application's used packages from v0.30.2 to the specified v1 tag or commit.
```

For consistent use, add this upgrade rule to the consuming project's root
`AGENTS.md`:

```markdown
When upgrading github.com/xgfone/go-toolkit from v0.30.2 to v1, use the go-toolkit skill.
Resolve the effective dependency version and exact target tag or commit.
Read the target revision's MIGRATION.md and relevant package migration guides.
```

Keep the skill and migration references synchronized when APIs or upgrade
guidance change. Linking skill files does not generate guidance from code
changes; use only guidance matching the application's selected revision.
