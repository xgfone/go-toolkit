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

package app

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"
)

// Private phases preserve the boundaries between public callbacks and module
// methods while letting them share the same registry and executor.
type lifecyclePhase int

const (
	phaseConfig lifecyclePhase = iota
	phaseInit
	phaseModuleInit
	phaseModulePrepare
	phaseStart
	phaseModuleStart
	phaseReady
	phaseStopping
	phaseModuleClose
	phaseCleanup
	phaseExited
	phaseCount
)

var lifecyclePhases = [phaseCount]struct {
	stage   Stage
	reverse bool
}{
	phaseInit:        {stage: StageInit},
	phaseStart:       {stage: StageStart},
	phaseReady:       {stage: StageReady},
	phaseStopping:    {stage: StageStopping},
	phaseModuleClose: {reverse: true},
	phaseCleanup:     {stage: StageCleanup, reverse: true},
	phaseExited:      {stage: StageExited},
}

// ErrShutdownIncomplete means a module or managed task could not be joined.
// Module.Close, Cleanup, Exited, and the finalizer were skipped so that live
// work retains its resources. A failed user retains its transitive providers;
// independent branches are still stopped. Run does not forcibly terminate
// goroutines or retry finalization in the background. The caller must arrange
// eventual termination or release retained resources after their users have exited.
var ErrShutdownIncomplete = errors.New("app: shutdown incomplete; resources retained")

func (a *App) shutdown(base context.Context) error {
	drainCtx, cancelDrain := context.WithTimeout(base, a.shutdownTimeout)
	defer cancelDrain()

	var errs []error
	if err := a.runPhase(drainCtx, phaseStopping); err != nil {
		errs = append(errs, err)
	}

	if len(a.moduleBatches) == 0 {
		if err := a.waitBackground(drainCtx); err != nil {
			return errors.Join(append(errs, ErrShutdownIncomplete, err)...)
		}
	}

	// A failed user pins only its transitive providers. Independent branches
	// must still receive Stop, even if they occur in a later startup layer.
	byName := make(map[string]*moduleRun)
	for _, batch := range a.moduleBatches {
		for _, r := range batch {
			byName[r.name] = r
		}
	}

	retained := make(map[*moduleRun]bool)
	var retainProviders func(*moduleRun)
	retainProviders = func(r *moduleRun) {
		for _, name := range r.dependencies {
			provider := byName[name]
			if !retained[provider] {
				retained[provider] = true
				retainProviders(provider)
			}
		}
	}

	incomplete := false
	firstBatch := true
	for _, batch := range slices.Backward(a.moduleBatches) {
		var stopping []*moduleRun
		for _, r := range batch {
			if !retained[r] {
				stopping = append(stopping, r)
			}
		}

		for _, r := range slices.Backward(stopping) {
			if r.initialized {
				err := a.callModule(drainCtx, r, "stop", r.module.Stop)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}

		// Notify every eligible member before joining any member of this batch.
		for _, r := range stopping {
			r.tasks.Stop()
		}

		for _, r := range slices.Backward(stopping) {
			joined := true
			if r.initialized {
				err := a.callModule(drainCtx, r, "wait", r.module.Wait)
				if err != nil {
					errs = append(errs, err)
					joined = false
				}
			}

			if err := r.tasks.Wait(drainCtx); err != nil {
				err = fmt.Errorf("app: wait module %q tasks: %w", r.name, err)
				errs = append(errs, err)
				joined = false
			}

			if !joined {
				incomplete = true
				retainProviders(r)
			}
		}

		if firstBatch {
			firstBatch = false
			if err := a.waitBackground(drainCtx); err != nil {
				// App-level tasks have no dependency declarations and may use
				// any provider, so conservatively retain all remaining batches.
				return errors.Join(append(errs, ErrShutdownIncomplete, err)...)
			}
		}
	}
	if incomplete {
		return errors.Join(append(errs, ErrShutdownIncomplete)...)
	}
	cancelDrain()

	cleanupCtx, cancelCleanup := context.WithTimeout(base, a.cleanupTimeout)
	defer cancelCleanup()
	for _, phase := range []lifecyclePhase{phaseModuleClose, phaseCleanup} {
		if err := a.runPhase(cleanupCtx, phase); err != nil {
			errs = append(errs, err)
		}
	}
	cancelCleanup()

	exitedCtx, cancelExited := context.WithTimeout(base, a.exitedTimeout)
	defer cancelExited()
	if err := a.runPhase(exitedCtx, phaseExited); err != nil {
		errs = append(errs, err)
	}
	cancelExited()

	a.mu.Lock()
	a.finalizing = true
	finalizer := a.finalizer
	a.mu.Unlock()
	if finalizer != nil {
		ctx, cancel := context.WithTimeout(base, a.finalizerTimeout)
		defer cancel()
		if err := saferun(ctx, finalizer.callback); err != nil {
			err = fmt.Errorf("app: finalizer %q: %w", finalizer.name, err)
			errs = append(errs, err)
		}
		cancel()
	}

	return errors.Join(errs...)
}

func phaseForStage(stage Stage) lifecyclePhase {
	for phase, spec := range lifecyclePhases {
		if spec.stage != "" && spec.stage == stage {
			return lifecyclePhase(phase)
		}
	}
	return phaseCount
}

func (a *App) registerCallbackLocked(
	phase lifecyclePhase,
	name string,
	callback func(ctx context.Context) error,
) {
	a.callbacks[phase] = append(a.callbacks[phase], namedCallback{
		name:     name,
		callback: callback,
	})
}

func (a *App) runPhase(ctx context.Context, phase lifecyclePhase) error {
	starting := phase < phaseStopping
	if starting && ctx.Err() != nil {
		return ctx.Err()
	}

	spec := lifecyclePhases[phase]
	a.mu.Lock()
	if spec.stage != "" {
		a.stage = phase
	}
	// Entering a stage prevents further registrations for it.
	callbacks := a.callbacks[phase]
	a.mu.Unlock()

	var seq iter.Seq2[int, namedCallback]
	if spec.reverse {
		seq = slices.Backward(callbacks)
	} else {
		seq = slices.All(callbacks)
	}

	var errs []error
	for i, callback := range seq {
		if starting && ctx.Err() != nil {
			return ctx.Err()
		}

		if err := saferun(ctx, callback.callback); err != nil {
			if spec.stage != "" {
				err = fmt.Errorf("app: callback %s: %w",
					callbackLabel(spec.stage, callback.name, i), err)
			}
			if starting {
				return err
			}

			// Shutdown must attempt every callback, including after the deadline.
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
