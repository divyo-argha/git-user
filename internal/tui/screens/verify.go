package screens

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/stats"
	"github.com/divyo-argha/git-user/internal/tui/components"
	"github.com/divyo-argha/git-user/internal/tui/core"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

type verifyLoadedMsg struct {
	rangeUsed string
	items     []stats.AuthorStat
	err       error
}

// VerifyScreen runs a signature-verification pass over a commit range,
// mirroring `git-user verify` — same defaults, same underlying
// stats.VerifyRange call.
type VerifyScreen struct {
	store     *config.Store
	rangeStr  string // "" = auto-resolve, same as the CLI's resolveDefaultRange
	rangeUsed string
	items     []stats.AuthorStat
	loading   bool
	err       error
	theme     theme.Theme
	spinner   components.Spinner
}

func NewVerifyScreen(store *config.Store, th theme.Theme) *VerifyScreen {
	return &VerifyScreen{
		store:   store,
		theme:   th,
		spinner: components.NewSpinner(th),
	}
}

const verifyDefaultDepth = 50

func resolveDefaultVerifyRange() string {
	n := fmt.Sprintf("HEAD~%d", verifyDefaultDepth)
	if err := exec.Command("git", "rev-parse", "--verify", "--quiet", n).Run(); err != nil {
		return ""
	}
	return n + "..HEAD"
}

func (v *VerifyScreen) loadCmd() tea.Cmd {
	store := v.store
	rangeStr := v.rangeStr
	return func() tea.Msg {
		effectiveRange := rangeStr
		if effectiveRange == "" {
			effectiveRange = resolveDefaultVerifyRange()
		}
		items, err := stats.VerifyRange(store, effectiveRange)
		display := effectiveRange
		if display == "" {
			display = "full history"
		}
		return verifyLoadedMsg{rangeUsed: display, items: items, err: err}
	}
}

func (v *VerifyScreen) startLoad() tea.Cmd {
	v.loading = true
	return tea.Batch(v.spinner.Init(), v.loadCmd())
}

func (v *VerifyScreen) Init() tea.Cmd { return v.startLoad() }

func (v *VerifyScreen) Title() string { return "Verify Commit Signatures" }

func (v *VerifyScreen) ShortHelp() string {
	return "e•edit range  r•refresh  Esc/b•back"
}

func (v *VerifyScreen) Update(msg tea.Msg) (core.Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case verifyLoadedMsg:
		v.loading = false
		v.rangeUsed = msg.rangeUsed
		v.items = msg.items
		v.err = msg.err
		return v, nil

	case tea.KeyMsg:
		if v.loading {
			if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
				return v, func() tea.Msg { return core.ScreenPopMsg{} }
			}
			if msg.String() == core.KeyCtrlC {
				return v, tea.Quit
			}
			return v, nil
		}
		if core.IsEscKey(msg) || msg.String() == "b" || msg.String() == "B" {
			return v, func() tea.Msg { return core.ScreenPopMsg{} }
		}
		switch msg.String() {
		case core.KeyCtrlC:
			return v, tea.Quit
		case core.KeyQuit:
			return v, func() tea.Msg { return core.ActionResultMsg{Kind: "quit-confirm"} }
		case "r":
			return v, v.startLoad()
		case "e":
			return v, func() tea.Msg { return core.ActionResultMsg{Kind: "verify-set-range"} }
		}

	default:
		if v.loading {
			var cmd tea.Cmd
			v.spinner, cmd = v.spinner.Update(msg)
			return v, cmd
		}
	}
	return v, nil
}

// SetRange is called by the app layer after the range-edit form submits.
func (v *VerifyScreen) SetRange(r string) tea.Cmd {
	v.rangeStr = r
	return v.startLoad()
}

func (v *VerifyScreen) View(width, height int) string {
	var sb strings.Builder

	sb.WriteString("\n  " + v.theme.Bold().Render(v.Title()) + "\n")

	if v.loading {
		sb.WriteString("  " + v.spinner.View() + " " + v.theme.Dim().Render("Verifying commit signatures...") + "\n")
		return sb.String()
	}

	sb.WriteString("  " + v.theme.Dim().Render("Range: "+v.rangeUsed) + "\n\n")

	if v.err != nil {
		sb.WriteString("  " + v.theme.ErrorStyle().Render("Error: "+v.err.Error()) + "\n")
		return sb.String()
	}
	if len(v.items) == 0 {
		sb.WriteString("  " + v.theme.Dim().Render("No commits found in this range.") + "\n")
		return sb.String()
	}

	totalNotSigned := 0
	for _, item := range v.items {
		notSigned := item.UnsignedCommits + item.RevokedSignatureCommits + item.BadSignatureCommits + item.UnverifiableCommits
		totalNotSigned += notSigned

		var sigStr string
		switch {
		case item.SignedCommits > 0 && notSigned == 0:
			sigStr = v.theme.SuccessStyle().Render(fmt.Sprintf("%s Signed (%d/%d)", theme.IconPass, item.SignedCommits, item.Commits))
		case item.SignedCommits > 0:
			sigStr = v.theme.WarningStyle().Render(fmt.Sprintf("%s Partially Signed (%d/%d)", theme.IconWarn, item.SignedCommits, item.Commits))
		default:
			sigStr = v.theme.ErrorStyle().Render(fmt.Sprintf("%s Not Signed (0/%d)", theme.IconFail, item.Commits))
		}

		sb.WriteString(fmt.Sprintf("  %-24s %-32s Commits: %-5d %s\n", item.DisplayName, fmt.Sprintf("<%s>", item.Email), item.Commits, sigStr))
	}

	sb.WriteString("\n")
	sb.WriteString(v.theme.SeparatorLine(width - 6))
	sb.WriteString("\n\n")

	if totalNotSigned > 0 {
		sb.WriteString("  " + v.theme.WarningStyle().Render(fmt.Sprintf("%d commit(s) unsigned or carrying an invalid/revoked/unverifiable signature.", totalNotSigned)) + "\n")
	} else {
		sb.WriteString("  " + v.theme.SuccessStyle().Render("All commits in range carry a valid, currently-trusted cryptographic signature.") + "\n")
	}

	return sb.String()
}
