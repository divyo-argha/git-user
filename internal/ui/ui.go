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
	"github.com/mattn/go-isatty"
)

var ErrNotInteractive = errors.New("this command requires an interactive terminal — pass the needed value as a flag instead")

var (
	colPrimary = lipgloss.Color("#7AA2F7")
	colSecond  = lipgloss.Color("#9ECE6A")
	colAccent  = lipgloss.Color("#BB9AF7")
	colDanger  = lipgloss.Color("#F7768E")
	colWarning = lipgloss.Color("#E0AF68")
	colMuted   = lipgloss.Color("#565F89")
	colText    = lipgloss.Color("#C0CAF5")
	colBg      = lipgloss.Color("#1F2335")

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

// IsTTY returns true if stdout is a character device (terminal).
func IsTTY() bool {
	if IsTTYFn != nil {
		return IsTTYFn()
	}
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return true
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// StdinIsTTY returns true if stdin is attached to an interactive terminal.
func StdinIsTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// IsPlainOutput reports whether styled/banner output should be suppressed.
func IsPlainOutput(args []string) bool {
	for _, a := range args {
		if a == "--plain" {
			return true
		}
	}
	return !IsTTY()
}

// IsJSONOutput reports whether the caller requested machine-readable JSON via --json.
func IsJSONOutput(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

func PrintLogo() {
	lines := logo.GetTrimmedLogo()
	fmt.Println(strings.Join(lines, "\n"))
}

func PrintBanner(ver string) {
	lines := logo.GetTrimmedLogo()
	fmt.Println(strings.Join(lines, "\n"))
	if ver != "" {
		fmt.Printf("  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99")).Render(fmt.Sprintf("Version %s", ver)))
	}
}

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
		sb.WriteString(" ")
		sb.WriteString(lipgloss.NewStyle().Foreground(colSecond).Render("(verified)"))
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colSecond).
		Padding(1, 3).
		Render(sb.String())

	fmt.Println(card)
	fmt.Println(lipgloss.NewStyle().Foreground(colMuted).Render("  Run 'git-user' (or 'gu') to launch the interactive dashboard."))
	fmt.Println()
}

func PrintUpdateCurrent(ver string) {
	fmt.Println()
	PrintBanner(ver)
	fmt.Println()

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colSecond).Bold(true).Render("✨ git-user is already up to date!"))
	sb.WriteString("\n\n   ")
	sb.WriteString(lipgloss.NewStyle().Foreground(colSecond).Bold(true).Render(ver))
	sb.WriteString(" ")
	sb.WriteString(lipgloss.NewStyle().Foreground(colMuted).Render("(latest release)"))

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colSecond).
		Padding(1, 3).
		Render(sb.String())

	fmt.Println(card)
	fmt.Println(lipgloss.NewStyle().Foreground(colMuted).Render("  Run 'git-user' (or 'gu') to launch the interactive dashboard."))
	fmt.Println()
}

func Success(msg string) {
	fmt.Println(styleSuccess.Render("✔ " + msg))
}

func Successf(format string, args ...any) {
	Success(fmt.Sprintf(format, args...))
}

func Info(msg string) {
	fmt.Println(styleInfo.Render("ℹ " + msg))
}

func Warn(msg string) {
	fmt.Println(styleWarn.Render("⚠ " + msg))
}

func Error(msg string) {
	fmt.Fprintln(os.Stderr, styleError.Render("✖ "+msg))
}

func Errorf(format string, args ...any) {
	Error(fmt.Sprintf(format, args...))
}

func StyleDim() lipgloss.Style     { return styleDim }
func StyleSuccess() lipgloss.Style { return styleSuccess }

func Header(msg string) {
	fmt.Println(styleHeader.Render(strings.ToUpper(msg)))
}

func Banner(msg string) {
	fmt.Println(styleBanner.Render("  " + strings.ToUpper(msg) + "  "))
}

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

func UserDetails(name, email, sshKey string) {
	label := lipgloss.NewStyle().Foreground(colPrimary).Bold(true)
	fmt.Printf("  %-10s  %s\n", label.Render("Name  :"), name)
	fmt.Printf("  %-10s  %s\n", label.Render("Email :"), styleDim.Render(email))
	if sshKey != "" {
		fmt.Printf("  %-10s  %s\n", label.Render("Key   :"), styleDim.Render(sshKey))
	}
}

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
	s.WriteString("  ")
	s.WriteString(styleDim.Render("↑/↓ navigate · Enter select"))
	s.WriteString("\n\n")

	for i, opt := range m.options {
		if m.cursor == i {
			s.WriteString("  ")
			s.WriteString(styleMenuSelected.Render("▶  " + opt))
			s.WriteString("\n")
		} else {
			s.WriteString("     ")
			s.WriteString(styleText.Render(opt))
			s.WriteString("\n")
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

type typewriterModel struct {
	full  string
	runes []rune
	pos   int
	done  bool
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
		if m.pos < len(m.runes) {
			m.pos++
			if m.pos >= len(m.runes) {
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
	if m.done || m.pos >= len(m.runes) {
		return "\r" + m.full + "\n"
	}
	visible := string(m.runes[:m.pos])
	cursor := lipgloss.NewStyle().Foreground(colAccent).Render("█")
	return "\r" + styleSuccess.Render("✔ "+visible) + cursor
}

// AnimatedSuccess prints msg with a typewriter animation when connected to a TTY.
func AnimatedSuccess(msg string) {
	if os.Getenv("CI") != "" || !IsTTY() {
		Success(msg)
		return
	}

	m := typewriterModel{
		full:  styleSuccess.Render("✔ " + msg),
		runes: []rune("✔ " + msg),
		pos:   0,
	}

	p := tea.NewProgram(m, tea.WithoutRenderer(), tea.WithInput(strings.NewReader("")))
	if _, err := p.Run(); err != nil {
		Success(msg)
	}
}

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

	p := tea.NewProgram(m, tea.WithoutRenderer(), tea.WithInput(strings.NewReader("")))

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
