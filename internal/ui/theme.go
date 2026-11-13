// Copyright 2025 Google LLC
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

// Package ui provides semantic terminal formatting and error hint helpers.
package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	// Accent is used for landmarks like Headers, Group Titles, Section Labels.
	Accent = color.New(color.FgCyan, color.Bold).SprintFunc()

	// Command is used for scan targets like Command names, Flags.
	Command = color.New(color.FgHiWhite, color.Bold).SprintFunc()

	// Pass indicates success states or completed tasks.
	Pass = color.New(color.FgGreen).SprintFunc()

	// Warn indicates transient states, warnings, or active tasks.
	Warn = color.New(color.FgYellow).SprintFunc()

	// Fail indicates errors or rejected states.
	Fail = color.New(color.FgRed).SprintFunc()

	// Muted is used for de-emphasis like metadata, types, defaults, previews.
	Muted = color.New(color.FgHiBlack).SprintFunc()

	// ID is used for unique identifiers.
	ID = color.New(color.FgHiCyan).SprintFunc()
)

// PrintSuccess prints a formatted success message to stdout.
func PrintSuccess(msg string, args ...any) {
	fmt.Println(Pass("✔ " + fmt.Sprintf(msg, args...)))
}

// ErrorWithHint returns an error wrapped with a formatted actionable hint.
func ErrorWithHint(err error, hint string) error {
	return fmt.Errorf("%w\n\n%s %s", err, Warn("Hint:"), hint)
}

// HandleError prints an error to stderr, honoring any attached hint, and exits.
func HandleError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%s %v\n", Fail("Error:"), err)
	os.Exit(1)
}
