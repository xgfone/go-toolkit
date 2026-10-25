# `app`: v0.30.2 to v1

This guide compares the API at tag `v0.30.2` with the v1 implementation in this
source tree. Use the guide from the v1 tag or revision selected by your
application. Examples omit imports and declarations of application resources.
Upgrade the toolchain from Go 1.24 to Go 1.27 before updating the module.

## API migration from v0.30.2

| v0.30.2 API                                          | v1 replacement                                                                                                                                            |
| ---------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `app.DefaultApp`                                     | `app.Default()`; the default instance is no longer an assignable exported variable. Use `app.New()` for an application-owned instance.                    |
| `app.Hook`, taking `context.Context, *app.App`       | Callbacks take only `context.Context` and return an error. Capture the owning `App` in a closure.                                                         |
| `a.On(stage, hook)` / `a.OnNamed(stage, name, hook)` | `a.On(stage, name, callback)`; use `""` for an unnamed callback.                                                                                          |
| `stage.On(hook)` / `stage.OnNamed(name, hook)`       | `stage.On(name, callback)` on the default `App`.                                                                                                          |
| `a.SetConfigLoader(hook)`                            | `a.SetConfigLoader(func(context.Context) error)`; capture the `App` when needed.                                                                          |
| `a.Go(fn)` / `a.GoNamed(name, fn)`                   | `a.Go(name, fn)`; use `""` for an unnamed task. The same change applies to package-level `app.Go` / `app.GoNamed`.                                        |
| `app.CloserFuncHook(fn)` / `a.OnCleanup(hook)`       | Use `a.Cleanup(fn)` for a simple closer, or `a.On(app.StageCleanup, name, callback)` for a context-aware callback.                                        |
| `a.AtExit(fn)`                                       | Use `a.On(app.StageExited, name, callback)` for a final notification; move logger release to `a.Finalize(name, callback)`.                                |
| `a.Wait()` / `a.WaitContext(ctx)`                    | `a.Wait(ctx) error`; it now returns `Run`'s final result rather than only wait cancellation. Use `context.Background()` for unbounded waiting.            |
| `Module` `Name/Init/Start/Stop`                      | Keep `Name`/`Init`/`Start`, add `Prepare`/`Wait`/`Close`, and split the old `Stop` into stop signaling, joining, and resource release as described below. |

The six public `Stage` constants and `a.Cleanup(func() error)` remain available.
`App.Stop()` still requests shutdown without waiting. `App.Context()`,
`App.Done()`, module `TaskGroup` instances, dependency declarations,
`ReportError`, and `Finalize` are additions.

For example, replace a v0.30.2 named hook:

```go
a.OnNamed(app.StageInit, "logger", func(ctx context.Context, owner *app.App) error {
    return setupLogger(ctx, owner.Name())
})
```

with a v1 callback that captures the same `App`:

```go
a.On(app.StageInit, "logger", func(ctx context.Context) error {
    return setupLogger(ctx, a.Name())
})
```

`Module` methods still receive `context.Context, *app.App`; only lifecycle
callbacks and configuration loaders lose the `App` argument. Task callbacks
already took only context in v0.30.2 and keep that signature.

## Lifecycle changes from v0.30.2

The v0.30.2 startup sequence was configuration -> `StageInit` -> `Module.Init` ->
`StageStart` -> `Module.Start` -> `StageReady`. v1 inserts **all `Module.Prepare` calls**
between `Module.Init` and `StageStart`. `Module` priority still defaults to 1 and
sorts descending; v1 dependency declarations take precedence over priority.
The module-priority comparison is also overflow-safe for extreme `int` values.

In v0.30.2, shutdown ran `StageStopping`, every initialized `Module.Stop` in
reverse order, the app-level task join, `StageCleanup`, and `StageExited`. A
database's old `Stop` could therefore close its pool before app-level tasks
finished. `Cleanup` and `Exited` received unbounded background contexts and ran
even after a failed join. v1 instead drains users before stopping providers,
adds `Module.Close` after all joins, gives cleanup and exit their own budgets,
and retains resources when exit cannot be proven. Logging moves from
`StageExited` to the finalizer.

In v0.30.2, stage callbacks, configuration loading, and module methods could panic
out of `Run`; managed task panics were recovered. v1 also recovers panics from the
lifecycle operations and metadata read during `Run`, so they follow the failure and
rollback paths described below. Managed task failures during shutdown are preserved
unless they consist solely of expected cancellation. Cancellation, including configured
signals, can interrupt startup in v1; `a.SetSignals()` with no arguments disables
OS-signal handling.

## `Module` contract

```go
type Module interface {
    Name() string
    Init(context.Context, *app.App) error
    Prepare(context.Context, *app.App) error
    Start(context.Context, *app.App) error
    Stop(context.Context, *app.App) error
    Wait(context.Context, *app.App) error
    Close(context.Context, *app.App) error
}
```

- `Init` acquires resources. On failure, it must clean its partial acquisitions.
- `Prepare` builds business state after **every `Init`** succeeded: load snapshots
  from configuration/MySQL, warm caches, and assemble request processing.
- `Start` starts activity and returns when ready. **Every `Prepare`** finishes
  before any `Start`. Use the module's task group for business goroutines;
  component-managed business activity outside the group must be joined through
  `Wait`.
- `Stop` closes admission and initiates shutdown promptly. It does not join work
  or release shared dependencies.
- `Wait` proves additional business activity outside the module's task group,
  including its required final writes, has ended. It may return `nil` immediately
  when there is no additional activity. The app independently joins group tasks;
  both joins must succeed before providers stop or resource release begins.
- `Close` releases resources after all business joins succeeded. Use it for
  database/Redis pools, files, and remaining module-local resources.

A pure database module can leave `Prepare`, `Start`, `Stop`, and `Wait` empty.
It stays usable during its callers' draining. Driver-internal maintenance
belongs to the pool's `Close`, rather than the application's business-task join.

Shutdown methods must handle initialized-but-not-started and partial-start
states and should be idempotent. If `Prepare` or `Start` fails, initialized modules
are stopped, joined, and closed. Managed work started during a failed `Init` is
still joined, but `Init` owns its partial-resource rollback. Modules are one-shot;
create new instances for another run. Metadata must remain stable and wrappers
must forward dependency and priority metadata explicitly if needed.

Panics in metadata read during `Run`, the configuration loader, module methods,
stage callbacks, and the finalizer become errors with their original cause.
Startup panics therefore enter rollback. Shutdown panics do not skip later safe
callbacks; a `Wait` panic means exit is unproven and retains the user's providers.
Validation panics from misuse outside `Run` (for example a second `Run`) still panic.

## Dependency batches

Modules may implement `Dependencies() []string`, returning registered module
names. Names must be unique; missing, repeated, self, and cyclic dependencies
are errors returned by `Run` before configuration or initialization. The complete
plan is validated before any module callback is registered.

Dependencies precede users in `Init`, `Prepare`, and `Start`. Among modules eligible
in the same dependency batch, descending `Priority() int` (default 1) and then
registration order determine order. Explicit dependency edges override priority.

Use `app.WithDependencies` to add dependencies to an existing module without
defining a wrapper type. It preserves the module's name, lifecycle methods,
priority, and existing dependencies. Supplied names are copied and appended to
existing dependencies, including those from earlier calls to `WithDependencies`.
Repeated names remain validation errors; wrapping does not replace dependencies.
With no supplied names, `WithDependencies` returns the original module unchanged,
preserving its concrete type and optional interfaces. A `nil` module still panics.

For example, declare an HTTP server's downstream worker when registering it:

```go
server := module.NewHttpServer("http", listenAddr, handler)
a.Use(app.WithDependencies(server, "worker"))
```

Declare the full pipeline, including indirect business dependencies:

```text
HTTP -> Worker -> Cache -> MySQL / Redis
```

Shutdown processes dependency batches in reverse order. For each batch it:

1. Calls every initialized module's `Stop` in reverse batch order.
2. Seals and cancels every task group in the batch.
3. Calls every initialized module's `Wait` and joins every group in reverse order.
4. Stops a provider only after its users have joined. If a user cannot join,
   retains its transitive providers and continues stopping independent branches.

An accepted HTTP request may therefore submit its final job to Worker while
HTTP drains. Worker remains usable, including its task admission, until HTTP
has joined. Cache remains usable until Worker joins; databases remain usable
until all business work has joined. Independent modules in a batch all receive
`Stop` before any `Wait` is called. `Stop` must not block waiting for another member.

Do not model this as one global `Stop` round followed by one global `Wait` round:
that can shut down a queue before its upstream requests finish producing work.

## Task groups and contexts

`TaskGroup` is a shared capability for modules that own business goroutines, such
as servers, consumers, and cache refreshers. Use is optional: resource-only
modules may leave their group unused.

Once `Run` validates the module plan, obtain `group := a.TaskGroup(moduleName)`
(normally in `Init` or `Start`) and launch work with
`group.Go(taskName, func(ctx context.Context) error { ... })`.
Tasks, including accepted child work, must be registered before their group
stops. Lookup remains available after `Run` returns: a live user can still find
and submit its final work to a retained downstream group. Stopped groups remain
observable but reject new tasks. Registration after group `Stop` panics.
Admission and the task count share
a lock, so a task racing `Stop` is either admitted and joined or rejected.

The group's `Context` retains `Run`'s values without inheriting its deadline or
cancellation. It is canceled after `Module.Stop` in that module's shutdown batch.
Treat cancellation as a shutdown signal and apply an explicit policy to accepted
work: finish, cancel, or durably return it. Use the shutdown method's context for
final writes that must survive running-context cancellation. Do not retain a
startup operation's context as the lifetime of a server or consumer.

The app seals, cancels, and joins the group automatically in its dependency batch.
Modules normally launch tasks with `Go` without repeating the group's `Stop` and `Wait`
calls in their lifecycle methods. `Module.Stop` handles the business stop protocol;
`Module.Wait` joins handlers, protocol sessions, or other business activity not
represented by group tasks. The app joins the group even if `Module.Wait` is empty.
It considers the module drained only after both waits succeed. A group only tracks
registered callbacks; detached child work still needs an explicit owner and join.

Task errors and panics trigger `App.Stop` and are collected in `Run`'s final result;
ordinary errors caused solely by the task context's cancellation are suppressed.
Other errors, including failures during draining, remain visible. `ReportError`
can record failures from a separately joined asynchronous operation; prefer
returning errors from managed tasks when possible.

`App.Go` remains for app-level work. It uses the run context and is canceled on
`App.Stop`, parent cancellation, signals, or a task failure. These tasks join with
the outermost shutdown batch before provider batches stop. They have no module
dependency ordering: migrate HTTP `Serve`, queue consumers, cache refreshers, and
other ordered business work to the owning module's `TaskGroup`. Goroutines started
directly with `go` and detached children are not automatically tracked.

For optional periodic scans and maintenance that can begin after HTTP starts,
register a `StageReady` callback that starts the task with `App.Go`:

```go
a.On(app.StageReady, "", func(context.Context) error {
    a.Go("", func(ctx context.Context) error {
        refreshOptionalCacheUntilCanceled(ctx)
        return nil
    })
    return nil
})
```

The task starts asynchronously, receives the run context, reports failures in
`Run`/`Wait`, and joins before providers stop. Startup failure before the
registration callback executes leaves the task unstarted. Use an empty name
when the callback always returns `nil`.

Required initial cache data belongs in `Prepare` or `StageStart`. Queue consumers
that must remain available to draining HTTP handlers belong in a module group;
make the HTTP module depend on that consumer and the consumer depend on its
database. An app-level periodic scanner may share the consumer's business logic,
but must serialize conflicting writes with callback handling.

Internal logger writers and rotation workers belong to logger finalization,
not the business groups that are canceled during draining.

## `App` stages and final result

Startup is configuration -> `StageInit` -> `Module.Init` -> `Module.Prepare` ->
`StageStart` -> `Module.Start` -> `StageReady`. `StageInit` is suitable for logging;
`StageStart` follows the new module preparation phase. HTTP may already serve during
`StageReady`, so required request data must be ready before `Start`. A strict global
admission barrier requires an explicit gate in addition to readiness reporting.

Stopping cancels `App.Context` and the run context, executes `StageStopping`, then
performs the dependency batches above. After every join succeeds, module `Close`
runs in reverse dependency order, followed by `Cleanup` (reverse registration),
`Exited` (registration order), and the single finalizer. `App.Done` closes after
the final result is stored, immediately before `Run` returns.

`App.Stop()` requests shutdown; it does not wait. Replace v0.30.2 `App.Wait()`
and `App.WaitContext(ctx)` with `App.Wait(ctx) error`. For unbounded waiting
use `a.Wait(context.Background())`. `Wait` returns the same final error as `Run`,
including startup, task, cleanup, finalizer, or incomplete shutdown failures.
It supports concurrent and repeated callers. If the app is already done, its
result takes precedence over a canceled waiting context. Canceling a wait returns
`ctx.Err` without stopping the app or changing budgets. Before `Run`, `Wait`
waits for a future run. Never call `App.Wait` from lifecycle code.

## Independent budgets and incomplete shutdown

`Run(ctx)` retains its running-lifetime meaning: cancellation or deadline triggers
shutdown. Shutdown preserves values but drops this cancellation and deadline.

| Setter                | Default    | Covers                                                 |
| --------------------- | ---------- | ------------------------------------------------------ |
| `SetShutdownTimeout`  | 30 seconds | `Stopping`, every batch's `Stop`/`Wait` and task joins |
| `SetCleanupTimeout`   | 5 seconds  | `Module.Close` and `Cleanup` callbacks                 |
| `SetExitedTimeout`    | 5 seconds  | `Exited` notifications                                 |
| `SetFinalizerTimeout` | 5 seconds  | Final logger drain/flush/close                         |

Each budget starts when its part begins. Batches share the remaining drain
budget; closers share the cleanup budget. Shutdown callbacks are attempted even
after their budget expires. These deadlines are cooperative: wrapping a blocking
call in a goroutine does not make resource release complete. Operations such as
`sql.DB.Close` and `file.Sync` do not accept a context themselves.

If a module or group cannot join, `Run` and `Wait` return an error matching
`app.ErrShutdownIncomplete`. Its transitive providers remain running for live
users; independent branches still receive `Stop` and are joined using the
remaining drain budget. If app-level tasks cannot join, all remaining provider
batches are retained because those tasks have no declared dependencies. All
`Close`, `Cleanup`, `Exited`, and finalization are skipped. `Done` signals that
the attempt returned, not that every task exited. There is no automatic background
cleanup or retry. Arrange eventual termination and release retained resources only
after their users exit. Errors arriving after `Run` do not mutate its result.

`Stop` errors alone do not prevent release when every join proves completion.
Resource-close errors are collected while remaining closers, notifications,
and finalization are attempted.

## HTTP, caching, and logging

In v0.30.2, `HttpServer.Start` launched an unmanaged `Serve` goroutine, and
`Stop` called `Shutdown` synchronously. In v1, `Start` requires a running `App`
and launches `Serve` in the module's `TaskGroup`; `Stop` initiates shutdown,
`Wait` joins business activity, and `Close` releases remaining resources.
`Serve` failures are included in `Run`'s final result. Register the server with
`a.Use`, and declare its downstream dependencies with `app.WithDependencies`
as shown above.

With a `nil` handler, v0.30.2 relied on `net/http`'s default-mux lookup while
serving; v1 captures `http.DefaultServeMux` during `Init`. Configure or replace
the default mux before `App.Run`; do not reassign `http.DefaultServeMux` while
the app is running.

`HttpServer` owns `Serve` through its module task group, which the app joins
independently of `HttpServer.Wait`. `Stop` gates new handlers, closes its listener,
and begins a separately tracked graceful shutdown operation. When `Stop`'s context
expires it cancels request contexts and closes active connections. `Wait` only
observes the graceful operation and admitted handlers; canceling a shorter observer
wait does not stop the server or shorten `Stop`'s grace period. Completion wins
when request completion races cancellation. A returned `Serve` or a closed socket
does not prove a handler ended. Explicitly own and join detached/hijacked work,
including WebSockets. Request contexts remain alive during ordinary draining.
Failed `Init` releases its acquired listening socket, including when listener
wrapping returns `nil` or panics, and leaves the server disabled.

Read-only cache snapshots are built in `Prepare`, refreshed in `Start`, and retained
until readers finish. Publish a complete snapshot instead of exposing a partial
map or shared Redis version. An asynchronous write-back cache must stop after its
producers, persist its pending writes while MySQL/Redis are available, and report
failure instead of silently discarding dirty data. Memory-only pending writes
cannot survive forced process termination. Close only resources owned by this
app; do not delete shared Redis cache data or stop an external database server.

Initialize logging in `StageInit` and register `a.Finalize("logger", callback)`.
It runs after all `Exited` notifications with a fresh budget. The finalizer should
stop rotation and asynchronous writers, drain the queue, flush user buffers,
optionally `Sync` the file, and close it, checking errors. Only one finalizer is
accepted; it can coordinate multiple outputs. Keep logging outside normal module
`Close` so cleanup and `Exited` can still log. Report the final returned error to
stderr after the logger has closed.

## Migrating a v0.30.2 module

For each existing `Module` implementation and wrapper:

1. Keep `Init` for resource acquisition and move required business-data loading
   to `Prepare` when it must run after every module's `Init`.
2. Keep `Start` for starting activity. Move module-owned `App.GoNamed`/`App.Go`
   work to `a.TaskGroup(moduleName).Go(name, callback)` when dependency order matters.
3. Split the old `Stop`: initiate business shutdown in `Stop`, join activity outside
   the `TaskGroup` in `Wait`, and release resources in `Close`. The `App` joins group
   tasks automatically. Return `nil` from phases with no work.
4. Declare required module names with `Dependencies` or `WithDependencies`;
   forward dependency and priority metadata through custom wrappers.
5. Move logger cleanup previously registered with `AtExit` or `StageExited` to
   `Finalize`, and update callback registration using the API table above.

## Generated build metadata

Regenerate files produced by `app/cmd/version`, or replace generated references
to `app.DefaultApp` with `app.Default()`. The v1 command also parses its flags
and appends the count of commits after its selected version tag when nonzero,
such as `v1.2.0-3`; check consumers that parse `App.Version` as a plain release
tag.

`App.BuildTime` now returns the stored instant in `time.Local`; v0.30.2 used
`timex.Location`, which defaulted to UTC. Use `.UTC()` or `.In(location)` when
your application needs a specific output location.
