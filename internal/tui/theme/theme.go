package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Danger     lipgloss.Color
	Warning    lipgloss.Color
	Muted      lipgloss.Color
	Text       lipgloss.Color
	TextDim    lipgloss.Color
	Background lipgloss.Color

	styles themeStyles
}

type themeStyles struct {
	Bold          lipgloss.Style
	Dim           lipgloss.Style
	Subtle        lipgloss.Style
	Italic        lipgloss.Style
	Success       lipgloss.Style
	Error         lipgloss.Style
	Warning       lipgloss.Style
	Info          lipgloss.Style
	DangerText    lipgloss.Style
	Featured      lipgloss.Style
	Selected      lipgloss.Style
	Active        lipgloss.Style
	PaneTitle     lipgloss.Style
	SectionHeader lipgloss.Style
	Separator     lipgloss.Style
}

// DefaultTheme returns the standard git-user color scheme.
func DefaultTheme() Theme {
	t := Theme{
		Primary:    lipgloss.Color("#7AA2F7"),
		Secondary:  lipgloss.Color("#9ECE6A"),
		Accent:     lipgloss.Color("#BB9AF7"),
		Danger:     lipgloss.Color("#F7768E"),
		Warning:    lipgloss.Color("#E0AF68"),
		Muted:      lipgloss.Color("#565F89"),
		Text:       lipgloss.Color("#C0CAF5"),
		TextDim:    lipgloss.Color("#787C99"),
		Background: lipgloss.Color("#1F2335"),
	}
	t.styles = t.buildStyles()
	return t
}

func (t Theme) buildStyles() themeStyles {
	return themeStyles{
		Bold:       lipgloss.NewStyle().Foreground(t.Text).Bold(true),
		Dim:        lipgloss.NewStyle().Foreground(t.Muted),
		Subtle:     lipgloss.NewStyle().Foreground(t.TextDim),
		Italic:     lipgloss.NewStyle().Foreground(t.TextDim).Italic(true),
		Success:    lipgloss.NewStyle().Foreground(t.Secondary).Bold(true),
		Error:      lipgloss.NewStyle().Foreground(t.Danger).Bold(true),
		Warning:    lipgloss.NewStyle().Foreground(t.Warning),
		Info:       lipgloss.NewStyle().Foreground(t.Primary),
		DangerText: lipgloss.NewStyle().Foreground(t.Danger),
		Featured:   lipgloss.NewStyle().Foreground(t.Primary).Bold(true),

		Selected: lipgloss.NewStyle().Foreground(t.Accent).Bold(true),
		Active:   lipgloss.NewStyle().Foreground(t.Secondary).Bold(true),

		PaneTitle:     lipgloss.NewStyle().Foreground(t.Primary).Bold(true),
		SectionHeader: lipgloss.NewStyle().Foreground(t.TextDim).Bold(true),

		Separator: lipgloss.NewStyle().Foreground(t.Muted),
	}
}

func (t Theme) Bold() lipgloss.Style         { return t.styles.Bold }
func (t Theme) Dim() lipgloss.Style          { return t.styles.Dim }
func (t Theme) Subtle() lipgloss.Style       { return t.styles.Subtle }
func (t Theme) ItalicStyle() lipgloss.Style  { return t.styles.Italic }
func (t Theme) SuccessStyle() lipgloss.Style { return t.styles.Success }
func (t Theme) ErrorStyle() lipgloss.Style   { return t.styles.Error }
func (t Theme) WarningStyle() lipgloss.Style { return t.styles.Warning }
func (t Theme) InfoStyle() lipgloss.Style    { return t.styles.Info }
func (t Theme) DangerText() lipgloss.Style   { return t.styles.DangerText }
func (t Theme) Featured() lipgloss.Style     { return t.styles.Featured }
func (t Theme) Selected() lipgloss.Style     { return t.styles.Selected }
func (t Theme) Active() lipgloss.Style       { return t.styles.Active }
func (t Theme) PaneTitle() lipgloss.Style    { return t.styles.PaneTitle }
func (t Theme) SectionHeader() lipgloss.Style { return t.styles.SectionHeader }
func (t Theme) Separator() lipgloss.Style    { return t.styles.Separator }

var pulseColors = []string{"#7AA2F7", "#89B4FA", "#B4BEFE", "#89B4FA"}

func (t Theme) PulsingActivePane(width, height int, frame uint64) lipgloss.Style {
	c := pulseColors[frame%uint64(len(pulseColors))]
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(c)).
		Padding(0, 2).
		Width(width).
		Height(height)
}

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

func (t Theme) PillActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.Secondary).
		Foreground(lipgloss.Color("#15161E")).
		Padding(0, 1).
		Bold(true)
}

func (t Theme) PillBadge() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.Background).
		Foreground(t.Primary).
		Padding(0, 1)
}

func (t Theme) PillWarning() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.Warning).
		Foreground(lipgloss.Color("#15161E")).
		Padding(0, 1).
		Bold(true)
}

func (t Theme) PillDanger() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.Danger).
		Foreground(lipgloss.Color("#15161E")).
		Padding(0, 1).
		Bold(true)
}

func (t Theme) PillMuted() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.Muted).
		Foreground(t.Text).
		Padding(0, 1)
}

func (t Theme) HUDBox(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(0, 1).
		Width(width)
}

func (t Theme) Keycap() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#2E3440")).
		Foreground(t.Accent).
		Padding(0, 1).
		Bold(true)
}

type ToastStyleKind int

const (
	ToastStyleSuccess ToastStyleKind = iota
	ToastStyleError
	ToastStyleInfo
)

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

const (
	// MinTermWidth is the minimum terminal width before switching to single-column mode.
	MinTermWidth = 60
	// PaneGap is the horizontal gap between side-by-side panes.
	PaneGap = 3
	// PaneBorder is the number of border columns a pane style adds beyond the
	// width it is requested with (lipgloss Width excludes the 1-column border
	// on each side).
	PaneBorder = 2
	// StatusBarHeight is the number of lines reserved for the full status bar.
	StatusBarHeight = 5
	// CompactStatusBarHeight is the status bar's height once it collapses to a
	// single line below StatusBarCompactBreakpoint terminal rows.
	CompactStatusBarHeight = 1
	// StatusBarCompactBreakpoint is the terminal height below which the status
	// bar switches to its single-line compact view.
	StatusBarCompactBreakpoint = 15
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
	sbHeight := StatusBarHeight
	if termHeight > 0 && termHeight < StatusBarCompactBreakpoint {
		sbHeight = CompactStatusBarHeight
	}
	h := termHeight - (sbHeight + HelpBarHeight + 3)
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
