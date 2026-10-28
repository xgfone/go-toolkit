# Migration index: v0.30.2 to v1

This index compares tag `v0.30.2` with the v1 implementation in the source tree
containing this document. All previous APIs below refer to `v0.30.2`. Use the
index and package guides from the v1 tag or revision selected by your application.

Paths in this document are relative to the selected **go-toolkit module source
root**, not this reference's installed location. For usage contracts and version
discovery, return to the skill's `SKILL.md`.

## Establish the upgrade boundary

- Upgrade the toolchain from the v0.30.2 requirement of Go 1.24 to the selected
  target's requirement, currently Go 1.27. The module path remains
  `github.com/xgfone/go-toolkit`.
- Inventory application imports and used symbols, including custom interfaces,
  codecs, generators, middleware, and lifecycle callbacks. Migrate only the packages
  actually used; do not introduce unrelated services or experimental APIs.
- Read the selected target's root `MIGRATION.md` for shared JSON, HTTP, and
  utility changes, then the package-local guides below. For additional details,
  compare signatures, comments, source, and tests between `v0.30.2` and the
  selected v1 revision with `git diff v0.30.2 TARGET -- PACKAGE`.

## Existing package migration guides

| Package   | Target module document        | Important checks                                                                                                                                                                                                                                                                              |
| --------- | ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `app`     | `app/MIGRATION.md`            | Replace `Hook`/`OnNamed`/`GoNamed` callbacks and the old `Module` contract; adopt dependency batches, module task groups, explicit joins and resource release, and independent shutdown budgets.                                                                                              |
| `iterx`   | `iterx/MIGRATION.md`          | `Map` becomes `To`; callback forms become `SumFunc`/`CountFunc`. `Sum`/`Count` now consume values directly. Replace removed `All`, `Any`, `Seq`, `Seq2`, and numeric constraints as documented. Preserve laziness and early termination.                                                      |
| `slicex`  | `slicex/MIGRATION.md`         | Remove the deprecated `Convert` alias in favor of the existing `To`; conversion `Filter` becomes `FilterTo`. Audit removed collection/containment helpers, `nil` versus empty outputs, single-slice sharing, capacity, and containment complexity before using standard replacements.         |
| `mapx`    | `mapx/MIGRATION.md`           | Remove the deprecated `Convert` alias in favor of the existing `To`; construct `SMap` with `make`. Replace removed keys/values/collection/filter/`Pair` helpers with standard iterators and the documented adapters. Preserve overwrite semantics, nilness, and capacity where needed.        |
| `stringx` | `stringx/README.md#migration` | Concrete `Desensitizer` becomes `MaskDesensitizer`; `Desensitizer` is now an interface. Presets become getters/setters. `Builder` becomes `AffixGenerator`; string-producing `Generate(n)` and buffer-appending `Append(dst,n)` replace the old API. Update custom generator implementations. |

Read the package guides from the selected v1 source rather than copying them
from an older installed dependency. The migration boundary remains v0.30.2 to v1
for every package.

## Other API removals and renames

| v0.30.2 API or import                                                    | v1 replacement                                                                                                                                                        |
| ------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `app.DefaultApp`                                                         | `app.Default()`; configure the returned instance or use an application-owned `app.New()`.                                                                             |
| `router.DefaultRouter`                                                   | `router.Default()`; use `router.New()` for an independent router.                                                                                                     |
| `logger.Config.Logger(priority)`                                         | `logger.Config.Middleware(priority)`.                                                                                                                                 |
| `cors.Config.CORS(priority)`                                             | `cors.Config.Middleware(priority)`.                                                                                                                                   |
| `logger.Config.GetRequestBody` and `GetResponse`                         | `GetLogAttrs(w,r) []slog.Attr`; choose named attributes and wrap the default callback to preserve its fields.                                                         |
| Assigning `httpx.Handler200`, `Handler404`, or other predefined handlers | These are constants implementing `http.Handler`. Use an explicit `http.Handler` type for replaceable local variables; `:=` now infers a concrete status-handler type. |
| `random.DefaultCharset` variable                                         | `random.DefaultCharset()` and `random.SetDefaultCharset(charset)`; the setter rejects empty charsets.                                                                 |
| `bytex.CommentHash` / `CommentSlashes`                                   | Pass application-owned `[]byte("#")` / `[]byte("//")` to `RemoveLineComments`.                                                                                        |
| `netx.IPIsOn`                                                            | `netx.IsLocalIP`.                                                                                                                                                     |
| `runtimex.TrimPkgFile`                                                   | `runtimex.TrimSourcePath`.                                                                                                                                            |
| `codeint.Error.TryError(err)`                                            | `codeint.Error.Wrap(err)`.                                                                                                                                            |
| Decoding methods on `codeint.Error` and `result.Response`                | Call the decoder directly; see the JSON replacements below.                                                                                                           |
| `codeint.Error` comparability                                            | Use `errors.Is` for code matching and application-owned comparable map keys; see error matching below.                                                                |
| `iox.Uploader` / `UploaderFunc`                                          | Define the required upload interface/adapter in the application; these abstractions were removed.                                                                     |
| `github.com/xgfone/go-toolkit/redact`                                    | `github.com/xgfone/go-toolkit/exp/redact`; this package is now experimental and has no compatibility guarantee.                                                       |

### Application lifecycle

In v0.30.2, `app.Hook` took `context.Context, *app.App`. v1 removes `Hook`;
lifecycle callbacks and configuration loaders take only `context.Context` and
return an error. Capture the owning `App` in a closure when it is needed. Task
callbacks already took only context in v0.30.2 and keep that signature. The
callback context is supplied at execution time; do not capture a startup context
for shutdown work.

| v0.30.2 API                                          | v1 replacement                                                                                          |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `a.On(stage, hook)` / `a.OnNamed(stage, name, hook)` | `a.On(stage, name, callback)`; use `""` for an unnamed callback.                                        |
| `stage.On(hook)` / `stage.OnNamed(name, hook)`       | `stage.On(name, callback)` on the default `App`.                                                        |
| `a.Go(fn)` / `a.GoNamed(name, fn)`                   | `a.Go(name, fn)`; the same change applies to package-level `app.Go` / `app.GoNamed`.                    |
| `a.SetConfigLoader(hook)`                            | `a.SetConfigLoader(func(context.Context) error)`.                                                       |
| `a.OnCleanup(hook)` / `app.CloserFuncHook(fn)`       | `a.Cleanup(fn)` for a simple closer, or `a.On(app.StageCleanup, name, callback)`.                       |
| `a.AtExit(fn)`                                       | `a.On(app.StageExited, name, callback)` for notifications; `a.Finalize(name, callback)` for the logger. |
| `a.Wait()` / `a.WaitContext(ctx)`                    | `a.Wait(ctx) error`, returning `Run`'s final result.                                                    |

`a.Cleanup(func() error)` remains available. `App.Context`, `App.Done`, `TaskGroup`,
`ReportError`, and `Finalize` are additions; use `Context` to observe cancellation,
and `Done` or `Wait` to observe completion.

Names may be empty. `Module` methods retain their `context.Context, *app.App` parameters.
Callbacks may register future stages during execution, but registering the current
or an earlier stage panics. Configuration setters and module registration must
finish before `Run` starts.

Startup runs configuration, `Init` callbacks, `Module.Init`, `Module.Prepare`,
`Start` callbacks, `Module.Start`, then `Ready` callbacks. All `Init` calls
finish before any `Prepare`; all `Prepare` calls finish before any `Start`.
`Prepare` loads critical cache snapshots and business data before servers
can accept requests.

`Module` now requires `Init`/`Prepare`/`Start`/`Stop`/`Wait`/`Close` (plus `Name`).
`Stop` initiates business shutdown promptly. `Wait` joins additional business
activity outside the module's `TaskGroup`, including required final writes, and
may return `nil` when there is no such activity. The app separately stops and
joins registered group tasks; both joins must succeed before providers stop or
`Close` releases resources. Replace the old `Stop` resource closer with `Close`.
Pure database/Redis modules can leave `Prepare`/`Start`/`Stop`/`Wait` empty;
driver-internal maintenance belongs to the pool's `Close`.

Optional `Dependencies() []string` names required registered modules. Use
`app.WithDependencies(module, names...)` to add dependencies without a custom
wrapper; it copies the supplied names and preserves existing dependencies and
priority. An empty names list returns the original module unchanged; a `nil` module
still panics. Repeated wrapping appends names; repeated names still fail validation.
The app validates unique names and missing/repeated/self/cyclic dependencies before
startup. Explicit dependencies override `Priority`; priority and registration order
break ties within dependency batches. Startup visits providers first. Shutdown visits
users first: each batch calls every `Stop`, cancels its `TaskGroup` instances, then
joins every `Wait` and group before stopping the provider batch.

`TaskGroup` use is optional: resource-only modules may leave their groups unused.
Use `a.TaskGroup(moduleName).Go(name, callback)` for module-owned workers, servers,
and refreshers. `App` handles group `Stop`/`Wait` automatically; modules implement
the business stop protocol and any additional join without repeating these calls.
HTTP `Serve` belongs to the group; HTTP `Wait` joins graceful shutdown and handlers.
Group contexts preserve `Run` values without inheriting its cancellation/deadline;
each group is canceled only in its shutdown batch, so a draining HTTP request can
submit its final job downstream. Admission racing `Stop` is either counted and
joined or rejected. `TaskGroup` lookup remains available after `Run` returns,
so a still-live user can find retained dependencies for its final work; stopped
groups reject new tasks. `App.Go` uses the run context and joins the outermost
batch; it has no module dependency ordering. Track child work explicitly.
Logger writer tasks belong to finalization, outside business groups.

Replace v0.30.2 `App.Wait()` and `App.WaitContext(ctx)` with `App.Wait(ctx) error`.
It returns the same final result as `Run`, supports concurrent/repeated waiters,
and prefers a completed result over a canceled wait context. Canceling a wait
does not stop the app. Use `context.Background()` for unbounded waiting.

After successful business joins, module `Close` precedes `Cleanup` (reverse order),
`Exited`, and `a.Finalize("logger", callback)`. Initialize logging in `StageInit`
and close it only in the finalizer. Write-back caches must persist pending data
before databases close; closing clients must not delete shared Redis data.

In v0.30.2, module `Stop` and the app-level task join shared `SetShutdownTimeout`;
`Cleanup` and `Exited` used unbounded background contexts. In v1, `SetShutdownTimeout`
controls all drain batches. `Cleanup`, `Exited`, and the logger finalizer have
separate `SetCleanupTimeout`, `SetExitedTimeout`, and `SetFinalizerTimeout` budgets.
`Run` cancellation triggers stopping; shutdown contexts retain values without the
running cancellation/deadline. Deadlines are cooperative, not forced goroutine
termination.

If joining fails, `Run` and `Wait` return `app.ErrShutdownIncomplete`. A failed
module's transitive providers keep running while independent branches still stop;
if app-level tasks cannot join, all remaining provider batches are kept. Resource
release, `Cleanup`, `Exited`, and finalization are skipped. `Done` means the attempt
returned, not that users ended. There is no automatic cleanup/retry; arrange
eventual termination and release retained resources only after their users exit.
Task failures and panics are collected, including non-cancellation failures during
draining. The final result does not change after `Run` returns. Panics during `Run`
in metadata, config, module methods, callbacks, and the finalizer become errors and
follow rollback; `Wait` panics retain dependencies. HTTP `Wait` cancellation only
ends the observer wait; `Stop`'s context controls its grace budget. Failed HTTP
`Init` closes its raw listener even when wrapping returns `nil` or panics. See
`app/MIGRATION.md` for the full v0.30.2-to-v1 lifecycle migration. Regenerate files
from `app/cmd/version` so generated initializers use `app.Default()` instead of
`app.DefaultApp`; the v1 command also parses its flags and adds a commit-count
suffix when HEAD is beyond its selected version tag.

### Error matching

In v0.30.2, `codeint.Error` was structurally comparable, but direct comparisons
and `errors.Is` could panic when `Data` contained a slice or map. v1 adds a
private zero-size field to make `codeint.Error` non-comparable, so `errors.Is`
skips direct equality and calls `Error.Is` for code matching. Value receivers,
value-returning constructors and `WithXxx` methods, and implementation of `error`
by both `codeint.Error` and `*codeint.Error` are retained.

In v0.30.2, a typed `nil` `*codeint.Error` target could panic in `Error.Is` or
`errors.Is` through the generic `GetCode()` branch. In v1, `Error.Is` handles
`*codeint.Error` targets explicitly: a `nil` pointer target does not match the
receiver, even when its `Code` is zero; non-`nil` pointer targets match by `Code`.

Replace `==` / `!=` on `codeint.Error` values with `errors.Is` for code matching,
or compare selected fields for a different application-defined equality policy.
`map[codeint.Error]V` and use of `codeint.Error` as a `comparable` type argument
no longer compile; use an explicit comparable key such as `Code` instead. Equality
between two `error` interfaces holding `codeint.Error` values and insertion of
such values into `map[error]V` now panic even with comparable `Data`.
Comparisons of an `error` to `nil` remain valid, and `*codeint.Error` pointers
remain comparable by identity. Replace positional struct literals with keyed
literals, such as `codeint.Error{Code: 400}`, or use `codeint.NewError` because
the struct now contains a private field. Root `MIGRATION.md` documents the
matching example and complete migration actions.

### JSON codecs

The `jsonx` package and its global marshal/unmarshal hooks were removed.
Current HTTP JSON helpers use `encoding/json/v2`. The decoding convenience
methods on `codeint.Error` and `result.Response` were also removed; decode into
these types with standard-library functions or application-owned codecs.

| v0.30.2 call                             | Replacement                                                                                |
| ---------------------------------------- | ------------------------------------------------------------------------------------------ |
| `jsonx.Marshal` / `MarshalBytes`         | `json.Marshal(value)`.                                                                     |
| `jsonx.Unmarshal` / `UnmarshalBytes`     | `json.Unmarshal(data, dst)`.                                                               |
| `jsonx.MarshalWriter(writer, value)`     | `json.MarshalWrite(writer, value)`.                                                        |
| `jsonx.UnmarshalReader(dst, reader)`     | `json.UnmarshalRead(reader, dst)`; note the argument order.                                |
| `MarshalString` / `MarshalStringWithCap` | Marshal to bytes and convert to a string, or use an application-owned writer/buffer.       |
| `UnmarshalString`                        | `json.Unmarshal([]byte(text), dst)` or `json.UnmarshalRead(strings.NewReader(text), dst)`. |

Replace all seven removed decoding methods from v0.30.2 as follows:

| v0.30.2 method                                                     | v1 replacement                                                        |
| ------------------------------------------------------------------ | --------------------------------------------------------------------- |
| `codeint.Error.Decode`, `result.Response.Decode`                   | `decode(dst)`; pass the destination pointer directly to the callback. |
| `codeint.Error.DecodeJSON`, `result.Response.DecodeJSON`           | `json.UnmarshalRead(reader, dst)`                                     |
| `codeint.Error.DecodeJSONBytes`, `result.Response.DecodeJSONBytes` | `json.Unmarshal(data, dst)`                                           |
| `codeint.Error.DecodeJSONString`                                   | `json.UnmarshalRead(strings.NewReader(text), dst)`                    |

Here `json` denotes `encoding/json/v2`, and `dst` is a non-`nil` destination
pointer, such as `&codedErr` for a `codeint.Error` or `&response` for a
`result.Response`. Pass `json.Options` to the standard-library functions when
needed. For a non-`null` JSON error, initialize `result.Response.Error` with a
concrete error pointer, such as
`response := result.Response{Error: new(codeint.Error)}`, before decoding.

Rework custom `SetMarshalWriterFunc`/`SetUnmarshalReaderFunc` integrations
explicitly; there is no replacement global hook. The old default writer already
disabled HTML escaping, and only its writer form appended a newline. v1 writer
operations no longer append that newline, including HTTP responses and requests.

The v1 defaults are case-sensitive for struct fields, reject duplicate names and
invalid UTF-8, and require a single complete JSON value through EOF. `nil` maps
and ordinary slices encode as empty collections, map member order is unspecified,
byte arrays encode as base64 strings, and ordinary array decoding requires the
exact length. JSON `null` now resets existing scalar values. A lower-case `code`
member no longer fills `codeint.Error.Code` and may be ignored without an error.
Review custom pointer-receiver serializers and the numeric-only `,string` tag
option.

The new `omitempty` meaning can add zero numeric fields or remove empty values
behind interfaces: `codeint.Error{}` now encodes as `{"Code":0}`, while
`result.Ok("")` encodes as `{}`. Plain errors such as `errors.New` contain no
exported fields and can fail JSON encoding; `time.Duration` also has no default
representation in the current codec. Use explicit wire DTOs or application-owned
codecs when preserving an old contract. Some convenience responders discard
rendering errors, so check payload serialization or own the responder. Root
`MIGRATION.md` documents the concrete changes and a writer retaining the old
newline/HTML policy.

### Time policy

`timex.Format`/`Formats` and their getters/setters are removed, as are
`Location`, `GetLocation`, `SetLocation`, and `Unix` in v1.
Keep formatting and parsing policies in the application. Use `time.Unix`
and explicit `In(location)` when a specific location is required.

`timex.Now` now uses the replaceable clock, initially `time.Now`, without
converting to the old UTC default. `Today` uses the returned time's location.
Code requiring UTC should request it explicitly; a custom `SetNowFunc`
can return UTC for toolkit clock consumers, including string time generators.
`App.BuildTime` now returns the stored instant in `time.Local` instead of the
v0.30.2 `timex.Location` policy (UTC by default). Convert explicitly when a
particular location is required. Restore `time.Now` after tests and do not
replace the clock concurrently with its consumers. `timex.Sleep` is a new
cancellation-aware helper.

## Behavioral changes to verify

- **HTTP clients:** automatic JSON decoding writes directly into the caller's
  destination; pass a non-`nil` pointer. Automatic handling accepts status 200,
  while a response callback owns status/format handling. Debug logging now
  includes method, host, path, and status rather than bodies and headers.
- **HTTP binding and responses:** `BindPath` reads populated path wildcards
  and then applies defaults/validation. Test wrapped errors and response status
  selection; invalid explicit `codeint.Error` HTTP statuses normalize to 500.
  Informational response headers no longer prevent a later final response;
  101 remains final. The `Context` writer rejects statuses outside [100, 999].
- **HTTP headers:** MIME types are normalized and charset parameters are
  unquoted. `Accept`/`AcceptEncoding`/`AcceptLanguage` read all header values,
  honor quoted commas and `q` parameters, and reject invalid weights. WebSocket
  checks accept comma-separated `Connection` and `Upgrade` tokens. See root
  `MIGRATION.md` for examples; these changes can affect content-type binding
  and negotiation.
- **Routing:** the default backend sends unmatched paths and methods to the
  fallback, and leaves handler-generated 404 responses alone. An explicit
  catch-all takes precedence. Route inspection returns snapshots; changing an
  inspected slice is not a way to register or update routes.
- **Logging:** `GetLogAttrs` is evaluated after the downstream handler. Defaults
  include status, safe error text, and the original sensitive error when present;
  replacing the callback changes those fields. Explicitly preserve or replace
  fields required by the application's logging policy.
- **Lifecycle:** cancellation and configured signals can interrupt startup.
  An empty signal list disables signal handling. Managed background failures are
  retained and can stop the app. `module.HttpServer.Start` requires a running
  app and participates in rollback/shutdown; integrate it through `app.Module`.
- **Utilities:** comment stripping respects double-quoted text and escapes.
  Network address conversion rejects `nil`/malformed values and preserves IPv6
  zones. `nil` values with an `IsZero` method count as zero without invoking that
  method; `EqualZero` uses ordinary equality instead. `Stacks` now includes its
  final frame. Text binding copies string inputs passed to `UnmarshalText` instead
  of exposing an unsafe string alias. Cryptographic integer generation no longer
  falls back to weaker randomness after a crypto-source failure. Generated
  millisecond time components are padded to three digits; string generators
  now validate minimum and exact output lengths.

New APIs without v0.30.2 predecessors also include `strconvx.ParseInt64Slice` and
`ParseUint64Slice`, `timex.Sleep`, `runtimex.EqualZero`, `stringx` email masking
and `DesensitizerFunc`, and the optional experimental `exp/iterx` streams. Read
their contracts in root `MIGRATION.md` and the target package source.

Run tests for the used packages and their application integration after updating
call sites. Verify behavior and output compatibility in addition to compilation,
especially collection storage/nilness, JSON payloads, time locations, routing,
logging, and lifecycle cancellation. Report any checks that could not be run.
