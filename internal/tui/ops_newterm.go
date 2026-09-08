package tui

import (
	"os"
	"strings"
)

// openNewTerminalWindow spawns a brand-new, detached terminal-emulator window
// running an isolated shell for the given identity — equivalent to running
// `git-user shell <name>` there by hand. Unlike openIdentityShellCmd (which
// suspends and takes over the TUI's own terminal), this never touches the
// TUI's terminal at all: the TUI keeps running in its window while the new
// one opens alongside it. That's what actually lets two identities be used
// at once, side by side — running an isolated shell in the SAME window one
// at a time is still only one identity active at any given moment.
//
// spawnIdentityTerminal is implemented per-OS (ops_newterm_linux.go,
// ops_newterm_darwin.go, ops_newterm_windows.go). If no supported terminal
// emulator can be found — a headless session, an unrecognized terminal, an
// unsupported OS — it returns an error so the caller can fall back to
// showing the manual `git-user shell <name>` command instead.
func openNewTerminalWindow(identityName string) error {
	exePath, err := os.Executable()
	if err != nil || exePath == "" {
		exePath = "git-user"
	}
	return spawnIdentityTerminal(exePath, identityName)
}

// shellJoin quotes and joins args into a single POSIX shell command line, for
// launchers (tilix, macOS Terminal.app) whose command-execution entry point
// re-parses one string rather than accepting argv directly.
func shellJoin(args ...string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(quoted, " ")
}

// appleScriptQuote escapes a string for embedding inside an AppleScript
// double-quoted string literal.
func appleScriptQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
