package screens

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/divyo-argha/git-user/internal/config"
	"github.com/divyo-argha/git-user/internal/stats"
	"github.com/divyo-argha/git-user/internal/tui/theme"
)

func TestStatsScreenToggle(t *testing.T) {
	store := &config.Store{}
	th := theme.DefaultTheme()
	sc := NewStatsScreen(store, th)

	if sc.sortMode != stats.SortByCommits {
		t.Fatalf("expected initial sortMode to be SortByCommits, got %v", sc.sortMode)
	}

	// Press Right Arrow -> switch to SortByLines
	updated, _ := sc.Update(tea.KeyMsg{Type: tea.KeyRight})
	newSc := updated.(*StatsScreen)
	if newSc.sortMode != stats.SortByLines {
		t.Fatalf("expected sortMode after KeyRight to be SortByLines, got %v", newSc.sortMode)
	}

	// Press Left Arrow -> switch back to SortByCommits
	updated2, _ := newSc.Update(tea.KeyMsg{Type: tea.KeyLeft})
	newSc2 := updated2.(*StatsScreen)
	if newSc2.sortMode != stats.SortByCommits {
		t.Fatalf("expected sortMode after KeyLeft to be SortByCommits, got %v", newSc2.sortMode)
	}
}
