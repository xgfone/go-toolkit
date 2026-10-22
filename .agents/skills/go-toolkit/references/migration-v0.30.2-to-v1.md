# Migration index: v0.30.2 to v1 development

This index was initially checked against tag `v0.30.2` and source commit
`a3df9f89e02c9996b6b6b5d4340ca5a61a3c0e45`, and includes the current application
lifecycle API described below. It is not a promise about a final `v1.0.0` release
or every beta. Check the exact target source before applying a replacement;
later commits and individual prereleases can differ.

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
- Read the target version's package-local guides. There is no root-level
  `MIGRATION.md` in the checked snapshot. For packages without a guide, compare
  signatures, comments, source, and tests between the actual source and target
  revisions. In a checkout, `git diff SOURCE TARGET -- PACKAGE` is useful.

## Existing package migration guides

| Package   | Target module document        | Important checks                                                                                                                                                                                                                                                                              |
| --------- | ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `iterx`   | `iterx/MIGRATION.md`          | `Map` becomes `To`; callback forms become `SumFunc`/`CountFunc`. `Sum`/`Count` now consume values directly. Replace removed `All`, `Any`, `Seq`, `Seq2`, and numeric constraints as documented. Preserve laziness and early termination.                                                      |
| `slicex`  | `slicex/MIGRATION.md`         | `Convert` becomes `To` and conversion `Filter` becomes `FilterTo`. Audit removed collection/containment helpers, nil versus empty outputs, single-slice sharing, capacity, and containment complexity before using standard replacements.                                                     |
| `mapx`    | `mapx/MIGRATION.md`           | `Convert` becomes `To`; construct `SMap` with `make`. Replace removed keys/values/collection/filter/Pair helpers with standard iterators and the documented adapters. Preserve overwrite semantics, nilness, and capacity where needed.                                                       |
| `stringx` | `stringx/README.md#migration` | Concrete `Desensitizer` becomes `MaskDesensitizer`; `Desensitizer` is now an interface. Presets become getters/setters. `Builder` becomes `AffixGenerator`; string-producing `Generate(n)` and buffer-appending `Append(dst,n)` replace the old API. Update custom generator implementations. |

Do not copy a module guide from an older installed dependency and assume it
describes the target. For a beta-to-beta or beta-to-development upgrade, also
check changes after that beta. For example, the current iterator adapters use
`Skip`; earlier v1 development used `Drop`.

## Other API removals and renames

| Previous API or import                                                   | Replacement in the checked source                                                                                                     |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `app.DefaultApp`                                                         | `app.Default()`; configure the returned instance or use an application-owned `app.New()`.                                             |
| `router.DefaultRouter`                                                   | `router.Default()`; use `router.New()` for an independent router.                                                                     |
| `logger.Config.Logger(priority)`                                         | `logger.Config.Middleware(priority)`.                                                                                                 |
| `cors.Config.CORS(priority)`                                             | `cors.Config.Middleware(priority)`.                                                                                                   |
| `logger.Config.GetRequestBody` and `GetResponse`                         | `GetLogAttrs(w,r) []slog.Attr`; choose named attributes and wrap the default callback to preserve its fields.                         |
| Assigning `httpx.Handler200`, `Handler404`, or other predefined handlers | These are immutable constants implementing `http.Handler`. Store custom handlers locally and configure the router/handler using them. |
| `random.DefaultCharset` variable                                         | `random.DefaultCharset()` and `random.SetDefaultCharset(charset)`; the setter rejects empty charsets.                                 |
| `bytex.CommentHash` / `CommentSlashes`                                   | Pass application-owned `[]byte("#")` / `[]byte("//")` to `RemoveLineComments`.                                                        |
| `netx.IPIsOn`                                                            | `netx.IsLocalIP`.                                                                                                                     |
| `runtimex.TrimPkgFile`                                                   | `runtimex.TrimSourcePath`.                                                                                                            |
| `codeint.Error.TryError(err)`                                            | `codeint.Error.Wrap(err)`.                                                                                                            |
| `iox.Uploader` / `UploaderFunc`                                          | Define the required upload interface/adapter in the application; these abstractions were removed.                                     |
| `github.com/xgfone/go-toolkit/redact`                                    | `github.com/xgfone/go-toolkit/exp/redact`; this package is now experimental and has no compatibility guarantee.                       |

### Application lifecycle

Current lifecycle callbacks have the signature `func(ctx context.Context) error`.
Capture the owning `*app.App` in a closure when it is needed. The callback context
is supplied at execution time; do not capture a startup context for shutdown work.

| Purpose                                | Current interface                                                       |
| -------------------------------------- | ----------------------------------------------------------------------- |
| Register a callback on an App          | `a.On(stage, name, callback)`                                           |
| Register a callback on the default App | `stage.On(name, callback)` or `app.Default().On(stage, name, callback)` |
| Start a managed task                   | `a.Go(name, callback)` or `app.Go(name, callback)`                      |
| Set the configuration loader           | `a.SetConfigLoader(func(ctx context.Context) error)`                    |
| Register a simple resource closer      | `a.Cleanup(func() error)`                                               |
| Observe cancellation                   | `a.Context().Done()`                                                    |
| Wait for shutdown to finish            | `a.Done()`, `a.Wait()`, or `a.WaitContext(ctx)`                         |

Names may be empty. Module methods retain their `context.Context, *app.App` parameters.
Callbacks may register future stages during execution, but registering the current
or an earlier stage panics. Configuration setters and module registration must
finish before `Run` starts.

Startup runs configuration, Init callbacks, Module.Init, Start callbacks,
Module.Start, then Ready callbacks. Required request data must be initialized in
Init or Start: servers may already accept requests during Ready.

Shutdown cancels the run and lifecycle contexts, runs Stopping callbacks, stops
successfully initialized modules in reverse order, waits for managed tasks, then
runs Cleanup callbacks in reverse registration order and Exited callbacks.
Shutdown callbacks share an independent timeout context. Remaining shutdown
callbacks still run after errors or timeout; timeout can leave managed tasks
running when Done closes. Resources needed by those tasks should stay open until
Cleanup, or a Stopping callback should explicitly join the tasks before Module.Stop.

### JSON codecs

The `jsonx` package and its global marshal/unmarshal hooks were removed.
Current HTTP and result helpers use `encoding/json/v2`.

| Previous call                            | Replacement                                                                                |
| ---------------------------------------- | ------------------------------------------------------------------------------------------ |
| `jsonx.Marshal` / `MarshalBytes`         | `json.Marshal(value)`.                                                                     |
| `jsonx.Unmarshal` / `UnmarshalBytes`     | `json.Unmarshal(data, dst)`.                                                               |
| `jsonx.MarshalWriter(writer, value)`     | `json.MarshalWrite(writer, value)`.                                                        |
| `jsonx.UnmarshalReader(dst, reader)`     | `json.UnmarshalRead(reader, dst)`; note the argument order.                                |
| `MarshalString` / `MarshalStringWithCap` | Marshal to bytes and convert to a string, or use an application-owned writer/buffer.       |
| `UnmarshalString`                        | `json.Unmarshal([]byte(text), dst)` or `json.UnmarshalRead(strings.NewReader(text), dst)`. |

Rework custom `SetMarshalWriterFunc`/`SetUnmarshalReaderFunc` integrations
explicitly; there is no replacement global hook. Audit JSON tags, custom
marshalers, nil versus empty values, trailing input, and options using the
target standard-library API. Do not assume the previous adapter's serialization
policy is preserved.

### Time policy

`timex.Format`/`Formats` and their getters/setters are removed, as are
`Location`, `GetLocation`, `SetLocation`, and `Unix` in the checked source.
Keep formatting and parsing policies in the application. Use `time.Unix`
and explicit `In(location)` when a specific location is required.

`timex.Now` now uses the replaceable clock, initially `time.Now`, without
converting to the old UTC default. `Today` uses the returned time's location.
Code requiring UTC should request it explicitly; a custom `SetNowFunc`
can return UTC for toolkit clock consumers, including string time generators.
Restore `time.Now` after tests and do not replace the clock concurrently with
its consumers. `timex.Sleep` is a new cancellation-aware helper.

## Behavioral changes to verify

- **HTTP clients:** automatic JSON decoding writes directly into the caller's
  destination; pass a non-nil pointer. Automatic handling accepts status 200,
  while a response callback owns status/format handling. Debug logging now
  includes method, host, path, and status rather than bodies and headers.
- **HTTP binding and responses:** `BindPath` reads populated path wildcards
  and then applies defaults/validation. Test wrapped errors and response status
  selection; invalid explicit `codeint.Error` HTTP statuses normalize to 500.
  Informational response headers no longer prevent a later final response.
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
  Network address conversion rejects nil/malformed values and preserves IPv6
  zones. Nil values with an `IsZero` method count as zero without invoking that
  method; `EqualZero` uses ordinary equality instead. Generated millisecond
  time components are padded to three digits.

Run tests for the used packages and their application integration after updating
call sites. Verify behavior and output compatibility in addition to compilation,
especially collection storage/nilness, JSON payloads, time locations, routing,
logging, and lifecycle cancellation. Report any checks that could not be run.
