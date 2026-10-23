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

	shutdownTimeout  time.Duration
	cleanupTimeout   time.Duration
	exitedTimeout    time.Duration
	finalizerTimeout time.Duration

	signals    []os.Signal
	finalizer  *namedCallback
	finalizing bool

	modules   []Module
	callbacks [phaseCount][]namedCallback
	stage     lifecyclePhase // Last public stage; phaseConfig before StageInit.
	state     state

	ctx           context.Context
	cancelContext context.CancelFunc
	runCtx        context.Context
	cancelRun     context.CancelFunc

	groups        map[string]*TaskGroup
	rootTasks     *TaskGroup
	moduleBatches [][]*moduleRun
	taskErrors    []error
	result        error

	done chan struct{}
}

// New creates an [App] with minimal default behavior.
//
// By default, it:
//   - sets version to "0.0.0"
//   - derives name from the first entry of [os.Args] using [filepath.Base]
//     and [strings.TrimSuffix] to remove the ".exe" suffix
//   - uses a minimal [flag] config loader (see [App.SetConfigLoader])
//   - allows 30 seconds for draining, and 5 seconds each for cleanup, exited
//     callbacks, and the optional finalizer
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
	app.SetCleanupTimeout(5 * time.Second)
	app.SetExitedTimeout(5 * time.Second)
	app.SetFinalizerTimeout(5 * time.Second)
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
// Use [App.Done] or [App.Wait] to wait for shutdown to finish.
//
// This context carries no values or deadline from the parent passed to [App.Run];
// startup callbacks and App.Go tasks receive its derived run context instead.
// Module TaskGroups retain values without inheriting its cancellation/deadline.
// Shutdown phases retain parent values but have independent deadlines.
func (a *App) Context() context.Context { return a.ctx }

// Done returns a channel that closes immediately before [App.Run] returns.
// Normally this follows Cleanup, Exited, and the optional finalizer. On
// [ErrShutdownIncomplete], these phases are skipped to retain dependencies
// still used by live tasks. Done then does not mean those tasks have exited.
// The channel is available from [New] onward and stays open until Run finishes.
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

// SetShutdownTimeout sets the budget for stopping callbacks, module Stop and
// Wait methods, and waiting for managed background tasks. The default is 30
// seconds. Cleanup, Exited, and the finalizer have separate budgets.
//
// Timeouts are cooperative: callbacks must observe their context or return
// promptly. If users of resources cannot be joined, Run returns
// [ErrShutdownIncomplete] without closing those resources or the logger.
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

// Run starts the app lifecycle and blocks until shutdown, which can only be
// called once.
//
// Startup loads configuration, runs StageInit, all Module.Init and Prepare
// methods, StageStart, all Module.Start methods, then StageReady. Dependencies
// precede their users. On shutdown, batches stop in reverse dependency order;
// app-level tasks join the outermost batch. Each batch initiates every Stop and
// cancels its task groups, then joins every Wait and group before stopping
// dependencies. Only successful joins permit Close, Cleanup, Exited, and the
// finalizer. Startup failures use the same rollback path for initialized modules
// and all admitted tasks. Run's final error is also returned by App.Wait.
//
// ctx controls the running lifetime, not the shutdown deadline. Cancellation
// starts shutdown; its independent phase budgets retain ctx's values without
// inheriting its cancellation or deadline. These budgets are cooperative.
//
// If [App.Run] returns a non-nil error, caller may print it and call [os.Exit]
// with status 1.
func (a *App) Run(ctx context.Context) (err error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cancelRun := func() { cancel(); a.cancelContext() }
	signals, planErr := a.startRun(runCtx, cancelRun)
	defer cancelRun()

	stopContextCancel := context.AfterFunc(runCtx, a.cancelContext)
	defer stopContextCancel()

	if len(signals) > 0 {
		signalCtx, stopSignal := signal.NotifyContext(runCtx, signals...)
		defer stopSignal()

		stopCancel := context.AfterFunc(signalCtx, cancelRun)
		defer stopCancel()
	}

	err = planErr
	if err == nil {
		for phase := phaseConfig; phase <= phaseReady; phase++ {
			if err = a.runPhase(runCtx, phase); err != nil {
				break
			}
		}
	}

	if err == nil {
		<-runCtx.Done()
	}

	a.markStopping()
	cancelRun()
	err = errors.Join(err, a.shutdown(context.WithoutCancel(ctx)))
	return a.markExited(err)
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

// Wait waits for Run's final result or ctx cancellation. Canceling a wait does
// not stop the app or alter its shutdown budgets. Multiple and repeated waiters
// receive the same final result, including ErrShutdownIncomplete. A completed
// run wins over an already canceled ctx. Before Run, Wait waits for a future
// run. Lifecycle methods and callbacks must not call App.Wait.
func (a *App) Wait(ctx context.Context) error {
	select {
	case <-a.done:
		return a.result
	default:
	}

	select {
	case <-a.done:
		return a.result

	case <-ctx.Done():
		select {
		case <-a.done:
			return a.result

		default:
			return ctx.Err()
		}
	}
}

func (a *App) startRun(ctx context.Context, cancel context.CancelFunc) ([]os.Signal, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != stateNew {
		panic("app: Run can only be called once")
	}

	a.state = stateRunning
	a.runCtx = ctx
	a.cancelRun = cancel
	a.rootTasks = &TaskGroup{
		app:    a,
		ctx:    ctx,
		cancel: func() {},
		done:   make(chan struct{}),
	}

	return a.signals, saferun(ctx, a.prepareModulesLocked)
}

func (a *App) waitBackground(ctx context.Context) error {
	if err := a.rootTasks.Wait(ctx); err != nil {
		return fmt.Errorf("app: wait background tasks: %w", err)
	}
	return nil
}

func (a *App) markStopping() {
	a.mu.Lock()
	a.state = stateStopping
	a.rootTasks.Stop()
	a.mu.Unlock()
}

func (a *App) markExited(err error) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.state = stateExited
	a.runCtx = nil
	a.cancelRun = nil
	a.result = errors.Join(append([]error{err}, a.taskErrors...)...)
	close(a.done)
	return a.result
}

func (a *App) mustBeNewLocked(method string) {
	if a.state != stateNew {
		panic(fmt.Sprintf("app: %s cannot be called after Run", method))
	}
}
