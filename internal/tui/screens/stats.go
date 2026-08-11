package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/stats"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// StatsScreen displays commit identity stats with toggleable sort modes (By Commits / By Lines).
type StatsScreen struct {
	store    *config.Store
	sortMode stats.SortMode
	items    []stats.AuthorStat
	offset   int
	maxLines int
	theme    theme.Theme
	err      error
}

func NewStatsScreen(store *config.Store, th theme.Theme) *StatsScreen {
	s := &StatsScreen{
		store:    store,
		sortMode: stats.SortByCommits,
		theme:    th,
	}
	s.reload()
	return s
}

func (s *StatsScreen) reload() {
	authorStats, err := stats.AuditRepositoryMode(s.store, "", s.sortMode)
	if err != nil {
		s.err = err
		s.items = nil
		return
	}
	s.err = nil
	s.items = authorStats
}

func (s *StatsScreen) Init() tea.Cmd { return nil }

func (s *StatsScreen) Title() string { return "Commit Identity Audit" }

func (s *StatsScreen) ShortHelp() string {
	return "  ←/→ toggle view (Commits/Lines) • ↑/↓ scroll • Esc back • q quit"
}

func (s *StatsScreen) maxScrollOffset() int {
	linesCount := len(s.items)
	if linesCount == 0 {
		linesCount = 1
	}
	max := linesCount - s.maxLines
	if max < 0 {
		max = 0
	}
	return max
}

func (s *StatsScreen) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
			return s, func() tea.Msg { return core.ScreenPopMsg{} }
		}
		switch msg.String() {
		case core.KeyCtrlC, core.KeyQuit:
			return s, tea.Quit

		case core.KeyLeft, core.KeyRight, core.KeyH, core.KeyL, core.KeyTab:
			if s.sortMode == stats.SortByCommits {
				s.sortMode = stats.SortByLines
			} else {
				s.sortMode = stats.SortByCommits
			}
			s.reload()

		case core.KeyUp, core.KeyK:
			if s.offset > 0 {
				s.offset--
			}

		case core.KeyDown, core.KeyJ:
			if s.offset < s.maxScrollOffset() {
				s.offset++
			}

		case "ctrl+d", "pgdown":
			half := s.maxLines / 2
			if half < 1 {
				half = 1
			}
			s.offset += half
			if s.offset > s.maxScrollOffset() {
				s.offset = s.maxScrollOffset()
			}

		case "ctrl+u", "pgup":
			half := s.maxLines / 2
			if half < 1 {
				half = 1
			}
			s.offset -= half
			if s.offset < 0 {
				s.offset = 0
			}
		}
	}
	return s, nil
}

func (s *StatsScreen) View(width, height int) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(s.theme.Bold().Render("  " + s.Title()))
	sb.WriteString("\n")

	// Render mode toggle tab header
	modeCommits := " [ Sorted by Commits ] "
	modeLines := " [ Sorted by Code Lines ] "
	if s.sortMode == stats.SortByCommits {
		modeCommits = s.theme.Selected().Render("▶" + modeCommits)
		modeLines = s.theme.Dim().Render(modeLines)
	} else {
		modeCommits = s.theme.Dim().Render(modeCommits)
		modeLines = s.theme.Selected().Render("▶" + modeLines)
	}

	sb.WriteString("  " + modeCommits + "  " + modeLines)
	sb.WriteString("\n")
	sb.WriteString(s.theme.SeparatorLine(width - 6))
	sb.WriteString("\n\n")

	if s.err != nil {
		sb.WriteString("  " + s.theme.ErrorStyle().Render("Error: "+s.err.Error()) + "\n")
	} else if len(s.items) == 0 {
		sb.WriteString("  " + s.theme.Dim().Render("No commits found in this repository.") + "\n")
	} else {
		maxLines := height - 10
		if maxLines < 3 {
			maxLines = 3
		}
		s.maxLines = maxLines

		maxOff := len(s.items) - maxLines
		if maxOff < 0 {
			maxOff = 0
		}
		if s.offset > maxOff {
			s.offset = maxOff
		}

		start := s.offset
		end := start + maxLines
		if end > len(s.items) {
			end = len(s.items)
		}

		hasUnregistered := false
		for i := start; i < end; i++ {
			item := s.items[i]
			statusStr := ""
			if item.VerifiedUser != nil {
				statusStr = s.theme.SuccessStyle().Render("✓ Verified (" + item.VerifiedUser.Name + ")")
			} else {
				statusStr = s.theme.ErrorStyle().Render("⚠ Unregistered (potential identity leak!)")
				hasUnregistered = true
			}

			if s.sortMode == stats.SortByLines {
				linesStr := fmt.Sprintf("+%d / -%d (Net: %+d)", item.CodeLinesAdded, item.CodeLinesDeleted, item.NetCodeLines)
				sb.WriteString(fmt.Sprintf("  %-25s  %-30s  Lines: %-22s  %s\n", item.DisplayName, fmt.Sprintf("<%s>", item.Email), linesStr, statusStr))
			} else {
				sb.WriteString(fmt.Sprintf("  %-25s  %-30s  Commits: %-5d  %s\n", item.DisplayName, fmt.Sprintf("<%s>", item.Email), item.Commits, statusStr))
			}
		}

		sb.WriteString("\n")
		if hasUnregistered {
			sb.WriteString("  " + s.theme.WarningStyle().Render("Unregistered authors found in commit history! Register identity to verify.") + "\n")
		} else {
			sb.WriteString("  " + s.theme.SuccessStyle().Render("All commit authors in history match registered identities.") + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(s.theme.Dim().Render("  ←/→ switch view mode • Esc back"))
	sb.WriteString("\n")

	return sb.String()
}
