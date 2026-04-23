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

package redact

import "fmt"

// Level controls how much information may be exposed by a redacted value.
//
// The levels are ordered from least redaction to most redaction:
//
//   - LevelRaw: for fully trusted internal use, such as debugging, diagnostics,
//     and other program-internal paths where raw data may be shown.
//   - LevelTrusted: for trusted but still bounded internal use, such as admin
//     tools, operations consoles, or protected internal pages, where only the
//     most sensitive parts should be removed.
//   - LevelExternal: for callers outside the current program boundary, such as
//     other services, SDK consumers, or external integrations, where secrets
//     and internal details should be hidden but business-useful information may
//     still be preserved.
//   - LevelPublic: for public or low-trust output, such as content visible to
//     end users or broadly exposed outside the platform, where only the
//     safest information should remain.
type Level uint8

const (
	// LevelRaw is for fully trusted internal use and typically allows raw,
	// unredacted output.
	LevelRaw Level = iota

	// LevelTrusted is for trusted internal consumers that may see more than
	// external callers, but should still avoid the most sensitive details.
	LevelTrusted

	// LevelExternal is for consumers outside the current program boundary and
	// should hide secrets and internal details.
	LevelExternal

	// LevelPublic is for public or lowest-trust output and should expose only
	// the safest information.
	LevelPublic
)

// Valid reports whether the level l is a defined Level value.
func (l Level) Valid() bool {
	return LevelRaw <= l && l <= LevelPublic
}

// String returns the symbolic name of the level l.
func (l Level) String() string {
	switch l {
	case LevelRaw:
		return "Raw"

	case LevelTrusted:
		return "Trusted"

	case LevelExternal:
		return "External"

	case LevelPublic:
		return "Public"

	default:
		return fmt.Sprintf("redact.Level(%d)", l)
	}
}

// Redactor is implemented by values that can produce a redacted view of
// themselves for the given Level.
//
// Implementations are not required to handle every possible Level explicitly.
// If an implementation receives a Level that it does not specifically support,
// it should fall back to another Level that it considers reasonably safe for
// its own use case. That fallback does not need to be the most restrictive
// Level.
type Redactor interface {
	Redact(level Level) any
}

// Redact returns a redacted view of value at the given Level.
//
// Only the top-level value is checked. If value implements Redactor, its
// Redact method is used. Otherwise value is returned unchanged.
func Redact(value any, level Level) any {
	if redactor, ok := value.(Redactor); ok {
		return redactor.Redact(level)
	}
	return value
}
