package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/logo"
)

type tea_Model = tea.Model

// ── Tokyo Night Palette ───────────────────────────────────────────────────────
// Single source of truth — matches internal/tui/theme/theme.go exactly.
var (
	colPrimary  = lipgloss.Color("#7AA2F7") // Soft Blue — info, headers
	colSecond   = lipgloss.Color("#9ECE6A") // Emerald — success, active
	colAccent   = lipgloss.Color("#BB9AF7") // Soft Purple — prompts, accents
	colDanger   = lipgloss.Color("#F7768E") // Rose — errors
	colWarning  = lipgloss.Color("#E0AF68") // Amber — warnings
	colMuted    = lipgloss.Color("#565F89") // Deep Gray — dim, separators
	colText     = lipgloss.Color("#C0CAF5") // Ice Blue-White — primary text
	colTextDim  = lipgloss.Color("#787C99") // Dimmed text
	colBg       = lipgloss.Color("#1F2335") // Card background

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
)

// ── TTY Detection ─────────────────────────────────────────────────────────────

// IsTTY returns true if stdout is a character device (terminal).
func IsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ── Logo ──────────────────────────────────────────────────────────────────────

// PrintLogo prints the git-user design logo to stdout.
func PrintLogo() {
	lines := logo.GetTrimmedLogo()
	fmt.Println(strings.Join(lines, "\n"))
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
func UserRow(name, email, sshKey string, active bool, isOriginal bool) {
	badge := ""
	cardStyle := styleCardInactive
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(colText)

	if active {
		badge = styleActiveBadge.Render(" ACTIVE ") + "  "
		cardStyle = styleCardActive
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(colSecond)
	}

	originalTag := ""
	if isOriginal {
		originalTag = "  " + lipgloss.NewStyle().Foreground(colSecond).Render("(original)")
	}

	content := fmt.Sprintf("%s%s%s\n%s",
		badge,
		nameStyle.Render(name),
		originalTag,
		styleDim.Render(email),
	)

	if sshKey != "" {
		content += "\n" + styleDim.Render("🔑 "+sshKey)
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
	fmt.Printf("%s %s ", styleAccent.Render("?"), styleText.Copy().Bold(true).Render(label))
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
	s.WriteString(styleText.Copy().Bold(true).Render(m.label))
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
	full    string // complete rendered line (with ANSI)
	raw     string // plain text for counting
	pos     int    // chars revealed so far
	done    bool
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
