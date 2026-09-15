package screens

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// PolicyScreen views and manages the repository-level .git-user-policy file
// and its enforcement, plus a way into the Signers screen for .allowed-signers.
type PolicyScreen struct {
	store    *config.Store
	theme    theme.Theme
	actions  components.ActionMenu
	repoRoot string
	repoErr  error
}

func NewPolicyScreen(store *config.Store, th theme.Theme) *PolicyScreen {
	p := &PolicyScreen{store: store, theme: th}
	p.repoRoot, p.repoErr = git.RepoRoot()
	p.refreshActions()
	return p
}

func (p *PolicyScreen) refreshActions() {
	var items []components.ActionItem
	items = append(items, components.ActionItem{Label: "Policy", IsSection: true})
	if p.repoErr != nil {
		items = append(items, components.ActionItem{Label: "Not in a git repository", Key: "", Disabled: true})
	} else {
		items = append(items, components.ActionItem{Label: "Create / edit policy", Key: "policy-edit"})
		items = append(items, components.ActionItem{Label: "Enforcement", IsSection: true})
		items = append(items, components.ActionItem{Label: "Install enforcing hook", Key: "policy-hook-install"})
		items = append(items, components.ActionItem{Label: "Trusted Signers", IsSection: true})
		items = append(items, components.ActionItem{Label: "Manage allowed-signers", Key: "signers"})
	}
	items = append(items, components.ActionItem{Label: "", IsSection: true})
	items = append(items, components.ActionItem{Label: "← Back", Key: "back"})

	prevKey := ""
	if sel := p.actions.Selected(); sel != nil {
		prevKey = sel.Key
	}
	p.actions = components.NewActionMenu("Repository Policy", items, p.theme)
	if prevKey != "" {
		p.actions.FindAndSetCursorByKey(prevKey)
	}
}

func (p *PolicyScreen) Init() tea.Cmd { return nil }

func (p *PolicyScreen) Title() string { return "Repository Policy" }

func (p *PolicyScreen) ShortHelp() string {
	return "↑/↓•navigate  ⏎•select  Esc•back"
}

func (p *PolicyScreen) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case core.StoreRefreshedMsg:
		p.refreshActions()
	case tea.KeyMsg:
		return p.handleKey(msg)
	}
	return p, nil
}

func (p *PolicyScreen) handleKey(msg tea.KeyMsg) (core.Screen, tea.Cmd) {
	if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
		return p, func() tea.Msg { return core.ScreenPopMsg{} }
	}
	switch msg.String() {
	case core.KeyCtrlC:
		return p, tea.Quit
	case core.KeyQuit:
		return p, func() tea.Msg { return core.ActionResultMsg{Kind: "quit-confirm"} }
	case core.KeyUp, core.KeyK:
		p.actions.CursorUp()
	case core.KeyDown, core.KeyJ:
		p.actions.CursorDown()
	case core.KeyEnter:
		return p.handleEnter()
	}
	return p, nil
}

func (p *PolicyScreen) handleEnter() (core.Screen, tea.Cmd) {
	item := p.actions.Selected()
	if item == nil || item.Disabled || item.Key == "" {
		return p, nil
	}
	if item.Key == "back" {
		return p, func() tea.Msg { return core.ScreenPopMsg{} }
	}
	return p, func() tea.Msg { return core.ActionResultMsg{Kind: item.Key} }
}

func hookInstalledMarker(repoRoot string) bool {
	content, err := os.ReadFile(filepath.Join(repoRoot, ".git", "hooks", "pre-commit"))
	if err != nil {
		return false
	}
	return strings.HasPrefix(string(content), "#!/bin/sh\n# git-user")
}

func (p *PolicyScreen) View(width, height int) string {
	var sb strings.Builder

	sb.WriteString("\n  " + p.theme.Bold().Render(p.Title()) + "\n\n")

	if p.repoErr != nil {
		sb.WriteString("  " + p.theme.Dim().Render("Not inside a git repository.") + "\n\n")
	} else {
		policy, err := config.LoadRepoPolicy(p.repoRoot)
		if err != nil {
			sb.WriteString("  " + p.theme.ErrorStyle().Render("Error reading .git-user-policy: "+err.Error()) + "\n\n")
		} else {
			policyPath := filepath.Join(p.repoRoot, config.RepoPolicyFileName)
			exists := false
			if _, statErr := os.Stat(policyPath); statErr == nil {
				exists = true
			}

			existsStr := p.theme.Dim().Render("No")
			if exists {
				existsStr = p.theme.SuccessStyle().Render("Yes")
			}
			sb.WriteString(fmt.Sprintf("  .git-user-policy exists : %s\n", existsStr))

			signStr := p.theme.Dim().Render("No")
			if policy.RequireSigning {
				signStr = p.theme.SuccessStyle().Render("Yes")
			}
			sb.WriteString(fmt.Sprintf("  Require signing         : %s\n", signStr))

			domainsStr := p.theme.Dim().Render("(none)")
			if len(policy.AllowedEmailDomains) > 0 {
				domainsStr = strings.Join(policy.AllowedEmailDomains, ", ")
			}
			sb.WriteString(fmt.Sprintf("  Allowed email domains   : %s\n", domainsStr))

			hookStr := p.theme.WarningStyle().Render(theme.IconWarn + " Not installed")
			if hookInstalledMarker(p.repoRoot) {
				hookStr = p.theme.SuccessStyle().Render(theme.IconPass + " Installed")
			}
			sb.WriteString(fmt.Sprintf("  Enforcing hook          : %s\n", hookStr))

			entries, _ := config.LoadAllowedSigners(p.repoRoot)
			sb.WriteString(fmt.Sprintf("  Trusted signers         : %d entr%s\n", len(entries), pluralY(len(entries))))
			sb.WriteString("\n")
		}
	}

	sb.WriteString(p.theme.SeparatorLine(width-6) + "\n\n")

	items := p.actions.Items()
	cursor := p.actions.Cursor()
	for i, item := range items {
		if item.IsSection {
			if item.Label != "" {
				sb.WriteString("  " + p.theme.SectionHeader().Render(item.Label) + "\n")
			}
			continue
		}
		label := item.Label
		if i == cursor {
			label = p.theme.Selected().Render("▶ " + label)
		} else if item.Disabled {
			label = p.theme.Dim().Render("  " + label)
		} else {
			label = "  " + label
		}
		sb.WriteString("  " + label + "\n")
	}

	return sb.String()
}

func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
