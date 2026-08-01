package components

import (
	"strings"
	"testing"

	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestToast(t *testing.T) {
	th := theme.DefaultTheme()
	toast := NewToast(th)

	// 1. Initial State
	if toast.IsVisible() {
		t.Error("Expected toast to be hidden initially")
	}
	if toast.View(80) != "" {
		t.Error("Expected empty view when hidden")
	}

	// 2. Show Success
	toast.Show("Success Message", theme.ToastStyleSuccess)
	if !toast.IsVisible() {
		t.Error("Expected toast to be visible")
	}
	viewSuccess := toast.View(80)
	if viewSuccess == "" {
		t.Error("Expected non-empty view when visible")
	}
	if !strings.Contains(viewSuccess, "Success Message") {
		t.Errorf("Expected view to contain message, got: %q", viewSuccess)
	}

	// 3. Show Error
	toast.Show("Error Message", theme.ToastStyleError)
	viewError := toast.View(80)
	if !strings.Contains(viewError, "Error Message") {
		t.Errorf("Expected view to contain error message, got: %q", viewError)
	}

	// 4. Show Info (default style)
	toast.Show("Info Message", theme.ToastStyleKind(999))
	viewInfo := toast.View(80)
	if !strings.Contains(viewInfo, "Info Message") {
		t.Errorf("Expected view to contain info message, got: %q", viewInfo)
	}

	// 5. Short width constraint
	toast.Show("Some Message", theme.ToastStyleSuccess)
	viewShort := toast.View(15)
	if viewShort == "" {
		t.Error("Expected non-empty view even with short width")
	}

	// 6. Hide
	toast.Hide()
	if toast.IsVisible() {
		t.Error("Expected toast to be hidden after Hide()")
	}
	if toast.View(80) != "" {
		t.Error("Expected empty view after Hide()")
	}
}
