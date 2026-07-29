package theme

import (
	"strings"
	"testing"
)

func TestDefaultTheme_NotEmpty(t *testing.T) {
	th := DefaultTheme()
	if th.Primary == "" {
		t.Error("Primary color should not be empty")
	}
	if th.Secondary == "" {
		t.Error("Secondary color should not be empty")
	}
	if th.Accent == "" {
		t.Error("Accent color should not be empty")
	}
	if th.Danger == "" {
		t.Error("Danger color should not be empty")
	}
	if th.Warning == "" {
		t.Error("Warning color should not be empty")
	}
	if th.Muted == "" {
		t.Error("Muted color should not be empty")
	}
	if th.Text == "" {
		t.Error("Text color should not be empty")
	}
	if th.TextDim == "" {
		t.Error("TextDim color should not be empty")
	}
	if th.Background == "" {
		t.Error("Background color should not be empty")
	}
}

func TestStyleAccessors(t *testing.T) {
	th := DefaultTheme()
	tests := []struct {
		name string
		got  string
	}{
		{"Bold", th.Bold().Render("test")},
		{"Dim", th.Dim().Render("test")},
		{"ItalicStyle", th.ItalicStyle().Render("test")},
		{"SuccessStyle", th.SuccessStyle().Render("test")},
		{"ErrorStyle", th.ErrorStyle().Render("test")},
		{"WarningStyle", th.WarningStyle().Render("test")},
		{"InfoStyle", th.InfoStyle().Render("test")},
		{"DangerText", th.DangerText().Render("test")},
		{"Selected", th.Selected().Render("test")},
		{"Active", th.Active().Render("test")},
		{"PaneTitle", th.PaneTitle().Render("test")},
		{"SectionHeader", th.SectionHeader().Render("test")},
		{"Separator", th.Separator().Render("test")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == "" {
				t.Errorf("%s rendered empty string", tt.name)
			}
		})
	}
}

func TestPaneStyles(t *testing.T) {
	th := DefaultTheme()
	tests := []struct {
		name string
		got  string
	}{
		{"ActivePane", th.ActivePane(40, 5).Render("test")},
		{"InactivePane", th.InactivePane(40, 5).Render("test")},
		{"PulsingActivePane", th.PulsingActivePane(40, 5, 0).Render("test")},
		{"DetailCardActive", th.DetailCardActive(40, 5).Render("test")},
		{"DetailCardInactive", th.DetailCardInactive(40, 5).Render("test")},
		{"ActionPane", th.ActionPane(40, 5).Render("test")},
		{"HUDBox", th.HUDBox(40).Render("test")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == "" {
				t.Errorf("%s rendered empty string", tt.name)
			}
		})
	}
}

func TestPillStyles(t *testing.T) {
	th := DefaultTheme()
	tests := []struct {
		name string
		got  string
	}{
		{"PillActive", th.PillActive().Render("test")},
		{"PillBadge", th.PillBadge().Render("test")},
		{"PillWarning", th.PillWarning().Render("test")},
		{"PillDanger", th.PillDanger().Render("test")},
		{"PillMuted", th.PillMuted().Render("test")},
		{"Keycap", th.Keycap().Render("test")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == "" {
				t.Errorf("%s rendered empty string", tt.name)
			}
		})
	}
}

func TestToastStyles(t *testing.T) {
	th := DefaultTheme()
	tests := []struct {
		name string
		got  string
	}{
		{"ToastSuccess", th.ToastSuccess(40).Render("test")},
		{"ToastError", th.ToastError(40).Render("test")},
		{"ToastInfo", th.ToastInfo(40).Render("test")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == "" {
				t.Errorf("%s rendered empty string", tt.name)
			}
		})
	}
}

func TestPaneWidth(t *testing.T) {
	tests := []struct {
		termWidth int
		expected  int
	}{
		{10, 20},
		{20, 20},
		{30, 13},
		{40, 18},
		{80, 38},
		{120, 58},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := PaneWidth(tt.termWidth)
			if got != tt.expected {
				t.Errorf("PaneWidth(%d) = %d, want %d", tt.termWidth, got, tt.expected)
			}
		})
	}
}

func TestContentHeight(t *testing.T) {
	tests := []struct {
		termHeight int
		expected   int
	}{
		{3, 5},
		{5, 5},
		{10, 5},
		{ChromeHeight + 5, 5},
		{ChromeHeight + 20, 20},
		{ChromeHeight + 50, 50},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := ContentHeight(tt.termHeight)
			if got != tt.expected {
				t.Errorf("ContentHeight(%d) = %d, want %d", tt.termHeight, got, tt.expected)
			}
		})
	}
}

func TestIsSingleColumn(t *testing.T) {
	tests := []struct {
		termWidth int
		expected  bool
	}{
		{30, true},
		{MinTermWidth - 1, true},
		{MinTermWidth, false},
		{MinTermWidth + 10, false},
		{200, false},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := IsSingleColumn(tt.termWidth)
			if got != tt.expected {
				t.Errorf("IsSingleColumn(%d) = %v, want %v", tt.termWidth, got, tt.expected)
			}
		})
	}
}

func TestSeparatorLine(t *testing.T) {
	th := DefaultTheme()
	tests := []struct {
		width int
	}{
		{0},
		{10},
		{40},
		{80},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := th.SeparatorLine(tt.width)
			if got == "" {
				t.Errorf("SeparatorLine(%d) should not be empty", tt.width)
			}
		})
	}
}

func TestSeparatorLine_DefaultsTo40(t *testing.T) {
	th := DefaultTheme()
	got := th.SeparatorLine(0)
	if len(got) == 0 {
		t.Error("SeparatorLine(0) should use default width of 40")
	}
}

func TestConstants(t *testing.T) {
	if MinTermWidth != 60 {
		t.Errorf("MinTermWidth = %d, want 60", MinTermWidth)
	}
	if PaneGap != 3 {
		t.Errorf("PaneGap = %d, want 3", PaneGap)
	}
	if StatusBarHeight != 5 {
		t.Errorf("StatusBarHeight = %d, want 5", StatusBarHeight)
	}
	if HelpBarHeight != 2 {
		t.Errorf("HelpBarHeight = %d, want 2", HelpBarHeight)
	}
	if ChromeHeight != StatusBarHeight+HelpBarHeight+3 {
		t.Errorf("ChromeHeight = %d, want %d", ChromeHeight, StatusBarHeight+HelpBarHeight+3)
	}
}

func TestToastStyleKind_Values(t *testing.T) {
	if ToastStyleSuccess != 0 {
		t.Errorf("ToastStyleSuccess = %d, want 0", ToastStyleSuccess)
	}
	if ToastStyleError != 1 {
		t.Errorf("ToastStyleError = %d, want 1", ToastStyleError)
	}
	if ToastStyleInfo != 2 {
		t.Errorf("ToastStyleInfo = %d, want 2", ToastStyleInfo)
	}
}

func TestDefaultTheme_HasSeparator(t *testing.T) {
	th := DefaultTheme()
	sep := th.SeparatorLine(20)
	if !strings.Contains(sep, "─") {
		t.Error("SeparatorLine should contain horizontal line characters")
	}
}
