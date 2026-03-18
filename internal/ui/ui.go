package ui

import (
	"fmt"
	"os"
)

// ANSI colour codes – fall back gracefully on non-TTY environments.
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	red    = "\033[31m"
	dim    = "\033[2m"
)

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func colorize(s, code string) string {
	if !isTTY() {
		return s
	}
	return code + s + reset
}

// Success prints a green ✔ message.
func Success(msg string) {
	fmt.Println(colorize("✔ "+msg, green))
}

// Info prints a cyan ℹ message.
func Info(msg string) {
	fmt.Println(colorize("ℹ "+msg, cyan))
}

// Warn prints a yellow ⚠ message.
func Warn(msg string) {
	fmt.Println(colorize("⚠ "+msg, yellow))
}

// Error prints a red ✖ message to stderr.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, colorize("✖ "+msg, red))
}

// Errorf prints a formatted red ✖ message to stderr.
func Errorf(format string, args ...any) {
	Error(fmt.Sprintf(format, args...))
}

// UserRow prints a single user row in the list.
func UserRow(name, email string, active bool) {
	marker := "  "
	nameStr := name
	if active {
		marker = colorize("▶ ", green)
		nameStr = colorize(bold+name, green)
	}
	emailStr := colorize(email, dim)
	fmt.Printf("%s%-20s %s\n", marker, nameStr, emailStr)
}

// Header prints a bold section header.
func Header(msg string) {
	fmt.Println(colorize(bold+msg, cyan))
}

// Divider prints a thin separator line.
func Divider() {
	fmt.Println(colorize("─────────────────────────────────────────", dim))
}
