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

// Package app provides the management of the application information.
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

	"github.com/xgfone/go-toolkit/timex"
)

// DefaultApp is the package-level default App instance.
var DefaultApp = New()

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				DefaultApp.SetCommit(setting.Value[:7])
				break
			}
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

	configLoader    Hook
	shutdownTimeout time.Duration
	signals         []os.Signal

	modules []Module
	hooks   map[Stage][]namedHook
	stage   Stage
	state   state

	runCtx    context.Context
	cancelRun context.CancelFunc

	wg    sync.WaitGroup
	errCh chan error
	done  chan struct{}
}

// New creates an App with minimal default behavior.
//
// By default, it:
//   - sets version to "0.0.0"
//   - sets name to strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
//   - uses a minimal flag-based ConfigLoader
//   - uses 30 seconds as shutdown timeout
//   - listens to SIGINT, SIGTERM
func New() *App {
	app := &App{
		state: stateNew,
		hooks: make(map[Stage][]namedHook),
		done:  make(chan struct{}),
	}
	app.SetName(strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe"))
	app.SetConfigLoader(defaultFlagConfigLoader)
	app.SetShutdownTimeout(30 * time.Second)
	app.SetSignals(os.Interrupt, syscall.SIGTERM)
	app.SetVersion("0.0.0")
	return app
}

// Run starts the default app, see App.Run.
func Run(ctx context.Context) error {
	return DefaultApp.Run(ctx)
}

// Name is a convenience function that returns the default app name.
func Name() string {
	return DefaultApp.Name()
}

// Version is a convenience function that returns the default app version.
func Version() string {
	return DefaultApp.Version()
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

// BuildTime returns the time when built the app.
func (a *App) BuildTime() time.Time {
	return timex.Unix(a.builtat.Load(), 0)
}

// SetName sets app name.
//
// It must be called before Run.
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
// It must be called before Run.
func (a *App) SetCommit(commit string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetCommit")
	a.commit.Store(commit)
}

// SetVersion sets app version.
//
// It must be called before Run.
func (a *App) SetVersion(version string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetVersion")
	a.version.Store(version)
}

// SetBuildTime sets the app build time.
//
// It must be called before Run.
func (a *App) SetBuildTime(builtat time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.mustBeNewLocked("SetBuildTime")
	a.builtat.Store(builtat.Unix())
}

// SetShutdownTimeout sets graceful shutdown timeout.
//
// It must be called before Run.
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
// Passing no signals means App will not listen to OS signals and will only stop
// when parent ctx is canceled or a background task returns error.
//
// It must be called before Run.
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
// If Run returns a non-nil error, caller may print it and os.Exit(1).
func (a *App) Run(ctx context.Context) (err error) {
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	modules, loader, signals := a.startRun(runCtx, cancelRun)

	signalCtx, stopSignal := signal.NotifyContext(runCtx, signals...)
	defer stopSignal()

	initialized := make([]Module, 0, len(modules))
	shutdownDone := false

	doShutdown := func() error {
		if shutdownDone {
			return nil
		}

		shutdownDone = true

		a.markStopping()
		cancelRun()

		shutdownCtx, cancelShutdown := a.newShutdownContext()
		defer cancelShutdown()

		var shutdownErr error

		if e := a.runHooks(shutdownCtx, StageStopping); e != nil {
			shutdownErr = errors.Join(shutdownErr, e)
		}

		if e := a.shutdown(shutdownCtx, initialized); e != nil {
			shutdownErr = errors.Join(shutdownErr, e)
		}

		if e := a.runHooks(context.Background(), StageCleanup); e != nil {
			shutdownErr = errors.Join(shutdownErr, e)
		}

		if e := a.runHooks(context.Background(), StageExited); e != nil {
			shutdownErr = errors.Join(shutdownErr, e)
		}

		a.markExited()

		return shutdownErr
	}

	defer func() {
		if err != nil {
			err = errors.Join(err, doShutdown())
		}
	}()

	// 1. Config
	if err = loader(runCtx, a); err != nil {
		return err
	}

	// 2. Init hooks
	if err = a.runHooks(runCtx, StageInit); err != nil {
		return err
	}

	// 3. Module Init
	for _, m := range modules {
		if e := m.Init(runCtx, a); e != nil {
			err = fmt.Errorf("app: init module %q: %w", m.Name(), e)
			return err
		}

		initialized = append(initialized, m)
	}

	// 4. Start hooks
	if err = a.runHooks(runCtx, StageStart); err != nil {
		return err
	}

	// 5. Module Start
	for _, m := range initialized {
		if e := m.Start(runCtx, a); e != nil {
			err = fmt.Errorf("app: start module %q: %w", m.Name(), e)
			return err
		}
	}

	// 6. Ready hooks
	if err = a.runHooks(runCtx, StageReady); err != nil {
		return err
	}

	// 7. Running
	select {
	case <-signalCtx.Done():
		// Normal shutdown path.

	case e := <-a.errCh:
		err = errors.Join(err, e)
	}

	// 8. Shutdown
	err = errors.Join(err, doShutdown())
	return err
}

// Stop requests Run to stop.
//
// It is safe to call Stop multiple times from different goroutines.
// If Run is not running anymore, Stop is a no-op.
func (a *App) Stop() {
	a.mu.Lock()
	cancel := a.cancelRun
	a.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// Wait blocks until Run returns.
//
// If Run is still running, Wait blocks until the shutdown lifecycle finishes.
// If Run has already returned, Wait returns immediately.
func (a *App) Wait() {
	_ = a.WaitContext(context.Background())
}

// WaitContext blocks until Run returns or ctx is done.
//
// It returns nil if Run has exited, or ctx.Err() if canceled first.
func (a *App) WaitContext(ctx context.Context) error {
	select {
	case <-a.done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *App) startRun(ctx context.Context, cancel context.CancelFunc) ([]Module, Hook, []os.Signal) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != stateNew {
		panic("app: Run can only be called once")
	}

	a.state = stateRunning
	a.runCtx = ctx
	a.cancelRun = cancel
	a.errCh = make(chan error, 1)

	loader := a.configLoader
	signals := slices.Clone(a.signals)
	modules := slices.Clone(a.modules)
	sortModules(modules)

	return modules, loader, signals
}

func (a *App) shutdown(ctx context.Context, initialized []Module) error {
	var errs []error

	// Stop modules in reverse order.
	for i := len(initialized) - 1; i >= 0; i-- {
		m := initialized[i]

		if err := m.Stop(ctx, a); err != nil {
			errs = append(errs, fmt.Errorf("app: stop module %q: %w", m.Name(), err))
		}
	}

	if err := a.waitBackground(ctx); err != nil {
		errs = append(errs, err)
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

func (a *App) newShutdownContext() (context.Context, context.CancelFunc) {
	a.mu.Lock()
	timeout := a.shutdownTimeout
	a.mu.Unlock()

	return context.WithTimeout(context.Background(), timeout)
}

func (a *App) markStopping() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state == stateRunning {
		a.state = stateStopping
	}
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
