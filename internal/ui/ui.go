package ui

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/logo"
)

// ErrNotInteractive is returned by Prompt/Select when stdout is not a
// terminal (e.g. a script or CI job piping output). Without this check,
// each would either block waiting on an invisible prompt or launch a
// raw-mode Bubble Tea program that renders garbled escape sequences into
// the pipe instead of failing clearly.
var ErrNotInteractive = errors.New("this command requires an interactive terminal — pass the needed value as a flag instead")

// ── Tokyo Night Palette ───────────────────────────────────────────────────────
// Single source of truth — matches internal/tui/theme/theme.go exactly.
var (
	colPrimary = lipgloss.Color("#7AA2F7") // Soft Blue — info, headers
	colSecond  = lipgloss.Color("#9ECE6A") // Emerald — success, active
	colAccent  = lipgloss.Color("#BB9AF7") // Soft Purple — prompts, accents
	colDanger  = lipgloss.Color("#F7768E") // Rose — errors
	colWarning = lipgloss.Color("#E0AF68") // Amber — warnings
	colMuted   = lipgloss.Color("#565F89") // Deep Gray — dim, separators
	colText    = lipgloss.Color("#C0CAF5") // Ice Blue-White — primary text
	colBg      = lipgloss.Color("#1F2335") // Card background

	// ── Component styles ─────────────────────────────────────────────────────

	styleSuccess = lipgloss.NewStyle().Foreground(colSecond).Bold(true)
	styleInfo    = lipgloss.NewStyle().Foreground(colPrimary)
	styleWarn    = lipgloss.NewStyle().Foreground(colWarning)
	styleError   = lipgloss.NewStyle().Foreground(colDanger).Bold(true)
	styleDim     = lipgloss.NewStyle().Foreground(colMuted)
	styleText    = lipgloss.NewStyle().Foreground(colText)
	styleAccent  = lipgloss.NewStyle().Foreground(colAccent).Bold(true)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colPrimary).
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colAccent).
			MarginBottom(1)

	styleBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(colBg).
			Background(colAccent).
			Padding(0, 2).
			MarginBottom(1).
			MarginTop(1)

	styleCardActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colSecond).
			Padding(0, 2).
			MarginBottom(1).
			Width(60)

	styleCardInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colMuted).
				Padding(0, 2).
				MarginBottom(1).
				Width(60)

	styleActiveBadge = lipgloss.NewStyle().
				Foreground(colBg).
				Background(colSecond).
				Padding(0, 1).
				Bold(true)

	styleMenuSelected = lipgloss.NewStyle().
				Foreground(colAccent).
				Bold(true)

	// Mock function hooks for unit tests
	PromptFn  func(label string) (string, error)
	SelectFn  func(label string, options []string) (int, error)
	ConfirmFn func(question string, defaultYes bool) bool
	IsTTYFn   func() bool
)

// ── TTY Detection ─────────────────────────────────────────────────────────────

// IsTTY returns true if stdout is a character device (terminal).
func IsTTY() bool {
	if IsTTYFn != nil {
		return IsTTYFn()
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// IsPlainOutput reports whether styled/banner output should be suppressed.
// It returns true when stdout is not a terminal (pipes, CI) or the caller
// passes an explicit --plain flag.
func IsPlainOutput(args []string) bool {
	if IsTTYFn != nil {
		return false
	}
	for _, a := range args {
		if a == "--plain" {
			return true
		}
	}
	return !IsTTY()
}

// IsJSONOutput reports whether the caller requested machine-readable JSON
// via an explicit --json flag.
func IsJSONOutput(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

// ── Logo ──────────────────────────────────────────────────────────────────────

// PrintLogo prints the git-user design logo to stdout.
func PrintLogo() {
	lines := logo.GetTrimmedLogo()
	fmt.Println(strings.Join(lines, "\n"))
}

// PrintBanner prints the git-user design logo with the version line to stdout, matching the TUI banner.
func PrintBanner(ver string) {
	lines := logo.GetTrimmedLogo()
	fmt.Println(strings.Join(lines, "\n"))
	if ver != "" {
		fmt.Printf("  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99")).Render(fmt.Sprintf("Version %s", ver)))
	}
}

// PrintUpdateSuccess renders the Unicode logo and an aesthetic version transition card.
func PrintUpdateSuccess(oldVer, newVer string, verified bool) {
	fmt.Println()
	PrintBanner(newVer)
	fmt.Println()

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colSecond).Bold(true).Render("✨ Successfully updated git-user!"))
	sb.WriteString("\n\n   ")
	sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render(oldVer))
	sb.WriteString(lipgloss.NewStyle().Foreground(colPrimary).Bold(true).Render(" ──▶ "))
	sb.WriteString(lipgloss.NewStyle().Foreground(colSecond).Bold(true).Render(newVer))
	if verified {
		sb.WriteString(" " + lipgloss.NewStyle().Foreground(colSecond).Render("(verified)"))
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colSecond).
		Padding(1, 3).
		Render(sb.String())

	fmt.Println(card)
	fmt.Println(lipgloss.NewStyle().Foreground(colMuted).Render("  Run 'git-user' to launch the interactive dashboard."))
	fmt.Println()
}

// PrintUpdateCurrent renders the Unicode logo and an up-to-date card.
func PrintUpdateCurrent(ver string) {
	fmt.Println()
	PrintBanner(ver)
	fmt.Println()

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colSecond).Bold(true).Render("✨ git-user is already up to date!"))
	sb.WriteString("\n\n   ")
	sb.WriteString(lipgloss.NewStyle().Foreground(colSecond).Bold(true).Render(ver))
	sb.WriteString(" " + lipgloss.NewStyle().Foreground(colMuted).Render("(latest release)"))

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colSecond).
		Padding(1, 3).
		Render(sb.String())

	fmt.Println(card)
	fmt.Println(lipgloss.NewStyle().Foreground(colMuted).Render("  Run 'git-user' to launch the interactive dashboard."))
	fmt.Println()
}

// ── Core Output ───────────────────────────────────────────────────────────────

// Success prints a green ✔ message.
func Success(msg string) {
	fmt.Println(styleSuccess.Render("✔ " + msg))
}

// Successf prints a formatted green ✔ message.
func Successf(format string, args ...any) {
	Success(fmt.Sprintf(format, args...))
}

// Info prints a soft-blue ℹ message.
func Info(msg string) {
	fmt.Println(styleInfo.Render("ℹ " + msg))
}

// Warn prints an amber ⚠ message.
func Warn(msg string) {
	fmt.Println(styleWarn.Render("⚠ " + msg))
}

// Error prints a rose ✖ message to stderr.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, styleError.Render("✖ "+msg))
}

// Errorf prints a formatted rose ✖ message to stderr.
func Errorf(format string, args ...any) {
	Error(fmt.Sprintf(format, args...))
}

// StyleDim returns the muted dim style.
func StyleDim() lipgloss.Style { return styleDim }

// StyleSuccess returns the emerald success style.
func StyleSuccess() lipgloss.Style { return styleSuccess }

// ── Layout Helpers ────────────────────────────────────────────────────────────

// Header prints a bold section header with a rounded accent border.
func Header(msg string) {
	fmt.Println(styleHeader.Render(strings.ToUpper(msg)))
}

// Banner prints a full-width accent-background banner.
func Banner(msg string) {
	fmt.Println(styleBanner.Render("  " + strings.ToUpper(msg) + "  "))
}

// Divider prints a thin muted separator line.
func Divider() {
	fmt.Println(styleDim.Render("─────────────────────────────────────────────────────────────────────────────"))
}

// ── Identity Cards ────────────────────────────────────────────────────────────

// UserRow prints a single identity card.
func UserRow(name, email, sshKey string, active bool) {
	badge := ""
	cardStyle := styleCardInactive
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(colText)

	if active {
		badge = styleActiveBadge.Render(" ACTIVE ") + "  "
		cardStyle = styleCardActive
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(colSecond)
	}

	content := fmt.Sprintf("%s%s\n%s",
		badge,
		nameStyle.Render(name),
		styleDim.Render(email),
	)

	if sshKey != "" {
		content += "\n" + styleDim.Render("key: "+sshKey)
	}

	fmt.Println(cardStyle.Render(content))
}

// UserDetails prints the details of a single user.
func UserDetails(name, email, sshKey string) {
	label := lipgloss.NewStyle().Foreground(colPrimary).Bold(true)
	fmt.Printf("  %-10s  %s\n", label.Render("Name  :"), name)
	fmt.Printf("  %-10s  %s\n", label.Render("Email :"), styleDim.Render(email))
	if sshKey != "" {
		fmt.Printf("  %-10s  %s\n", label.Render("Key   :"), styleDim.Render(sshKey))
	}
}

// ── Prompt ────────────────────────────────────────────────────────────────────

// RawMode is a no-op — managed by Bubble Tea.
func RawMode(on bool) error { return nil }

// Prompt asks the user for text input.
func Prompt(label string) (string, error) {
	if PromptFn != nil {
		return PromptFn(label)
	}
	if !IsTTY() {
		return "", ErrNotInteractive
	}
	fmt.Printf("%s %s ", styleAccent.Render("?"), styleText.Bold(true).Render(label))
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// ── Select / Confirm ──────────────────────────────────────────────────────────

// SelectModel is the Bubble Tea model for the selection menu.
type SelectModel struct {
	label    string
	options  []string
	cursor   int
	chosen   int
	canceled bool
}

func (m SelectModel) Init() tea.Cmd { return nil }

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.cursor
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectModel) View() string {
	s := strings.Builder{}
	s.WriteString("\n")
	s.WriteString(styleAccent.Render("? "))
	s.WriteString(styleText.Bold(true).Render(m.label))
	s.WriteString("  " + styleDim.Render("↑/↓ navigate · Enter select"))
	s.WriteString("\n\n")

	for i, opt := range m.options {
		if m.cursor == i {
			s.WriteString("  " + styleMenuSelected.Render("▶  "+opt) + "\n")
		} else {
			s.WriteString("     " + styleText.Render(opt) + "\n")
		}
	}
	s.WriteString("\n")
	return s.String()
}

// Select displays a list of options and returns the index of the chosen one.
func Select(label string, options []string) (int, error) {
	if SelectFn != nil {
		return SelectFn(label, options)
	}
	if !IsTTY() {
		return -1, ErrNotInteractive
	}
	m := SelectModel{
		label:   label,
		options: options,
		chosen:  -1,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return -1, err
	}

	m = finalModel.(SelectModel)
	if m.canceled {
		return -1, fmt.Errorf("interrupted")
	}

	return m.chosen, nil
}

// Confirm asks a yes/no question and returns true for yes.
func Confirm(question string, defaultYes bool) bool {
	if ConfirmFn != nil {
		return ConfirmFn(question, defaultYes)
	}
	if !IsTTY() {
		Warn("Not an interactive terminal — using default answer (" + map[bool]string{true: "yes", false: "no"}[defaultYes] + ") for: " + question)
		return defaultYes
	}
	options := []string{"Yes", "No"}
	cursor := 0
	if !defaultYes {
		cursor = 1
	}

	m := SelectModel{
		label:   question,
		options: options,
		chosen:  -1,
		cursor:  cursor,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return defaultYes
	}

	m = finalModel.(SelectModel)
	if m.canceled {
		return defaultYes
	}
	return m.chosen == 0
}

// ── Animated Success ──────────────────────────────────────────────────────────

// typewriterModel animates a message character by character using Bubble Tea.
type typewriterModel struct {
	full string // complete rendered line (with ANSI)
	raw  string // plain text for counting
	pos  int    // chars revealed so far
	done bool
}

type twTickMsg struct{}

func twTick() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(_ time.Time) tea.Msg {
		return twTickMsg{}
	})
}

func (m typewriterModel) Init() tea.Cmd { return twTick() }

func (m typewriterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case twTickMsg:
		if m.pos < len(m.raw) {
			m.pos++
			if m.pos >= len(m.raw) {
				m.done = true
				return m, tea.Quit
			}
			return m, twTick()
		}
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m typewriterModel) View() string {
	if m.done || m.pos >= len(m.raw) {
		return "\r" + m.full + "\n"
	}
	// Show plain-text prefix up to m.pos + blinking cursor block
	visible := m.raw[:m.pos]
	cursor := lipgloss.NewStyle().Foreground(colAccent).Render("█")
	return "\r" + styleSuccess.Render("✔ "+visible) + cursor
}

// AnimatedSuccess prints msg with a typewriter animation when connected to a TTY.
// Falls back to plain Success() in non-interactive contexts (pipes, CI).
func AnimatedSuccess(msg string) {
	if !IsTTY() {
		Success(msg)
		return
	}

	m := typewriterModel{
		full: styleSuccess.Render("✔ " + msg),
		raw:  "✔ " + msg,
		pos:  0,
	}

	p := tea.NewProgram(m, tea.WithoutRenderer())
	if _, err := p.Run(); err != nil {
		// Fallback if Bubble Tea can't run
		Success(msg)
	}
}

// ── Spinner ───────────────────────────────────────────────────────────────────

// spinnerModel drives a dot-cycle spinner using Bubble Tea.
type spinnerModel struct {
	label  string
	frames []string
	frame  int
	stop   chan struct{}
	done   chan struct{}
}

type spinTickMsg struct{}

func spinTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
		return spinTickMsg{}
	})
}

type spinStopMsg struct{}

func (m spinnerModel) Init() tea.Cmd { return spinTick() }

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case spinTickMsg:
		select {
		case <-m.stop:
			return m, tea.Quit
		default:
		}
		m.frame = (m.frame + 1) % len(m.frames)
		return m, spinTick()
	case spinStopMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m spinnerModel) View() string {
	dot := lipgloss.NewStyle().Foreground(colAccent).Bold(true).Render(m.frames[m.frame])
	label := styleText.Render(m.label)
	return "\r" + dot + "  " + label + "  "
}

// Spinner starts a spinner with the given label and returns a stop function.
// Call the returned function when the operation is done. Spinner clears the line.
// Falls back to a no-op in non-TTY environments.
func Spinner(label string) func() {
	if !IsTTY() {
		Info(label)
		return func() {}
	}

	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	m := spinnerModel{
		label:  label,
		frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		stop:   stopCh,
		done:   doneCh,
	}

	p := tea.NewProgram(m, tea.WithoutRenderer())

	go func() {
		defer close(doneCh)
		p.Run() //nolint:errcheck
		// Clear the spinner line
		fmt.Print("\r\033[K")
	}()

	return func() {
		close(stopCh)
		<-doneCh
	}
}
