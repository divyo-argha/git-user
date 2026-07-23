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
	logoH := len(logoLines)

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true)
	badgeStyle := lipgloss.NewStyle().Background(lipgloss.Color("#2E3440")).Foreground(lipgloss.Color("#7AA2F7")).Padding(0, 1).Bold(true)
	tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99")).Italic(true)
	dotStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A"))
	actName := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")).Bold(true)
	actEmail := lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true)

	topTitle := lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render("⚡ GIT-USER"), "  ", badgeStyle.Render(version.Version))

	rightLines := []string{
		topTitle,
		tagStyle.Render("switch git identities & ssh keys in one command"),
		"",
	}

	// Active profile details
	if store.Current != "" {
		if u := store.CurrentUser(); u != nil {
			rightLines = append(rightLines, fmt.Sprintf("%s  %s %s",
				labelStyle.Render("Active profile :"),
				dotStyle.Render("●"),
				actName.Render(u.Name)+" "+actEmail.Render("("+u.Email+")"),
			))
		} else {
			rightLines = append(rightLines, fmt.Sprintf("%s  %s",
				labelStyle.Render("Active profile :"),
				lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")).Render(store.Current+" (missing)"),
			))
		}
	} else {
		rightLines = append(rightLines, fmt.Sprintf("%s  %s",
			labelStyle.Render("Active profile :"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("None (logged out)"),
		))
	}

	// Agent status
	client, conn, err := ssh.GetAgentClient()
	if err == nil {
		conn.Close()
		_ = client
		agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")).Bold(true).Render("Connected")
		keyCount := 0
		if fingerprints, errList := ssh.LoadedSSHKeyFingerprints(); errList == nil {
			keyCount = len(fingerprints)
		}
		countStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#787C99")).Render(fmt.Sprintf("(%d keys loaded)", keyCount))
		rightLines = append(rightLines, fmt.Sprintf("%s  %s %s",
			labelStyle.Render("SSH Agent      :"),
			agentStr,
			countStr,
		))
	} else {
		agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")).Render("Not reachable")
		rightLines = append(rightLines, fmt.Sprintf("%s  %s",
			labelStyle.Render("SSH Agent      :"),
			agentStr,
		))
	}

	// vertically center text within logo height
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
