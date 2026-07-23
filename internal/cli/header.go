package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/ssh"
	"github.com/divyo-argha/git-user/internal/version"
	"github.com/divyo-argha/git-user/logo"
)

var (
	tuiDim  = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	tuiBold = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
)

// ── Header style selector ─────────────────────────────────────────────────────
//
// Switch between header styles by changing this constant:
//
//	headerStyle = "logo"   → pixel-art logo + text side by side
//	headerStyle = "text"   → plain bold text (original)
const headerStyle = "logo"

// Choose which logo to display:
//
//	true  → new logo from logo.png (logo.NewSmallPixelLines)
//	false → original logo from git-userhub-logo.png (logo.SmallPixelLines)
const useNewLogo = true

// RenderHeader returns the TUI header block for the main screen.
func renderHeader(store *config.Store, termHeight int) string {
	style := headerStyle
	// If terminal is under 15 lines, use text fallback
	if termHeight > 0 && termHeight < 15 {
		style = "text"
	}

	switch style {
	case "logo":
		return renderLogoHeader(store)
	default:
		return renderTextHeader(store)
	}
}

// ── Logo header ───────────────────────────────────────────────────────────────

func renderLogoHeader(store *config.Store) string {
	logoLines := logo.GetTrimmedLogo()

	versionLine := fmt.Sprintf("  \x1b[38;2;148;163;184mVersion %s\x1b[0m", version.Version)

	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A"))
	actName := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")).Bold(true)
	actEmail := lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true)

	var infoLines []string

	// Active profile details
	if store.Current != "" {
		if u := store.CurrentUser(); u != nil {
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s %s",
				labelStyle.Render("Active profile :"),
				dotStyle.Render("●"),
				actName.Render(u.Name)+" "+actEmail.Render("("+u.Email+")"),
			))
		} else {
			infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
				labelStyle.Render("Active profile :"),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")).Render(store.Current+" (missing)"),
			))
		}
	} else {
		infoLines = append(infoLines, fmt.Sprintf("  %s  %s",
			labelStyle.Render("Active profile :"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("None (logged out)"),
		))
	}

	// SSH Agent Connection
	_, conn, err := ssh.GetAgentClient()
	if err == nil {
		defer conn.Close()
		agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")).Render("Connected")
		keyCount := 0
		if fingerprints, errList := ssh.LoadedSSHKeyFingerprints(); errList == nil {
			keyCount = len(fingerprints)
		}
		countStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99")).Render(fmt.Sprintf("(%d keys loaded)", keyCount))
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

	var sb strings.Builder
	sb.WriteString(strings.Join(logoLines, "\n"))
	sb.WriteString("\n")
	sb.WriteString(versionLine)
	sb.WriteString("\n\n")
	sb.WriteString(strings.Join(infoLines, "\n"))

	return sb.String()
}

// ── Text header (original) ────────────────────────────────────────────────────

func renderTextHeader(store *config.Store) string {
	header := tuiBold.Render("  git-user")
	if store.Current != "" {
		if u := store.CurrentUser(); u != nil {
			header += "  " + tuiDim.Render("active: "+u.Name+" ("+u.Email+")")
		}
	}
	return header
}
