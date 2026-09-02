package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

// LatestVersion returns the latest known remote version string.
func (s StatusBar) LatestVersion() string { return s.latestVersion }

// UpdateAvailable reports whether a newer remote version is available.
func (s StatusBar) UpdateAvailable() bool { return s.updateAvailable }

// SetAgentStatus updates the SSH agent status.
func (s *StatusBar) SetAgentStatus(connected bool, keyCount int) {
	s.agentConnected = connected
	s.agentKeyCount = keyCount
	s.agentChecked = true
}

// View renders the status bar.
func (s StatusBar) View(width, termHeight int) string {
	if termHeight > 0 && termHeight < theme.StatusBarCompactBreakpoint {
		return s.viewCompact()
	}
	return s.viewFull(width)
}

func (s StatusBar) viewFull(width int) string {
	logoLines := logo.GetTrimmedLogo()

	versionLine := "  " + s.theme.Subtle().Render(fmt.Sprintf("Version %s", version.GetVersion()))
	if s.updateAvailable && s.latestVersion != "" {
		versionLine += "  " + s.theme.PillWarning().Render(fmt.Sprintf("Update available: %s", s.latestVersion))
	}

	infoLine := "  " + strings.Join(s.buildInfoSegments(), s.theme.Dim().Render("  ·  "))

	var sb strings.Builder
	sb.WriteString(strings.Join(logoLines, "\n"))
	sb.WriteString("\n")
	sb.WriteString(versionLine)
	sb.WriteString("\n")
	maxW := width - 2
	if maxW < 8 {
		maxW = 8
	}
	sb.WriteString(lipgloss.NewStyle().MaxWidth(maxW).Render(infoLine))

	return sb.String()
}

// buildInfoSegments assembles the active-profile, SSH-agent, and (if in a
// git repo) repository segments shown on the status bar's single info line.
func (s StatusBar) buildInfoSegments() []string {
	var segments []string

	if s.store != nil && s.store.Current != "" {
		if u := s.store.CurrentUser(); u != nil {
			segments = append(segments, s.theme.Active().Render("●")+" "+
				s.theme.Active().Render(u.Name)+" "+s.theme.Subtle().Render("("+u.Email+")"))
		} else {
			segments = append(segments, s.theme.DangerText().Render(s.store.Current+" (missing)"))
		}
	} else {
		segments = append(segments, s.theme.Dim().Render("No active profile"))
	}

	if s.agentChecked {
		if s.agentConnected {
			segments = append(segments, "SSH: "+s.theme.SuccessStyle().Render("Connected")+
				" "+s.theme.Subtle().Render(fmt.Sprintf("(%d keys)", s.agentKeyCount)))
		} else {
			segments = append(segments, "SSH: "+s.theme.DangerText().Render("Not reachable"))
		}
	} else {
		segments = append(segments, s.theme.Dim().Render("SSH: checking..."))
	}

	repoName := git.CurrentRepoName()
	branch := git.CurrentBranch()
	if repoName != "" {
		repoStr := s.theme.Selected().Render(repoName)
		if branch != "" {
			repoStr += " " + s.theme.Subtle().Render("("+branch+")")
		}
		segments = append(segments, repoStr)
	}

	return segments
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
