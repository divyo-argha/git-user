package cmd

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/logo"
)

// ── Header style selector ─────────────────────────────────────────────────────
//
// Switch between header styles by changing this constant:
//   headerStyle = "logo"   → pixel-art logo + text side by side
//   headerStyle = "text"   → plain bold text (original)
//
const headerStyle = "logo"

// RenderHeader returns the TUI header block for the main screen.
func renderHeader(store *config.Store) string {
	switch headerStyle {
	case "logo":
		return renderLogoHeader(store)
	default:
		return renderTextHeader(store)
	}
}

// ── Logo header ───────────────────────────────────────────────────────────────

func renderLogoHeader(store *config.Store) string {
	logoH := len(logo.PixelLines)

	nameStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFAA")).Bold(true)
	tagStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("#444466")).Italic(true)
	dotStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF99"))
	actName    := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF99")).Bold(true)
	actEmail   := lipgloss.NewStyle().Foreground(lipgloss.Color("#555577"))

	rightLines := []string{
		nameStyle.Render("git-user"),
		tagStyle.Render("switch git identities in one command"),
	}
	if store.Current != "" {
		if u := store.CurrentUser(); u != nil {
			rightLines = append(rightLines, "")
			rightLines = append(rightLines,
				dotStyle.Render("●")+" "+actName.Render(u.Name)+actEmail.Render(" · "+u.Email))
		}
	}

	// vertically center text within logo height
	padTop := (logoH - len(rightLines)) / 2
	if padTop < 0 {
		padTop = 0
	}
	rightBlock := strings.Repeat("\n", padTop) + strings.Join(rightLines, "\n")

	return lipgloss.JoinHorizontal(lipgloss.Top,
		strings.Join(logo.PixelLines, "\n"),
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
