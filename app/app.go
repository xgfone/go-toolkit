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

// Package app manages application metadata, lifecycle callbacks, modules, and
// background tasks. Use [Stage.On] or [App.On] for callbacks, and [Use] or
// [App.Use] to register modules. See [App.Run] for their execution order.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var defaultApp = New()

// Default returns the package-level default [App] instance.
func Default() *App {
	return defaultApp
}

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		setCommitFromBuildSettings(defaultApp, info.Settings)
	}
}

func setCommitFromBuildSettings(a *App, settings []debug.BuildSetting) {
	for _, setting := range settings {
		if setting.Key == "vcs.revision" {
			a.SetCommit(setting.Value[:min(7, len(setting.Value))])
			return
		}
	}
}

type state int

const (
	stateNew state = iota
	stateRunning
	stateStopping
	stateExited
)

// App is a lightweight backend application lifecycle manager.
type App struct {
	mu sync.Mutex

	name    atomic.Value
	commit  atomic.Value
	version atomic.Value
	builtat atomic.Int64

	shutdownTimeout time.Duration
	signals         []os.Signal

	modules   []Module
	callbacks [phaseCount][]namedCallback
	stage     lifecyclePhase // Last public stage; phaseConfig before StageInit.
	state     state

	ctx           context.Context
	cancelContext context.CancelFunc
	runCtx        context.Context
	cancelRun     context.CancelFunc

	wg    sync.WaitGroup
	errCh chan error
	done  chan struct{}
}

// New creates an [App] with minimal default behavior.
//
// By default, it:
//   - sets version to "0.0.0"
//   - derives name from the first entry of [os.Args] using [filepath.Base]
//     and [strings.TrimSuffix] to remove the ".exe" suffix
//   - uses a minimal [flag] config loader (see [App.SetConfigLoader])
//   - uses 30 seconds as shutdown timeout
//   - listens to [syscall.SIGINT], [syscall.SIGTERM]
func New() *App {
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		state:         stateNew,
		done:          make(chan struct{}),
		ctx:           ctx,
		cancelContext: cancel,
	}
	app.SetName(strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe"))
	app.SetConfigLoader(app.defaultFlagConfigLoader)
	app.SetShutdownTimeout(30 * time.Second)
	app.SetSignals(os.Interrupt, syscall.SIGTERM)
	app.SetVersion("0.0.0")
	app.SetCommit("")
	return app
}

// Run starts the default app, see [App.Run].
func Run(ctx context.Context) error {
	return defaultApp.Run(ctx)
}

// Name is a convenience function that returns the default app name.
func Name() string {
	return defaultApp.Name()
}

// Version is a convenience function that returns the default app version.
func Version() string {
	return defaultApp.Version()
}

// Name returns app name.
func (a *App) Name() string {
	return a.name.Load().(string)
}

// Commit returns app commit.
func (a *App) Commit() string {
	return a.commit.Load().(string)
}

// Version returns app version.
func (a *App) Version() string {
	return a.version.Load().(string)
}

// BuildTime returns the time when built the app, in [time.Local].
func (a *App) BuildTime() time.Time {
	return time.Unix(a.builtat.Load(), 0)
}

// Context returns the app's lifecycle cancellation context. It is created by
// [New] and remains the same before, during, and after [App.Run]. It is canceled
// when shutdown is requested, before [StageStopping], including on startup failure.
// Use [App.Done] or [App.WaitContext] to wait for shutdown to finish.
//
// This context carries no values or deadline from the parent passed to [App.Run];
// callbacks and managed background tasks receive that parent's derived run
// context instead. Shutdown callbacks receive a separate shutdown context.
func (a *App) Context() context.Context { return a.ctx }

// Done returns a channel that closes after [StageCleanup] and [StageExited]
// callbacks finish, immediately before [App.Run] returns. The same channel is
// available from [New] onward. It stays open if Run has never been called.
// If waiting for managed tasks times out, some tasks may still be running
// when Done closes.
func (a *App) Done() <-chan struct{} { return a.done }

// SetName sets app name.
//
// It must be called before [App.Run].
func (a *App) SetName(name string) {
	if name == "" {
		panic("app: empty name")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetName")
	a.name.Store(name)
}

// SetCommit sets the app commit.
//
// It must be called before [App.Run].
func (a *App) SetCommit(commit string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetCommit")
	a.commit.Store(commit)
}

// SetVersion sets app version.
//
// It must be called before [App.Run].
func (a *App) SetVersion(version string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetVersion")
	a.version.Store(version)
}

// SetBuildTime sets the app build time.
//
// It must be called before [App.Run].
func (a *App) SetBuildTime(builtat time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetBuildTime")
	a.builtat.Store(builtat.Unix())
}

// SetShutdownTimeout sets graceful shutdown timeout.
//
// The timeout is shared by stopping callbacks, module Stop methods, waiting
// for managed background tasks, cleanup callbacks, and exited callbacks. It
// is cooperative: callbacks must observe their context or otherwise return
// promptly. All shutdown callbacks are attempted even after the deadline.
//
// It must be called before [App.Run].
func (a *App) SetShutdownTimeout(timeout time.Duration) {
	if timeout <= 0 {
		panic("app: shutdown timeout must be positive")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetShutdownTimeout")
	a.shutdownTimeout = timeout
}

// SetSignals sets signals that trigger graceful shutdown.
//
// Passing no signals means [App] will not listen to OS signals and will only
// stop on [App.Stop], parent context cancellation, or a managed task failure.
//
// It must be called before [App.Run].
func (a *App) SetSignals(signals ...os.Signal) {
	for _, sig := range signals {
		if sig == nil {
			panic("app: nil signal")
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetSignals")
	a.signals = slices.Clone(signals)
}

// Run starts the app lifecycle and blocks until shutdown,
// which can only be called once.
//
// Startup loads configuration, runs [StageInit], initializes modules, runs
// [StageStart], starts modules, then runs [StageReady]. Shutdown cancels the
// run context, runs [StageStopping], stops initialized modules in reverse order,
// waits for managed background tasks, then runs [StageCleanup] and [StageExited].
// The same shutdown sequence is used if startup fails.
//
// If [App.Run] returns a non-nil error, caller may print it and call [os.Exit]
// with status 1.
func (a *App) Run(ctx context.Context) (err error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cancelRun := func() {
		cancel()
		a.cancelContext()
	}

	signals := a.startRun(runCtx, cancelRun)
	defer cancelRun()
	// A parent can cancel while a startup callback is still blocked. Forward
	// cancellation without waiting for that callback to return.
	stopContextCancel := context.AfterFunc(runCtx, a.cancelContext)
	defer stopContextCancel()
	errCh := a.errCh

	if len(signals) > 0 {
		signalCtx, stopSignal := signal.NotifyContext(runCtx, signals...)
		defer stopSignal()

		// Cancel the context used by startup callbacks and modules as soon as
		// a signal arrives, even if startup is still waiting on a dependency.
		stopCancel := context.AfterFunc(signalCtx, cancelRun)
		defer stopCancel()
	}

	for phase := phaseConfig; phase <= phaseReady; phase++ {
		if err = a.runPhase(runCtx, phase); err != nil {
			break
		}
	}

	if err == nil {
		<-runCtx.Done()
	}

	// A task publishes its error before canceling runCtx. Cancellation may
	// have interrupted startup; preserve the original failure.
	select {
	case backgroundErr := <-errCh:
		err = errors.Join(err, backgroundErr)
	default:
	}

	a.markStopping()
	cancelRun()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancelShutdown()

	err = errors.Join(err, a.shutdown(shutdownCtx))
	// A task may have passed its cancellation check before shutdown and only
	// published its error while shutdown was waiting for it to finish.
	select {
	case backgroundErr := <-errCh:
		err = errors.Join(err, backgroundErr)
	default:
	}
	a.markExited()
	return err
}

// Stop requests [App.Run] to stop.
//
// It is safe to call [App.Stop] multiple times from different goroutines.
// If [App.Run] is not running anymore, [App.Stop] is a no-op.
func (a *App) Stop() {
	a.mu.Lock()
	cancel := a.cancelRun
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// Wait blocks until [App.Run] returns.
//
// If [App.Run] is still running, [App.Wait] blocks until the shutdown lifecycle
// finishes. If [App.Run] has already returned, [App.Wait] returns immediately.
func (a *App) Wait() {
	_ = a.WaitContext(context.Background())
}

// WaitContext blocks until [App.Run] returns or ctx is done.
//
// It returns nil if [App.Run] has exited, or the error from [context.Context.Err]
// if canceled first.
func (a *App) WaitContext(ctx context.Context) error {
	select {
	case <-a.done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *App) startRun(ctx context.Context, cancel context.CancelFunc) []os.Signal {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != stateNew {
		panic("app: Run can only be called once")
	}

	a.state = stateRunning
	a.runCtx = ctx
	a.cancelRun = cancel
	a.errCh = make(chan error, 1)

	sortModules(a.modules)
	for _, m := range a.modules {
		a.registerModuleLocked(m)
	}
	a.registerCallbackLocked(phaseWaitBackground, "", a.waitBackground)

	// Configuration is frozen once Run starts; these slices cannot change.
	return a.signals
}

func (a *App) shutdown(ctx context.Context) error {
	var errs []error
	for phase := phaseStopping; phase < phaseCount; phase++ {
		if err := a.runPhase(ctx, phase); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (a *App) waitBackground(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		return fmt.Errorf("app: wait background tasks: %w", ctx.Err())
	}
}

func (a *App) markStopping() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.state = stateStopping
}

func (a *App) markExited() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.state = stateExited
	a.runCtx = nil
	a.cancelRun = nil
	a.errCh = nil
	close(a.done)
}

func (a *App) mustBeNewLocked(method string) {
	if a.state != stateNew {
		panic(fmt.Sprintf("app: %s cannot be called after Run", method))
	}
}
