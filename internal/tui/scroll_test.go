package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

func size(w, h int) tea.WindowSizeMsg { return tea.WindowSizeMsg{Width: w, Height: h} }

// longChangelog builds a model whose selected plugin has many versions, so its
// changelog is far taller than any reasonable viewport.
func longChangelog() Model {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	var candidates []plugin.Version
	for i := range 40 {
		candidates = append(candidates, plugin.Version{
			SHA:     "sha" + string(rune('a'+i%26)) + string(rune('0'+i/26)),
			Subject: "change number " + string(rune('0'+i%10)),
			Date:    now.Add(-time.Duration(30+i) * day),
		})
	}
	p := plugin.Plugin{Name: "big.nvim", Current: plugin.Version{SHA: "cur"}, Candidates: candidates}
	return New([]plugin.Plugin{p}, now, 14*day)
}

func drive(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func TestRightArrowFromVersionsFocusesChanges(t *testing.T) {
	m := drive(sample(), size(120, 30), key(tea.KeyRight), key(tea.KeyRight))
	if m.focus != focusChanges {
		t.Fatalf("expected focusChanges after two right arrows, got %v", m.focus)
	}
}

func TestDownInChangesFocusScrollsChangelog(t *testing.T) {
	m := drive(longChangelog(), size(120, 30), key(tea.KeyRight), key(tea.KeyRight), key(tea.KeyDown))
	if m.changesScroll != 1 {
		t.Fatalf("expected changesScroll 1 after one down, got %d", m.changesScroll)
	}
}

func TestChangesScrollClampsToBottom(t *testing.T) {
	m := drive(longChangelog(), size(120, 30), key(tea.KeyRight), key(tea.KeyRight))
	max := m.maxChangesScroll()
	if max <= 0 {
		t.Fatalf("expected a positive max scroll for a long changelog, got %d", max)
	}
	for range 1000 {
		m = drive(m, key(tea.KeyDown))
	}
	if m.changesScroll != max {
		t.Fatalf("expected changesScroll clamped to %d, got %d", max, m.changesScroll)
	}
}

func TestShortChangelogDoesNotScroll(t *testing.T) {
	m := drive(sample(), size(120, 30), key(tea.KeyRight), key(tea.KeyRight), key(tea.KeyDown))
	if m.changesScroll != 0 {
		t.Fatalf("expected no scroll for short changelog, got %d", m.changesScroll)
	}
}

func TestChangingVersionResetsChangesScroll(t *testing.T) {
	m := drive(longChangelog(), size(120, 30), key(tea.KeyRight), key(tea.KeyRight), key(tea.KeyDown), key(tea.KeyDown))
	if m.changesScroll == 0 {
		t.Fatal("precondition: expected a scrolled changelog")
	}
	// Back to versions, pick a different version — the changelog resets to top.
	m = drive(m, key(tea.KeyLeft), key(tea.KeyDown))
	if m.changesScroll != 0 {
		t.Fatalf("expected changesScroll reset to 0 on version change, got %d", m.changesScroll)
	}
}

func TestLeftArrowStepsBackThroughPanes(t *testing.T) {
	m := drive(sample(), size(120, 30), key(tea.KeyRight), key(tea.KeyRight))
	m = drive(m, key(tea.KeyLeft))
	if m.focus != focusVersions {
		t.Fatalf("expected focusVersions after left from changes, got %v", m.focus)
	}
	m = drive(m, key(tea.KeyLeft))
	if m.focus != focusPlugins {
		t.Fatalf("expected focusPlugins after left from versions, got %v", m.focus)
	}
}
