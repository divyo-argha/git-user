package ui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
func UserRow(name, email, sshKey string, active bool) {
	marker := "  "
	nameStr := name
	if active {
		marker = colorize("▶ ", green)
		nameStr = colorize(bold+name, green)
	}
	emailStr := colorize(email, dim)

	sshStr := ""
	if sshKey != "" {
		sshStr = colorize(fmt.Sprintf(" [key: %s]", sshKey), dim)
	}

	fmt.Printf("%s%-20s %s%s\n", marker, nameStr, emailStr, sshStr)
}

// UserDetails prints the details of a single user.
func UserDetails(name, email, sshKey string) {
	fmt.Printf("  Name  : %s\n", name)
	fmt.Printf("  Email : %s\n", email)
	if sshKey != "" {
		fmt.Printf("  Key   : %s\n", sshKey)
	}
}

// Header prints a bold section header.
func Header(msg string) {
	fmt.Println(colorize(bold+msg, cyan))
}

// Divider prints a thin separator line.
func Divider() {
	fmt.Println(colorize("─────────────────────────────────────────", dim))
}

// RawMode toggles terminal raw mode using 'stty'.
func RawMode(on bool) error {
	if !isTTY() {
		return nil
	}
	arg := "raw"
	if !on {
		arg = "-raw"
	}
	cmd := exec.Command("stty", arg, "-echo")
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// Prompt asks the user for text input.
func Prompt(label string) (string, error) {
	fmt.Printf("%s %s ", colorize("?", cyan), label)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// Select displays a list of options and returns the index of the chosen one.
func Select(label string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options provided")
	}

	selected := 0
	if err := RawMode(true); err != nil {
		return -1, err
	}
	defer RawMode(false)

	fmt.Printf("%s %s %s\n", colorize("?", cyan), label, colorize("(Use arrow keys)", dim))

	for {
		// Print options
		for i, opt := range options {
			if i == selected {
				fmt.Printf("%s %s\n", colorize(">", cyan), colorize(opt, bold+cyan))
			} else {
				fmt.Printf("  %s\n", opt)
			}
		}

		// Read input
		var b [3]byte
		n, err := os.Stdin.Read(b[:])
		if err != nil {
			return -1, err
		}

		// Move cursor back up
		fmt.Printf("\033[%dA", len(options))

		if n == 1 {
			if b[0] == '\r' || b[0] == '\n' {
				// Clear the menu before returning
				for i := 0; i < len(options); i++ {
					fmt.Printf("\033[K\n")
				}
				fmt.Printf("\033[%dA", len(options))
				return selected, nil
			}
			if b[0] == 3 { // Ctrl+C
				return -1, fmt.Errorf("interrupted")
			}
		} else if n == 3 && b[0] == 27 && b[1] == 91 { // Escape sequence
			switch b[2] {
			case 65: // Up
				if selected > 0 {
					selected--
				}
			case 66: // Down
				if selected < len(options)-1 {
					selected++
				}
			}
		}
	}
}
