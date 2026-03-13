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

// PrintSuccess prints a success message.
func PrintSuccess(msg string, args ...interface{}) {
	fmt.Println(Pass("✔ " + fmt.Sprintf(msg, args...)))
}

// ErrorWithHint returns an error wrapped with a formatted Hint.
func ErrorWithHint(err error, hint string) error {
	return fmt.Errorf("%w\n\n%s %s", err, Warn("Hint:"), hint)
}

// HandleError prints an error, honoring the Hint if present, and exits.
func HandleError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%s %v\n", Fail("Error:"), err)
	os.Exit(1)
}
