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
	phaseStart
	phaseModuleStart
	phaseReady
	phaseStopping
	phaseModuleStop
	phaseWaitBackground
	phaseCleanup
	phaseExited
	phaseCount
)

var lifecyclePhases = [phaseCount]struct {
	stage   Stage
	reverse bool
}{
	phaseInit:       {stage: StageInit},
	phaseStart:      {stage: StageStart},
	phaseReady:      {stage: StageReady},
	phaseStopping:   {stage: StageStopping},
	phaseModuleStop: {reverse: true},
	phaseCleanup:    {stage: StageCleanup, reverse: true},
	phaseExited:     {stage: StageExited},
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

		if err := callback.callback(ctx); err != nil {
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
