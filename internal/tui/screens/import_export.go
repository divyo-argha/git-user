package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// importExportOption is one row in the ImportExport screen.
type importExportOption struct {
	label string
	key   string
	desc  string
}

var importExportOptions = []importExportOption{
	{
		label: "📤  Export current identity",
		key:   "export-current",
		desc:  "Bundle the active identity's keys into an encrypted file",
	},
	{
		label: "📦  Export all identities",
		key:   "export-all",
		desc:  "Bundle all non-temporary identities (skips passphrase-protected keys)",
	},
	{
		label: "📥  Import identities",
		key:   "import",
		desc:  "Restore identities from an encrypted bundle file",
	},
	{
		label: "🗂  Import original gitconfig",
		key:   "import-original",
		desc:  "Import identity from your original ~/.gitconfig backup",
	},
}

// ImportExport is the sub-screen for import/export operations.
type ImportExport struct {
	store  *config.Store
	cursor int
	theme  theme.Theme
}

// NewImportExport creates a new ImportExport sub-screen.
func NewImportExport(store *config.Store, th theme.Theme) *ImportExport {
	return &ImportExport{
		store:  store,
		cursor: 0,
		theme:  th,
	}
}

func (s *ImportExport) Init() tea.Cmd { return nil }

func (s *ImportExport) Title() string { return "Import / Export" }

func (s *ImportExport) ShortHelp() string { return core.ImportExportHelp() }

func (s *ImportExport) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case core.KeyCtrlC:
			return s, tea.Quit

		case core.KeyEsc, core.KeyQuit:
			return s, func() tea.Msg { return core.ScreenPopMsg{} }

		case core.KeyUp, core.KeyK:
			if s.cursor > 0 {
				s.cursor--
			}

		case core.KeyDown, core.KeyJ:
			if s.cursor < len(importExportOptions)-1 {
				s.cursor++
			}

		case core.KeyEnter:
			opt := importExportOptions[s.cursor]
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
	title := s.theme.PaneTitle().Render("Import / Export")
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(s.theme.SeparatorLine(width - 6))
	sb.WriteString("\n\n")

	// Subtitle
	subtitle := s.theme.Dim().Render("  Choose an operation:")
	sb.WriteString(subtitle)
	sb.WriteString("\n\n")

	// Options
	for i, opt := range importExportOptions {
		isCursor := i == s.cursor

		var labelLine string
		if isCursor {
			labelLine = s.theme.Selected().Render("▶ " + opt.label)
		} else {
			labelLine = "  " + opt.label
		}

		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Italic(true)
		if isCursor {
			descStyle = descStyle.Foreground(lipgloss.Color("#888888"))
		}
		descLine := descStyle.Render("    " + opt.desc)

		sb.WriteString(labelLine)
		sb.WriteString("\n")
		sb.WriteString(descLine)
		sb.WriteString("\n\n")
	}

	return sb.String()
}
