package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// SignersScreen manages the repository's committed .allowed-signers file.
type SignersScreen struct {
	store    *config.Store
	theme    theme.Theme
	actions  components.ActionMenu
	repoRoot string
	repoErr  error
}

func NewSignersScreen(store *config.Store, th theme.Theme) *SignersScreen {
	s := &SignersScreen{store: store, theme: th}
	s.repoRoot, s.repoErr = git.RepoRoot()
	s.refreshActions()
	return s
}

func (s *SignersScreen) refreshActions() {
	var items []components.ActionItem
	items = append(items, components.ActionItem{Label: "Entries", IsSection: true})

	if s.repoErr == nil {
		entries, _ := config.LoadAllowedSigners(s.repoRoot)
		if len(entries) == 0 {
			items = append(items, components.ActionItem{Label: "(none yet)", Key: "", Disabled: true})
		}
		for _, e := range entries {
			label := fmt.Sprintf("✕ Remove %s", strings.Join(e.Principals, ","))
			items = append(items, components.ActionItem{Label: label, Key: "signer-remove:" + e.Principals[0]})
		}
		items = append(items, components.ActionItem{Label: "Add", IsSection: true})
		items = append(items, components.ActionItem{Label: "Add from a local identity", Key: "signers-add-identity"})
		items = append(items, components.ActionItem{Label: "Add by email + public key file", Key: "signers-add-email"})
	} else {
		items = append(items, components.ActionItem{Label: "Not in a git repository", Key: "", Disabled: true})
	}

	items = append(items, components.ActionItem{Label: "", IsSection: true})
	items = append(items, components.ActionItem{Label: "← Back", Key: "back"})

	prevKey := ""
	if sel := s.actions.Selected(); sel != nil {
		prevKey = sel.Key
	}
	s.actions = components.NewActionMenu("Trusted Signers (.allowed-signers)", items, s.theme)
	if prevKey != "" {
		s.actions.FindAndSetCursorByKey(prevKey)
	}
}

func (s *SignersScreen) Init() tea.Cmd { return nil }

func (s *SignersScreen) Title() string { return "Trusted Signers" }

func (s *SignersScreen) ShortHelp() string {
	return "↑/↓•navigate  ⏎•select  Esc•back"
}

func (s *SignersScreen) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case core.StoreRefreshedMsg:
		s.refreshActions()
	case tea.KeyMsg:
		return s.handleKey(msg)
	}
	return s, nil
}

func (s *SignersScreen) handleKey(msg tea.KeyMsg) (core.Screen, tea.Cmd) {
	if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
		return s, func() tea.Msg { return core.ScreenPopMsg{} }
	}
	switch msg.String() {
	case core.KeyCtrlC:
		return s, tea.Quit
	case core.KeyQuit:
		return s, func() tea.Msg { return core.ActionResultMsg{Kind: "quit-confirm"} }
	case core.KeyUp, core.KeyK:
		s.actions.CursorUp()
	case core.KeyDown, core.KeyJ:
		s.actions.CursorDown()
	case core.KeyEnter:
		return s.handleEnter()
	}
	return s, nil
}

func (s *SignersScreen) handleEnter() (core.Screen, tea.Cmd) {
	item := s.actions.Selected()
	if item == nil || item.Disabled || item.Key == "" {
		return s, nil
	}
	if item.Key == "back" {
		return s, func() tea.Msg { return core.ScreenPopMsg{} }
	}
	return s, func() tea.Msg { return core.ActionResultMsg{Kind: item.Key} }
}

func (s *SignersScreen) View(width, height int) string {
	var sb strings.Builder

	sb.WriteString("\n  " + s.theme.Bold().Render(s.Title()) + "\n")
	sb.WriteString("  " + s.theme.Dim().Render("A git-tracked file mapping contributor emails to SSH public keys, so signature") + "\n")
	sb.WriteString("  " + s.theme.Dim().Render("verification works independent of GitHub's own key store.") + "\n\n")
	sb.WriteString(s.theme.SeparatorLine(width-6) + "\n\n")

	items := s.actions.Items()
	cursor := s.actions.Cursor()
	for i, item := range items {
		if item.IsSection {
			if item.Label != "" {
				sb.WriteString("  " + s.theme.SectionHeader().Render(item.Label) + "\n")
			}
			continue
		}
		label := item.Label
		if i == cursor {
			label = s.theme.Selected().Render("▶ " + label)
		} else if item.Disabled {
			label = s.theme.Dim().Render("  " + label)
		} else {
			label = "  " + label
		}
		sb.WriteString("  " + label + "\n")
	}

	return sb.String()
}
