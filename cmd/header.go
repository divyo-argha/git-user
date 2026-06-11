package cmd

import (
	"fmt"
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

// Choose which logo to display:
//   true  → new logo from logo.png (logo.NewSmallPixelLines)
//   false → original logo from git-userhub-logo.png (logo.SmallPixelLines)
const useNewLogo = true

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
	var logoLines []string
	if useNewLogo {
		logoLines = logo.NewSmallPixelLines
	} else {
		logoLines = logo.SmallPixelLines
	}
	logoH := len(logoLines)

	nameStyle  := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFAA")).Bold(true)
	tagStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("#666688")).Italic(true)
	dotStyle   := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF99"))
	actName    := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF99")).Bold(true)
	actEmail   := lipgloss.NewStyle().Foreground(lipgloss.Color("#8888AA"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#777799"))

	rightLines := []string{
		nameStyle.Render("GIT-USER"),
		tagStyle.Render("switch git identities in one command"),
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
				lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render(store.Current+" (missing)"),
			))
		}
	} else {
		rightLines = append(rightLines, fmt.Sprintf("%s  %s", 
			labelStyle.Render("Active profile :"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("None (logged out)"),
		))
	}

	// Agent status
	client, conn, err := getAgentClient()
	if err == nil {
		conn.Close()
		_ = client
		agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF66")).Render("Connected")
		keyCount := 0
		if fingerprints, errList := loadedSSHKeyFingerprints(); errList == nil {
			keyCount = len(fingerprints)
		}
		rightLines = append(rightLines, fmt.Sprintf("%s  %s (%d keys loaded)", 
			labelStyle.Render("SSH Agent      :"),
			agentStr,
			keyCount,
		))
	} else {
		agentStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("Not reachable")
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
