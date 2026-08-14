package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// firstRunOption is one row in the FirstRun screen.
type firstRunOption struct {
	label string
	key   string
	desc  string
}

// FirstRun is the first-run onboarding screen shown when git-user detects an
// existing Git identity in the global gitconfig. It lets the user import it
// (with its original username) or skip and register a new identity manually.
type FirstRun struct {
	store  *config.Store
	name   string
	email  string
	cursor int
	theme  theme.Theme
}

var firstRunOptions = []firstRunOption{
	{
		label: "↓ Import existing identity",
		key:   "import",
		desc:  "Bring your existing ~/.gitconfig identity into git-user",
	},
	{
		label: "→ Skip for now",
		key:   "skip",
		desc:  "Register identities manually from the dashboard instead",
	},
}

// NewFirstRun creates the first-run onboarding screen for a detected original
// identity. name/email are the detected gitconfig identity details.
func NewFirstRun(store *config.Store, name, email string, th theme.Theme) *FirstRun {
	return &FirstRun{
		store:  store,
		name:   name,
		email:  email,
		cursor: 0,
		theme:  th,
	}
}

func (s *FirstRun) Init() tea.Cmd { return nil }

func (s *FirstRun) Title() string { return "First Run Setup" }

func (s *FirstRun) ShortHelp() string { return core.OptionsHelp() }

func (s *FirstRun) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if core.IsEscKey(msg) {
			// Esc = decline, same as Skip.
			return s, func() tea.Msg {
				return core.ActionResultMsg{Kind: "firstrun-skip"}
			}
		}
		switch msg.String() {
		case core.KeyCtrlC, core.KeyQuit:
			return s, tea.Quit
		case core.KeyUp, core.KeyK:
			if s.cursor > 0 {
				s.cursor--
			}
		case core.KeyDown, core.KeyJ:
			if s.cursor < len(firstRunOptions)-1 {
				s.cursor++
			}
		case core.KeyEnter:
			opt := firstRunOptions[s.cursor]
			switch opt.key {
			case "import":
				return s, func() tea.Msg {
					return core.ActionResultMsg{Kind: "firstrun-import"}
				}
			case "skip":
				return s, func() tea.Msg {
					return core.ActionResultMsg{Kind: "firstrun-skip"}
				}
			}
		}
	}
	return s, nil
}

func (s *FirstRun) View(width, height int) string {
	var sb strings.Builder

	// Title
	sb.WriteString(s.theme.PaneTitle().Render("First Run Setup"))
	sb.WriteString("\n")
	sb.WriteString(s.theme.SeparatorLine(width - 6))
	sb.WriteString("\n\n")

	detected := "Your Git user.name and user.email"
	if s.name != "" && s.email != "" {
		detected = fmt.Sprintf("%s <%s>", s.name, s.email)
	} else if s.name != "" {
		detected = s.name
	} else if s.email != "" {
		detected = s.email
	}

	sb.WriteString(s.theme.Dim().Render("  git-user found an existing Git identity:"))
	sb.WriteString("\n")
	sb.WriteString(s.theme.Bold().Render("    " + detected))
	sb.WriteString("\n\n")
	sb.WriteString(s.theme.Dim().Render("  Import it now (keeps its original username) or skip and add it later."))
	sb.WriteString("\n\n")

	for i, opt := range firstRunOptions {
		isCursor := i == s.cursor
		var labelLine string
		if isCursor {
			labelLine = s.theme.Selected().Render("▶ " + opt.label)
		} else {
			labelLine = "  " + opt.label
		}
		descLine := s.theme.Dim().Render("    " + opt.desc)
		sb.WriteString(labelLine)
		sb.WriteString("\n")
		sb.WriteString(descLine)
		sb.WriteString("\n\n")
	}

	return sb.String()
}
