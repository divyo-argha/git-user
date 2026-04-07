package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tea_Model = tea.Model

var (
	// Base Colors
	cyan    = lipgloss.Color("#00FFFF")
	magenta = lipgloss.Color("#FF00FF")
	green   = lipgloss.Color("#00FF00")
	yellow  = lipgloss.Color("#FFFF00")
	red     = lipgloss.Color("#FF0000")
	gray    = lipgloss.Color("#444444")
	white   = lipgloss.Color("#FFFFFF")

	// Styles
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan).
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(magenta).
			MarginBottom(1)

	styleBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(cyan).
			Padding(0, 1).
			MarginBottom(1).
			MarginTop(1)

	styleSuccess = lipgloss.NewStyle().Foreground(green).Bold(true)
	styleInfo    = lipgloss.NewStyle().Foreground(cyan)
	styleWarn    = lipgloss.NewStyle().Foreground(yellow)
	styleError   = lipgloss.NewStyle().Foreground(red).Bold(true)
	styleDim     = lipgloss.NewStyle().Foreground(gray)

	styleCardActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(green).
			Padding(0, 2).
			MarginBottom(1).
			Width(60)

	styleCardInactive = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(gray).
				Padding(0, 2).
				MarginBottom(1).
				Width(60)

	styleActiveBadge = lipgloss.NewStyle().
				Foreground(white).
				Background(green).
				Padding(0, 1).
				Bold(true)

	styleMenuSelected = lipgloss.NewStyle().
				Foreground(cyan).
				Bold(true)

	styleHelp = lipgloss.NewStyle().
			Foreground(gray).
			Italic(true)
)

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Success prints a green ✔ message.
func Success(msg string) {
	fmt.Println(styleSuccess.Render("✔ " + msg))
}

// Successf prints a formatted green ✔ message.
func Successf(format string, args ...any) {
	Success(fmt.Sprintf(format, args...))
}

// Info prints a cyan ℹ message.
func Info(msg string) {
	fmt.Println(styleInfo.Render("ℹ " + msg))
}

// Warn prints a yellow ⚠ message.
func Warn(msg string) {
	fmt.Println(styleWarn.Render("⚠ " + msg))
}

// Error prints a red ✖ message to stderr.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, styleError.Render("✖ "+msg))
}

// Errorf prints a formatted red ✖ message to stderr.
func Errorf(format string, args ...any) {
	Error(fmt.Sprintf(format, args...))
}

// StyleDim returns the style used for dimmed text.
func StyleDim() lipgloss.Style {
	return styleDim
}

// StyleSuccess returns the style used for success/verified text.
func StyleSuccess() lipgloss.Style {
	return styleSuccess
}

// UserRow prints a single user card in the list.
func UserRow(name, email, sshKey string, active bool) {
	badge := ""
	cardStyle := styleCardInactive
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(white)

	if active {
		badge = styleActiveBadge.Render(" ACTIVE ") + " "
		cardStyle = styleCardActive
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(green)
	}

	content := fmt.Sprintf("%s%s\n%s",
		badge,
		nameStyle.Render(name),
		styleDim.Render(email),
	)

	if sshKey != "" {
		content += "\n" + styleDim.Render("Key: "+sshKey)
	}

	fmt.Println(cardStyle.Render(content))
}

// UserDetails prints the details of a single user.
func UserDetails(name, email, sshKey string) {
	fmt.Printf("  %-10s: %s\n", styleDim.Render("Name"), name)
	fmt.Printf("  %-10s: %s\n", styleDim.Render("Email"), email)
	if sshKey != "" {
		fmt.Printf("  %-10s: %s\n", styleDim.Render("Key"), sshKey)
	}
}

// Header prints a bold section header with a border.
func Header(msg string) {
	fmt.Println(styleHeader.Render(strings.ToUpper(msg)))
}

// Banner prints a full-width background banner.
func Banner(msg string) {
	fmt.Println(styleBanner.Render(" " + strings.ToUpper(msg) + " "))
}

// Divider prints a thin separator line.
func Divider() {
	fmt.Println(styleDim.Render("────────────────────────────────────────────────────────────"))
}

// RawMode toggles terminal raw mode. (Now managed by Bubble Tea for Select)
func RawMode(on bool) error {
	return nil
}

// Prompt asks the user for text input.
func Prompt(label string) (string, error) {
	fmt.Printf("%s %s ", styleInfo.Render("?"), lipgloss.NewStyle().Bold(true).Render(label))
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// SelectModel is the Bubble Tea model for the selection menu.
type SelectModel struct {
	label    string
	options  []string
	cursor   int
	chosen   int
	canceled bool
}

func (m SelectModel) Init() tea.Cmd {
	return nil
}

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
	s.WriteString(styleInfo.Render("? "))
	s.WriteString(lipgloss.NewStyle().Bold(true).Render(m.label))
	s.WriteString(" " + styleDim.Render("(Use arrows, Enter to select)"))
	s.WriteString("\n\n")

	for i, opt := range m.options {
		if m.cursor == i {
			s.WriteString("  " + styleMenuSelected.Render("▶ "+opt) + "\n")
		} else {
			s.WriteString("    " + opt + "\n")
		}
	}
	s.WriteString("\n")
	return s.String()
}

// Select displays a list of options and returns the index of the chosen one.
func Select(label string, options []string) (int, error) {
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
