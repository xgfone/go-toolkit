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

import "context"

// Stage is a lifecycle callback stage.
type Stage string

const (
	// StageInit runs after configuration is loaded and before [Module.Init].
	// Use it for setup that does not depend on initialized modules.
	StageInit Stage = "init"

	// StageStart runs after all [Module.Init] calls and before [Module.Start].
	// Use it to prepare business data that must be available before services start.
	StageStart Stage = "start"

	// StageReady runs after all [Module.Start] calls. Services such as HTTP servers
	// may already be handling requests, so required request data belongs in
	// [StageInit] or [StageStart].
	StageReady Stage = "ready"

	// StageStopping runs after the run context is canceled and before [Module.Stop].
	//
	// If a [StageStopping] callback returns an error, the error is collected,
	// but the remaining shutdown callbacks will continue to run.
	StageStopping Stage = "stopping"

	// StageCleanup runs after [Module.Stop] and waiting for managed background
	// tasks, and before [StageExited]. If waiting times out, tasks may still
	// be running.
	//
	// Callbacks registered for [StageCleanup] are executed in reverse registration
	// order, making it suitable for releasing resources that were initialized
	// earlier in the app lifecycle.
	//
	// If a [StageCleanup] callback returns an error, the error is collected,
	// but the remaining shutdown callbacks will continue to run.
	StageCleanup Stage = "cleanup"

	// StageExited is the final lifecycle stage.
	//
	// It is triggered after [StageCleanup] has completed and before [App.Run]
	// returns.  Callbacks registered for [StageExited] are intended for fast
	// final notification logic, such as logging, metrics, or status reporting.
	//
	// [StageExited] does not mean the process has already exited. It means the
	// [App] has reached its final lifecycle stage.
	//
	// If a [StageExited] callback returns an error, the error is collected,
	// but the remaining shutdown callbacks will continue to run.
	StageExited Stage = "exited"
)

// On registers a context-only callback on the default app at this stage.
// See [App.On].
func (s Stage) On(name string, callback func(ctx context.Context) error) {
	defaultApp.On(s, name, callback)
}
