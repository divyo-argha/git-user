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

// StatsScreen displays commit identity stats with interactive pointer selection
// and detailed breakdown cards for the focused author.
type StatsScreen struct {
	store         *config.Store
	sortMode      stats.SortMode
	items         []stats.AuthorStat
	selectedIndex int
	offset        int
	maxLines      int
	theme         theme.Theme
	err           error
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
	if s.selectedIndex >= len(s.items) {
		s.selectedIndex = len(s.items) - 1
	}
	if s.selectedIndex < 0 && len(s.items) > 0 {
		s.selectedIndex = 0
	}
}

func (s *StatsScreen) Init() tea.Cmd { return nil }

func (s *StatsScreen) Title() string { return "Commit Identity Audit" }

func (s *StatsScreen) ShortHelp() string {
	return "  ↑/↓ select author • ←/→ toggle view (Commits/Lines) • Esc back • q quit"
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

		case core.KeyLeft, core.KeyRight, core.KeyTab:
			if s.sortMode == stats.SortByCommits {
				s.sortMode = stats.SortByLines
			} else {
				s.sortMode = stats.SortByCommits
			}
			s.reload()

		case core.KeyUp, core.KeyK:
			if s.selectedIndex > 0 {
				s.selectedIndex--
				if s.selectedIndex < s.offset {
					s.offset = s.selectedIndex
				}
			}

		case core.KeyDown, core.KeyJ:
			if s.selectedIndex < len(s.items)-1 {
				s.selectedIndex++
				if s.selectedIndex >= s.offset+s.maxLines {
					s.offset = s.selectedIndex - s.maxLines + 1
				}
			}

		case "ctrl+d", "pgdown":
			half := s.maxLines / 2
			if half < 1 {
				half = 1
			}
			s.selectedIndex += half
			if s.selectedIndex >= len(s.items) {
				s.selectedIndex = len(s.items) - 1
			}
			if s.selectedIndex < 0 {
				s.selectedIndex = 0
			}
			if s.selectedIndex >= s.offset+s.maxLines {
				s.offset = s.selectedIndex - s.maxLines + 1
			}

		case "ctrl+u", "pgup":
			half := s.maxLines / 2
			if half < 1 {
				half = 1
			}
			s.selectedIndex -= half
			if s.selectedIndex < 0 {
				s.selectedIndex = 0
			}
			if s.selectedIndex < s.offset {
				s.offset = s.selectedIndex
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
		// Reserve room for header (6 lines), footer/help (3 lines), and focused breakdown panel (6 lines)
		maxLines := height - 15
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

		for i := start; i < end; i++ {
			item := s.items[i]
			isFocused := (i == s.selectedIndex)

			pointer := "  "
			if isFocused {
				pointer = s.theme.Selected().Render("▶ ")
			}

			// Clean list rendering: focus on clean metrics without inline status text pollution
			var lineStr string
			if s.sortMode == stats.SortByLines {
				metrics := fmt.Sprintf("+%d / -%d (Net: %+d)", item.CodeLinesAdded, item.CodeLinesDeleted, item.NetCodeLines)
				lineStr = fmt.Sprintf("%s%-24s %-32s Lines: %s", pointer, item.DisplayName, fmt.Sprintf("<%s>", item.Email), metrics)
			} else {
				lineStr = fmt.Sprintf("%s%-24s %-32s Commits: %-6d", pointer, item.DisplayName, fmt.Sprintf("<%s>", item.Email), item.Commits)
			}

			if isFocused {
				sb.WriteString(s.theme.Bold().Render(lineStr) + "\n")
			} else {
				sb.WriteString(lineStr + "\n")
			}
		}

		// Detailed breakdown panel for currently selected author profile
		if s.selectedIndex >= 0 && s.selectedIndex < len(s.items) {
			sel := s.items[s.selectedIndex]
			sb.WriteString("\n")
			sb.WriteString(s.theme.SeparatorLine(width - 6))
			sb.WriteString("\n")

			headerText := fmt.Sprintf("COMMIT SIGNATURE & IDENTITY AUDIT: %s <%s>", sel.DisplayName, sel.Email)
			sb.WriteString(s.theme.PaneTitle().Render("  "+headerText) + "\n")

			notSigned := sel.UnsignedCommits + sel.RevokedSignatureCommits + sel.BadSignatureCommits + sel.UnverifiableCommits

			// Cryptographic signature status — driven exclusively by git's own
			// %G? check. Independent of whether the author is a registered
			// git-user identity.
			signatureAudit := fmt.Sprintf("  • Cryptographic Signature : %s",
				func() string {
					if notSigned == 0 && sel.SignedCommits > 0 {
						return s.theme.SuccessStyle().Render(fmt.Sprintf("✓ Signed (%d/%d cryptographically signed)", sel.SignedCommits, sel.Commits))
					} else if sel.SignedCommits > 0 {
						return s.theme.WarningStyle().Render(fmt.Sprintf("⚠ Partially Signed (%d Signed, %d Not Signed)", sel.SignedCommits, notSigned))
					}
					return s.theme.ErrorStyle().Render(fmt.Sprintf("⚠ Not Signed (0/%d cryptographically signed)", sel.Commits))
				}(),
			)
			sb.WriteString(signatureAudit + "\n")

			if sel.RevokedSignatureCommits > 0 {
				sb.WriteString("      " + s.theme.ErrorStyle().Render(fmt.Sprintf("⚠ %d commit(s) signed by a since-revoked key — not trusted", sel.RevokedSignatureCommits)) + "\n")
			}
			if sel.BadSignatureCommits > 0 {
				sb.WriteString("      " + s.theme.ErrorStyle().Render(fmt.Sprintf("⚠ %d commit(s) with an invalid/corrupt signature", sel.BadSignatureCommits)) + "\n")
			}
			if sel.UnverifiableCommits > 0 {
				sb.WriteString("      " + s.theme.WarningStyle().Render(fmt.Sprintf("⚠ %d commit(s) unverifiable locally (no matching public key / allowedSignersFile configured here)", sel.UnverifiableCommits)) + "\n")
			}

			// Identity registration status — purely local, not a cryptographic
			// check. Independent of the signature status above.
			identityAudit := "  • Local Identity Match   : " + s.theme.ErrorStyle().Render("⚠ Unregistered (no matching git-user profile)")
			if sel.IsRegisteredIdentity() {
				identityAudit = "  • Local Identity Match   : " + s.theme.SuccessStyle().Render(fmt.Sprintf("✓ Registered (%s)", sel.VerifiedUser.Name))
			}
			sb.WriteString(identityAudit + "\n")

			commitBreakdown := fmt.Sprintf("  • Commits Breakdown       : Total: %d  |  %s  |  %s",
				sel.Commits,
				s.theme.SuccessStyle().Render(fmt.Sprintf("Signed: %d", sel.SignedCommits)),
				s.theme.ErrorStyle().Render(fmt.Sprintf("Not Signed: %d", notSigned)),
			)
			sb.WriteString(commitBreakdown + "\n")

			lineBreakdown := fmt.Sprintf("  • Code Lines Breakdown    : Net: %+d  |  %s  |  %s",
				sel.NetCodeLines,
				s.theme.SuccessStyle().Render(fmt.Sprintf("Signed: +%d/-%d", sel.SignedLinesAdded, sel.SignedLinesDeleted)),
				s.theme.ErrorStyle().Render(fmt.Sprintf("Not Signed: +%d/-%d", sel.UnsignedLinesAdded, sel.UnsignedLinesDeleted)),
			)
			sb.WriteString(lineBreakdown + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(s.theme.Dim().Render("  ↑/↓ select author • ←/→ switch view mode • Esc back"))
	sb.WriteString("\n")

	return sb.String()
}
