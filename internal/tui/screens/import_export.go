package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// importExportOption is one row in the ImportExport screen.
type importExportOption struct {
	label string
	key   string
	desc  string
}

// hasUnimportedOriginalIdentity reports whether the machine's global git
// config (~/.gitconfig — not a repo-local override) currently holds a
// name+email that isn't already one of git-user's registered profiles.
// Used to decide whether "import existing git identity" has anything new to
// offer, so it isn't shown when there's nothing left to import.
func hasUnimportedOriginalIdentity(store *config.Store) bool {
	name := git.CurrentGlobalName()
	email := git.CurrentGlobalEmail()
	if name == "" && email == "" {
		return false
	}
	return !store.HasUserWithNameEmail(name, email)
}

// buildImportExportOptions returns the ImportExport screen's rows, including
// "Import original gitconfig" only when there's actually something new to
// import (see hasUnimportedOriginalIdentity).
func buildImportExportOptions(store *config.Store) []importExportOption {
	options := []importExportOption{
		{
			label: "› Export current identity",
			key:   "export-current",
			desc:  "Bundle the active identity's keys into an encrypted file",
		},
		{
			label: "› Export all identities",
			key:   "export-all",
			desc:  "Bundle all non-temporary identities (skips passphrase-protected keys)",
		},
		{
			label: "› Import identities",
			key:   "import",
			desc:  "Restore identities from an encrypted bundle file",
		},
	}
	if hasUnimportedOriginalIdentity(store) {
		options = append(options, importExportOption{
			label: "› Import original gitconfig",
			key:   "import-original",
			desc:  "Import your existing ~/.gitconfig identity (you pick the name)",
		})
	}
	return append(options, importExportOption{
		label: "← Back",
		key:   "back",
		desc:  "Return to previous menu",
	})
}

// ImportExport is the sub-screen for import/export operations.
type ImportExport struct {
	store   *config.Store
	options []importExportOption
	cursor  int
	theme   theme.Theme
}

// NewImportExport creates a new ImportExport sub-screen.
func NewImportExport(store *config.Store, th theme.Theme) *ImportExport {
	return &ImportExport{
		store:   store,
		options: buildImportExportOptions(store),
		cursor:  0,
		theme:   th,
	}
}

func (s *ImportExport) Init() tea.Cmd { return nil }

func (s *ImportExport) Title() string { return "Import / Export" }

func (s *ImportExport) ShortHelp() string { return core.ImportExportHelp() }

func (s *ImportExport) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
			return s, func() tea.Msg { return core.ScreenPopMsg{} }
		}
		switch msg.String() {
		case core.KeyCtrlC:
			return s, tea.Quit
		case core.KeyQuit:
			return s, func() tea.Msg { return core.ActionResultMsg{Kind: "quit-confirm"} }

		case core.KeyUp, core.KeyK:
			if s.cursor > 0 {
				s.cursor--
			}

		case core.KeyDown, core.KeyJ:
			if s.cursor < len(s.options)-1 {
				s.cursor++
			}

		case core.KeyEnter:
			opt := s.options[s.cursor]
			if opt.key == "back" {
				return s, func() tea.Msg { return core.ScreenPopMsg{} }
			}
			return s, func() tea.Msg {
				return core.ActionResultMsg{Kind: opt.key}
			}
		}
	}
	return s, nil
}

func (s *ImportExport) View(width, height int) string {
	var sb strings.Builder

	// Title
	sb.WriteString(s.theme.PaneTitle().Render("Import / Export"))
	sb.WriteString("\n")
	sb.WriteString(s.theme.SeparatorLine(width - 6))
	sb.WriteString("\n\n")

	// Subtitle
	sb.WriteString("  ")
	sb.WriteString(s.theme.Dim().Render("Choose an operation:"))
	sb.WriteString("\n\n")

	// Options
	for i, opt := range s.options {
		isCursor := i == s.cursor

		var labelLine string
		if isCursor {
			labelLine = s.theme.Selected().Render("▶ " + opt.label)
		} else {
			labelLine = "  " + opt.label
		}

		descStyle := s.theme.Dim().Italic(true)
		if isCursor {
			descStyle = s.theme.Subtle().Italic(true)
		}

		sb.WriteString(labelLine)
		sb.WriteString("\n")
		sb.WriteString("    ")
		sb.WriteString(descStyle.Render(opt.desc))
		sb.WriteString("\n\n")
	}

	return sb.String()
}
