package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/version"
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
	if termHeight > 0 && termHeight < 15 {
		return s.viewCompact()
	}
	return s.viewFull()
}

func (s StatusBar) viewFull() string {
	logoLines := logo.GetTrimmedLogo()

	versionLine := fmt.Sprintf("  \x1b[38;2;148;163;184mVersion %s\x1b[0m", version.Version)

	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A"))
	actName := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")).Bold(true)
	actEmail := lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true)

	var infoLines []string

	if s.store != nil && s.store.Current != "" {
		if u := s.store.CurrentUser(); u != nil {
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s %s",
				labelStyle.Render("Active profile :"),
				dotStyle.Render("●"),
				actName.Render(u.Name)+" "+actEmail.Render("("+u.Email+")"),
			))
		} else {
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
				labelStyle.Render("Active profile :"),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")).Render(s.store.Current+" (missing)"),
			))
		}
	} else {
		infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Active profile :"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("None (logged out)"),
		))
	}

	if s.agentChecked {
		if s.agentConnected {
			agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")).Bold(true).Render("Connected")
			countStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99")).Render(fmt.Sprintf("(%d keys loaded)", s.agentKeyCount))
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s %s",
				labelStyle.Render("SSH Agent      :"),
				agentStr,
				countStr,
			))
		} else {
			agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")).Render("Not reachable")
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
				labelStyle.Render("SSH Agent      :"),
				agentStr,
			))
		}
	} else {
		infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("SSH Agent      :"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("checking..."),
		))
	}

	var sb strings.Builder
	sb.WriteString(strings.Join(logoLines, "\n"))
	sb.WriteString("\n")
	sb.WriteString(versionLine)
	sb.WriteString("\n\n")
	sb.WriteString(strings.Join(infoLines, "\n"))

	return sb.String()
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
