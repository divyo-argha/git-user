package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/diagnostics"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type healthLoadedMsg struct {
	report diagnostics.Report
	err    error
}

type healthRowKind int

const (
	healthRowSection healthRowKind = iota
	healthRowSubsection
	healthRowCheck
)

type healthRow struct {
	kind  healthRowKind
	label string
	check diagnostics.Check
}

// Health is the structured replacement for the old plain-text doctor report:
// grouped, colored, navigable checks built on internal/diagnostics — the
// same package internal/cli/doctor.go uses, so the CLI and TUI can never
// disagree about what's healthy.
type Health struct {
	store      *config.Store
	report     diagnostics.Report
	rows       []healthRow
	selectable []int // indices into rows that the cursor can land on
	cursor     int
	offset     int
	maxLines   int
	expanded   map[int]bool
	loading    bool
	fixing     bool
	err        error
	theme      theme.Theme
	spinner    components.Spinner
}

func NewHealth(store *config.Store, th theme.Theme) *Health {
	return &Health{
		store:    store,
		theme:    th,
		spinner:  components.NewSpinner(th),
		expanded: map[int]bool{},
	}
}

func verifySSHForHealth(keyPath string) error {
	for _, p := range ssh.DefaultPlatforms {
		if ssh.CheckPlatformConnection(keyPath, p.Name, p.Host, p.Patterns).Status == "connected" {
			return nil
		}
	}
	return fmt.Errorf("connection failed on all platforms")
}

func (h *Health) loadCmd(fix bool) tea.Cmd {
	store := h.store
	return func() tea.Msg {
		report, err := diagnostics.Run(store, diagnostics.Options{Fix: fix, VerifySSH: verifySSHForHealth})
		return healthLoadedMsg{report: report, err: err}
	}
}

func (h *Health) startLoad(fix bool) tea.Cmd {
	h.loading = true
	h.fixing = fix
	return tea.Batch(h.spinner.Init(), h.loadCmd(fix))
}

func (h *Health) Init() tea.Cmd { return h.startLoad(false) }

func (h *Health) Title() string { return "Health & Security" }

func (h *Health) ShortHelp() string {
	return "↑/↓/j/k•navigate  ⏎•expand fix hint  f•fix all now  r•refresh  Esc/b•back"
}

func buildHealthRows(checks []diagnostics.Check) ([]healthRow, []int) {
	var rows []healthRow
	var selectable []int
	currentCategory := ""
	currentSubject := ""

	for _, c := range checks {
		if c.IsProgress {
			continue
		}
		if c.Category != currentCategory {
			rows = append(rows, healthRow{kind: healthRowSection, label: strings.ToUpper(c.Category)})
			currentCategory = c.Category
			currentSubject = ""
		}
		if c.Subject != "" {
			if c.Subject != currentSubject {
				rows = append(rows, healthRow{kind: healthRowSubsection, label: c.Subject})
				currentSubject = c.Subject
			}
		} else {
			currentSubject = ""
		}
		selectable = append(selectable, len(rows))
		rows = append(rows, healthRow{kind: healthRowCheck, check: c})
	}

	return rows, selectable
}

func (h *Health) cursorRowIndex() int {
	if h.cursor < 0 || h.cursor >= len(h.selectable) {
		return -1
	}
	return h.selectable[h.cursor]
}

func (h *Health) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case healthLoadedMsg:
		h.loading = false
		h.fixing = false
		h.expanded = map[int]bool{}
		if msg.err != nil {
			h.err = msg.err
			return h, nil
		}
		h.err = nil
		h.report = msg.report
		h.rows, h.selectable = buildHealthRows(h.report.Checks)
		if h.cursor >= len(h.selectable) {
			h.cursor = len(h.selectable) - 1
		}
		if h.cursor < 0 && len(h.selectable) > 0 {
			h.cursor = 0
		}
		return h, nil

	case tea.KeyMsg:
		if h.loading {
			if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
				return h, func() tea.Msg { return core.ScreenPopMsg{} }
			}
			if msg.String() == core.KeyCtrlC {
				return h, tea.Quit
			}
			return h, nil
		}
		if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
			return h, func() tea.Msg { return core.ScreenPopMsg{} }
		}
		switch msg.String() {
		case core.KeyCtrlC:
			return h, tea.Quit
		case core.KeyQuit:
			return h, func() tea.Msg { return core.ActionResultMsg{Kind: "quit-confirm"} }

		case "f":
			return h, h.startLoad(true)

		case "r":
			return h, h.startLoad(false)

		case "enter", " ":
			if idx := h.cursorRowIndex(); idx >= 0 {
				h.expanded[idx] = !h.expanded[idx]
			}

		case core.KeyUp, core.KeyK:
			if h.cursor > 0 {
				h.cursor--
			}

		case core.KeyDown, core.KeyJ:
			if h.cursor < len(h.selectable)-1 {
				h.cursor++
			}
		}

	default:
		if h.loading {
			var cmd tea.Cmd
			h.spinner, cmd = h.spinner.Update(msg)
			return h, cmd
		}
	}
	return h, nil
}

func statusGlyph(th theme.Theme, status diagnostics.Status) string {
	switch status {
	case diagnostics.StatusPass:
		return th.SuccessStyle().Render(theme.IconPass)
	case diagnostics.StatusWarn, diagnostics.StatusNotice:
		return th.WarningStyle().Render(theme.IconWarn)
	case diagnostics.StatusInfo:
		return th.Dim().Render(theme.IconInfo)
	default:
		return th.Dim().Render(theme.IconSkipped)
	}
}

func scoreBand(passed, total int) string {
	if total == 0 {
		return ""
	}
	pct := passed * 100 / total
	switch {
	case pct >= 90:
		return "green"
	case pct >= 60:
		return "amber"
	default:
		return "red"
	}
}

func (h *Health) renderScoreHeader() string {
	if h.report.ScoreTotal == 0 {
		return ""
	}
	band := scoreBand(h.report.ScorePassed, h.report.ScoreTotal)
	label := fmt.Sprintf(" SECURITY SCORE  %d/%d ", h.report.ScorePassed, h.report.ScoreTotal)
	var pill string
	switch band {
	case "green":
		pill = h.theme.PillActive().Render(label)
	case "amber":
		pill = h.theme.PillWarning().Render(label)
	default:
		pill = h.theme.PillDanger().Render(label)
	}
	return pill
}

func (h *Health) View(width, height int) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("  " + h.theme.Bold().Render(h.Title()))
	if score := h.renderScoreHeader(); score != "" {
		sb.WriteString("   " + score)
	}
	sb.WriteString("\n")
	if h.report.ScoreTotal > 0 {
		issueWord := "issues"
		if h.report.Issues == 1 {
			issueWord = "issue"
		}
		sb.WriteString("  " + h.theme.Dim().Render(fmt.Sprintf("%d %s found", h.report.Issues, issueWord)) + "\n")
	}
	sb.WriteString(h.theme.SeparatorLine(width - 4))
	sb.WriteString("\n\n")

	if h.loading {
		verb := "Running diagnostics"
		if h.fixing {
			verb = "Fixing what can be fixed"
		}
		sb.WriteString("  " + h.spinner.View() + " " + h.theme.Dim().Render(verb+"...") + "\n")
		return sb.String()
	}
	if h.err != nil {
		sb.WriteString("  " + h.theme.ErrorStyle().Render("Error: "+h.err.Error()) + "\n")
		return sb.String()
	}
	if len(h.rows) == 0 {
		sb.WriteString("  " + h.theme.Dim().Render("No checks to show.") + "\n")
		return sb.String()
	}

	maxLines := height - 8
	if maxLines < 3 {
		maxLines = 3
	}
	h.maxLines = maxLines

	curRow := h.cursorRowIndex()
	if curRow >= 0 {
		if curRow < h.offset {
			h.offset = curRow
		}
		if curRow >= h.offset+maxLines {
			h.offset = curRow - maxLines + 1
		}
	}
	maxOff := len(h.rows) - maxLines
	if maxOff < 0 {
		maxOff = 0
	}
	if h.offset > maxOff {
		h.offset = maxOff
	}
	if h.offset < 0 {
		h.offset = 0
	}

	start := h.offset
	end := start + maxLines
	if end > len(h.rows) {
		end = len(h.rows)
	}

	for i := start; i < end; i++ {
		row := h.rows[i]
		switch row.kind {
		case healthRowSection:
			sb.WriteString("\n  " + h.theme.SectionHeader().Render(row.label) + "\n")
		case healthRowSubsection:
			sb.WriteString("    " + h.theme.Dim().Render("Profile: "+row.label) + "\n")
		case healthRowCheck:
			c := row.check
			indent := "    "
			if c.Subject != "" {
				indent = "      "
			}
			pointer := "  "
			if i == curRow {
				pointer = h.theme.Selected().Render("▶ ")
			}
			line := fmt.Sprintf("%s%s%s %s", indent, pointer, statusGlyph(h.theme, c.Status), c.Message)
			if i == curRow {
				line = h.theme.Bold().Render(line)
			}
			sb.WriteString(line + "\n")

			if h.expanded[i] {
				for _, d := range c.Detail {
					sb.WriteString(indent + "    " + h.theme.Dim().Render(strings.TrimSpace(d)) + "\n")
				}
				if c.FixHint != "" {
					sb.WriteString(indent + "    " + h.theme.InfoStyle().Render("Fix: "+c.FixHint) + "\n")
				}
			}
		}
	}

	return sb.String()
}
