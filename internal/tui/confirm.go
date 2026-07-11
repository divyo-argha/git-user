package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Confirm is a modal confirmation dialog.
type Confirm struct {
	question string
	context  string // identifies what is being confirmed
	cursor   int    // 0 = Yes, 1 = Cancel
	theme    theme.Theme
}

// NewConfirm creates a new confirmation dialog.
func NewConfirm(question, context string, th theme.Theme) *Confirm {
	return &Confirm{
		question: question,
		context:  context,
		cursor:   1, // default to Cancel for safety
		theme:    th,
	}
}

func (c *Confirm) Init() tea.Cmd { return nil }

func (c *Confirm) Title() string { return "Confirm" }

func (c *Confirm) ShortHelp() string { return ConfirmHelp() }

func (c *Confirm) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case KeyEsc:
			return c, func() tea.Msg {
				return ConfirmResultMsg{Confirmed: false, Context: c.context}
			}

		case KeyLeft, KeyH:
			c.cursor = 0

		case KeyRight, KeyL:
			c.cursor = 1

		case "y", "Y":
			return c, func() tea.Msg {
				return ConfirmResultMsg{Confirmed: true, Context: c.context}
			}

		case "n", "N":
			return c, func() tea.Msg {
				return ConfirmResultMsg{Confirmed: false, Context: c.context}
			}

		case KeyEnter:
			return c, func() tea.Msg {
				return ConfirmResultMsg{Confirmed: c.cursor == 0, Context: c.context}
			}

		case KeyCtrlC:
			return c, tea.Quit
		}
	}
	return c, nil
}

func (c *Confirm) View(width, height int) string {
	var sb strings.Builder

	boxWidth := 50
	if width < 60 {
		boxWidth = width - 10
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	padTop := (height - 8) / 2
	if padTop < 0 {
		padTop = 0
	}
	for i := 0; i < padTop; i++ {
		sb.WriteString("\n")
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, c.theme.Bold().Render(c.question))
	lines = append(lines, "")

	yesLabel := "  Yes  "
	noLabel := "  Cancel  "

	if c.cursor == 0 {
		yesLabel = c.theme.Selected().Render("▶ Yes ")
		noLabel = c.theme.Dim().Render("  Cancel  ")
	} else {
		yesLabel = c.theme.Dim().Render("  Yes  ")
		noLabel = c.theme.Selected().Render("▶ Cancel ")
	}

	lines = append(lines, yesLabel+"    "+noLabel)
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	box := c.theme.ActionPane(boxWidth, 0).Render(content)

	padLeft := (width - boxWidth - 6) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	for _, line := range strings.Split(box, "\n") {
		sb.WriteString(strings.Repeat(" ", padLeft) + line + "\n")
	}

	return sb.String()
}
