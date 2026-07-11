package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/logo"
)

// StatusBar renders the top header bar with logo, active profile, and SSH agent status.
type StatusBar struct {
	store          *config.Store
	agentConnected bool
	agentKeyCount  int
	agentChecked   bool
	theme          theme.Theme
}

// NewStatusBar creates a new status bar component.
func NewStatusBar(store *config.Store, th theme.Theme) StatusBar {
	return StatusBar{store: store, theme: th}
}

// SetStore updates the config store reference.
func (s *StatusBar) SetStore(store *config.Store) { s.store = store }

// SetAgentStatus updates the SSH agent status.
func (s *StatusBar) SetAgentStatus(connected bool, keyCount int) {
	s.agentConnected = connected
	s.agentKeyCount = keyCount
	s.agentChecked = true
}

// View renders the status bar.
func (s StatusBar) View(width, termHeight int) string {
	if termHeight > 0 && termHeight < 35 {
		return s.viewCompact()
	}
	return s.viewFull()
}

func (s StatusBar) viewFull() string {
	logoLines := logo.NewSmallPixelLines

	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFAA")).Bold(true)
	tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666688")).Italic(true)
	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF99"))
	actName := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF99")).Bold(true)
	actEmail := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888AA"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777799"))

	rightLines := []string{
		nameStyle.Render("GIT-USER"),
		tagStyle.Render("switch git identities in one command"),
		"",
	}

	if s.store != nil && s.store.Current != "" {
		if u := s.store.CurrentUser(); u != nil {
			rightLines = append(rightLines, fmt.Sprintf("%s  %s %s",
				labelStyle.Render("Active profile :"),
				dotStyle.Render("●"),
				actName.Render(u.Name)+" "+actEmail.Render("("+u.Email+")"),
			))
		} else {
			rightLines = append(rightLines, fmt.Sprintf("%s  %s",
				labelStyle.Render("Active profile :"),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render(s.store.Current+" (missing)"),
			))
		}
	} else {
		rightLines = append(rightLines, fmt.Sprintf("%s  %s",
			labelStyle.Render("Active profile :"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("None (logged out)"),
		))
	}

	if s.agentChecked {
		if s.agentConnected {
			agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF66")).Render("Connected")
			rightLines = append(rightLines, fmt.Sprintf("%s  %s (%d keys loaded)",
				labelStyle.Render("SSH Agent      :"),
				agentStr,
				s.agentKeyCount,
			))
		} else {
			agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("Not reachable")
			rightLines = append(rightLines, fmt.Sprintf("%s  %s",
				labelStyle.Render("SSH Agent      :"),
				agentStr,
			))
		}
	} else {
		rightLines = append(rightLines, fmt.Sprintf("%s  %s",
			labelStyle.Render("SSH Agent      :"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("checking..."),
		))
	}

	logoH := len(logoLines)
	padTop := (logoH - len(rightLines)) / 2
	if padTop < 0 {
		padTop = 0
	}
	rightBlock := strings.Repeat("\n", padTop) + strings.Join(rightLines, "\n")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		strings.Join(logoLines, "\n"),
		"   ",
		rightBlock,
	)
}

func (s StatusBar) viewCompact() string {
	header := s.theme.Bold().Render("  git-user")
	if s.store != nil && s.store.Current != "" {
		if u := s.store.CurrentUser(); u != nil {
			header += "  " + s.theme.Dim().Render("active: "+u.Name+" ("+u.Email+")")
		}
	}
	return header
}
