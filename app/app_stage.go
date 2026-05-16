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

// Stage is a lifecycle hook stage.
type Stage string

const (
	StageInit  Stage = "init"
	StageStart Stage = "start"
	StageReady Stage = "ready"

	// If a StageStopping hook returns an error, the error is collected, but the
	// remaining cleanup hooks will continue to run.
	StageStopping Stage = "stopping"

	// StageCleanup is triggered after StageStopping and before StageExited.
	//
	// Hooks registered for StageCleanup are executed in reverse registration
	// order, making it suitable for releasing resources that were initialized
	// earlier in the app lifecycle.
	//
	// If a StageCleanup hook returns an error, the error is collected, but the
	// remaining cleanup hooks will continue to run.
	StageCleanup Stage = "cleanup"

	// StageExited is the final lifecycle stage.
	//
	// It is triggered after StageCleanup has completed and before Run returns.
	// Hooks registered for StageExited are intended for fast final notification
	// logic, such as logging, metrics, or status reporting.
	//
	// StageExited does not mean the process has already exited. It means the App
	// has reached its final lifecycle stage.
	//
	// If a StageExited hook returns an error, the error is collected, but the
	// remaining cleanup hooks will continue to run.
	StageExited Stage = "exited"
)

// On registers a hook function into DefaultApp to be executed at the given stage.
func (s Stage) On(hook Hook) {
	DefaultApp.On(s, hook)
}

// OnNamed registers a named hook function into DefaultApp to be executed at the given stage.
func (s Stage) OnNamed(name string, hook Hook) {
	DefaultApp.OnNamed(s, name, hook)
}

func (a *App) canRegisterHookLocked(stage Stage) bool {
	if a.state == stateNew {
		return true
	}

	if a.state == stateExited {
		return false
	}

	return stageOrder(stage) > stageOrder(a.stage)
}

func validStage(stage Stage) bool {
	return stageOrder(stage) > 0
}

func stageOrder(stage Stage) int {
	switch stage {
	case "":
		return 0

	case StageInit:
		return 10

	case StageStart:
		return 20

	case StageReady:
		return 30

	case StageStopping:
		return 40

	case StageCleanup:
		return 50

	case StageExited:
		return 60

	default:
		return -1
	}
}
