package components

import (
	"fmt"
	"strings"

	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/git"
	"github.com/divyo-argha/git-user/internal/tui/theme"
	"github.com/divyo-argha/git-user/internal/version"
	"github.com/divyo-argha/git-user/logo"
)

// StatusBar renders the top header bar with logo, active profile, and SSH agent status.
type StatusBar struct {
	store           *config.Store
	agentConnected  bool
	agentKeyCount   int
	agentChecked    bool
	latestVersion   string
	updateAvailable bool
	theme           theme.Theme
}

// NewStatusBar creates a new status bar component.
func NewStatusBar(store *config.Store, th theme.Theme) StatusBar {
	return StatusBar{store: store, theme: th}
}

// SetStore updates the config store reference.
func (s *StatusBar) SetStore(store *config.Store) { s.store = store }

// SetVersionStatus updates the remote version check status.
func (s *StatusBar) SetVersionStatus(latestVersion string, updateAvailable bool) {
	s.latestVersion = latestVersion
	s.updateAvailable = updateAvailable
}

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

	versionLine := "  " + s.theme.Subtle().Render(fmt.Sprintf("Version %s", version.GetVersion()))
	if s.updateAvailable && s.latestVersion != "" {
		versionLine += "  " + s.theme.PillWarning().Render(fmt.Sprintf("Update available: %s", s.latestVersion))
	}

	labelStyle := s.theme.PaneTitle()

	var infoLines []string

	if s.store != nil && s.store.Current != "" {
		if u := s.store.CurrentUser(); u != nil {
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s %s",
				labelStyle.Render("Active profile :"),
				s.theme.Active().Render("●"),
				s.theme.Active().Render(u.Name)+" "+s.theme.Subtle().Render("("+u.Email+")"),
			))
		} else {
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
				labelStyle.Render("Active profile :"),
				s.theme.DangerText().Render(s.store.Current+" (missing)"),
			))
		}
	} else {
		infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Active profile :"),
			s.theme.Dim().Render("None (logged out)"),
		))
	}

	if s.agentChecked {
		if s.agentConnected {
			agentStr := s.theme.SuccessStyle().Render("Connected")
			countStr := s.theme.Subtle().Render(fmt.Sprintf("(%d keys loaded)", s.agentKeyCount))
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s %s",
				labelStyle.Render("SSH Agent      :"),
				agentStr,
				countStr,
			))
		} else {
			agentStr := s.theme.DangerText().Render("Not reachable")
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
				labelStyle.Render("SSH Agent      :"),
				agentStr,
			))
		}
	} else {
		infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("SSH Agent      :"),
			s.theme.Dim().Render("checking..."),
		))
	}

	repoName := git.CurrentRepoName()
	branch := git.CurrentBranch()
	if repoName != "" {
		repoStr := s.theme.Selected().Render(repoName)
		if branch != "" {
			repoStr += " " + s.theme.Subtle().Render("("+branch+")")
		}
		infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Repository     :"),
			repoStr,
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
	if s.updateAvailable && s.latestVersion != "" {
		header += "  " + s.theme.PillWarning().Render("Update: "+s.latestVersion)
	}
	if s.store != nil && s.store.Current != "" {
		if u := s.store.CurrentUser(); u != nil {
			header += "  " + s.theme.Dim().Render("active: "+u.Name+" ("+u.Email+")")
		}
	}
	return header
}
