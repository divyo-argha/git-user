package theme

import "github.com/charmbracelet/lipgloss"

// ── Color Palette ─────────────────────────────────────────────────────────────
// Single source of truth for all TUI colors. No other file should define colors.

type Theme struct {
	// Semantic colors
	Primary     lipgloss.Color // Cyan — main accent, active borders, highlights
	Secondary   lipgloss.Color // Green — success states, active identity
	Accent      lipgloss.Color // Magenta — decorative touches
	Danger      lipgloss.Color // Red — destructive actions, errors
	Warning     lipgloss.Color // Yellow/Orange — warnings
	Muted       lipgloss.Color // Gray — disabled, separators, dim text
	Text        lipgloss.Color // White — primary text
	TextDim     lipgloss.Color // Lighter gray — secondary text
	Background  lipgloss.Color // For cards and panels (not terminal bg)

	// Derived styles (computed once)
	styles themeStyles
}

type themeStyles struct {
	// Text styles
	Bold       lipgloss.Style
	Dim        lipgloss.Style
	Italic     lipgloss.Style
	Success    lipgloss.Style
	Error      lipgloss.Style
	Warning    lipgloss.Style
	Info       lipgloss.Style
	DangerText lipgloss.Style

	// Cursor / selection
	Selected lipgloss.Style
	Active   lipgloss.Style

	// Pane titles
	PaneTitle lipgloss.Style

	// Section headers (inside action menus)
	SectionHeader lipgloss.Style

	// Separator line
	Separator lipgloss.Style
}

// DefaultTheme returns the standard git-user color scheme.
func DefaultTheme() Theme {
	t := Theme{
		Primary:    lipgloss.Color("#00FFFF"),
		Secondary:  lipgloss.Color("#00FF00"),
		Accent:     lipgloss.Color("#FF00FF"),
		Danger:     lipgloss.Color("#FF5555"),
		Warning:    lipgloss.Color("#FFAA00"),
		Muted:      lipgloss.Color("#555555"),
		Text:       lipgloss.Color("#FFFFFF"),
		TextDim:    lipgloss.Color("#888899"),
		Background: lipgloss.Color("#1a1a2e"),
	}
	t.styles = t.buildStyles()
	return t
}

func (t Theme) buildStyles() themeStyles {
	return themeStyles{
		Bold:       lipgloss.NewStyle().Foreground(t.Text).Bold(true),
		Dim:        lipgloss.NewStyle().Foreground(t.Muted),
		Italic:     lipgloss.NewStyle().Foreground(t.TextDim).Italic(true),
		Success:    lipgloss.NewStyle().Foreground(t.Secondary).Bold(true),
		Error:      lipgloss.NewStyle().Foreground(t.Danger).Bold(true),
		Warning:    lipgloss.NewStyle().Foreground(t.Warning),
		Info:       lipgloss.NewStyle().Foreground(t.Primary),
		DangerText: lipgloss.NewStyle().Foreground(t.Danger),

		Selected: lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		Active:   lipgloss.NewStyle().Foreground(t.Secondary).Bold(true),

		PaneTitle:     lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		SectionHeader: lipgloss.NewStyle().Foreground(t.TextDim).Italic(true),

		Separator: lipgloss.NewStyle().Foreground(t.Muted),
	}
}

// ── Style Accessors ───────────────────────────────────────────────────────────

func (t Theme) Bold() lipgloss.Style       { return t.styles.Bold }
func (t Theme) Dim() lipgloss.Style        { return t.styles.Dim }
func (t Theme) ItalicStyle() lipgloss.Style { return t.styles.Italic }
func (t Theme) SuccessStyle() lipgloss.Style { return t.styles.Success }
func (t Theme) ErrorStyle() lipgloss.Style  { return t.styles.Error }
func (t Theme) WarningStyle() lipgloss.Style { return t.styles.Warning }
func (t Theme) InfoStyle() lipgloss.Style   { return t.styles.Info }
func (t Theme) DangerText() lipgloss.Style  { return t.styles.DangerText }
func (t Theme) Selected() lipgloss.Style    { return t.styles.Selected }
func (t Theme) Active() lipgloss.Style      { return t.styles.Active }
func (t Theme) PaneTitle() lipgloss.Style   { return t.styles.PaneTitle }
func (t Theme) SectionHeader() lipgloss.Style { return t.styles.SectionHeader }
func (t Theme) Separator() lipgloss.Style   { return t.styles.Separator }

// ── Dynamic Pane Styles ───────────────────────────────────────────────────────
// These accept width/height so they adapt to terminal size.

func (t Theme) ActivePane(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(0, 2).
		Width(width).
		Height(height)
}

func (t Theme) InactivePane(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Muted).
		Padding(0, 2).
		Width(width).
		Height(height)
}

func (t Theme) DetailCardActive(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Secondary).
		Padding(0, 2).
		Width(width).
		Height(height)
}

func (t Theme) DetailCardInactive(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Muted).
		Padding(0, 2).
		Width(width).
		Height(height)
}

func (t Theme) ActionPane(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(0, 2).
		Width(width).
		Height(height)
}

// ── Toast Style Type ──────────────────────────────────────────────────────────

// ToastStyleKind defines the visual style of a toast notification.
type ToastStyleKind int

const (
	ToastStyleSuccess ToastStyleKind = iota
	ToastStyleError
	ToastStyleInfo
)

// ── Toast Styles ──────────────────────────────────────────────────────────────

func (t Theme) ToastSuccess(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Secondary).
		Padding(0, 2).
		Width(width)
}

func (t Theme) ToastError(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Danger).
		Padding(0, 2).
		Width(width)
}

func (t Theme) ToastInfo(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(0, 2).
		Width(width)
}

// ── Layout Helpers ────────────────────────────────────────────────────────────

const (
	// MinTermWidth is the minimum terminal width before switching to single-column mode.
	MinTermWidth = 60
	// PaneGap is the horizontal gap between side-by-side panes.
	PaneGap = 3
	// StatusBarHeight is the number of lines reserved for the status bar.
	StatusBarHeight = 5
	// HelpBarHeight is the number of lines reserved for the help footer.
	HelpBarHeight = 2
	// ChromeHeight is total lines consumed by status bar + help bar + margins.
	ChromeHeight = StatusBarHeight + HelpBarHeight + 3
)

// PaneWidth calculates the width for each pane in a two-column layout.
// It accounts for borders (2 chars each side), padding (2 chars each side from Padding(0,2)),
// and the gap between panes.
func PaneWidth(termWidth int) int {
	// Each pane has: 2 border chars + 4 padding chars (2 each side) = 6 extra chars
	// Total: 2 panes * 6 + gap = 12 + gap
	usable := termWidth - PaneGap
	if usable < 20 {
		return 20
	}
	return usable / 2
}

// ContentHeight calculates the available height for screen content.
func ContentHeight(termHeight int) int {
	h := termHeight - ChromeHeight
	if h < 5 {
		return 5
	}
	return h
}

// IsSingleColumn returns true if the terminal is too narrow for side-by-side panes.
func IsSingleColumn(termWidth int) bool {
	return termWidth < MinTermWidth
}

// SeparatorLine returns a dim horizontal rule fitting the given width.
func (t Theme) SeparatorLine(width int) string {
	if width <= 0 {
		width = 40
	}
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return t.styles.Separator.Render(line)
}
