package components

import (
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

// Toast renders a transient notification overlay.
type Toast struct {
	text    string
	style   theme.ToastStyleKind
	visible bool
	theme   theme.Theme
}

// NewToast creates a new toast component.
func NewToast(th theme.Theme) Toast {
	return Toast{theme: th}
}

// Show displays a toast message.
func (t *Toast) Show(text string, style theme.ToastStyleKind) {
	t.text = text
	t.style = style
	t.visible = true
}

// Hide dismisses the toast.
func (t *Toast) Hide() {
	t.visible = false
}

// IsVisible returns whether the toast is shown.
func (t Toast) IsVisible() bool { return t.visible }

// View renders the toast notification.
func (t Toast) View(width int) string {
	if !t.visible || t.text == "" {
		return ""
	}

	toastWidth := 40
	if width < 50 {
		toastWidth = width - 10
	}
	if toastWidth < 20 {
		toastWidth = 20
	}

	var icon string
	var s string

	switch t.style {
	case theme.ToastStyleSuccess:
		icon = "✔ "
		s = t.theme.ToastSuccess(toastWidth).Render(icon + t.text)
	case theme.ToastStyleError:
		icon = "✖ "
		s = t.theme.ToastError(toastWidth).Render(icon + t.text)
	default:
		icon = "ℹ "
		s = t.theme.ToastInfo(toastWidth).Render(icon + t.text)
	}

	return s
}
