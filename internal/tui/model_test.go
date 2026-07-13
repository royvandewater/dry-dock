package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

func key(k tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: k} }

// sample builds a model with two plugins. telescope has three candidates, the
// newest of which is too young to install under a 14-day minimum age.
func sample() Model {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	tel := plugin.Plugin{
		Name:    "telescope.nvim",
		Current: plugin.Version{SHA: "telCur", Subject: "current"},
		Candidates: []plugin.Version{
			{SHA: "telC", Subject: "youngest", Date: now.Add(-2 * day)},  // too young
			{SHA: "telB", Subject: "middle", Date: now.Add(-30 * day)},   // ok
			{SHA: "telA", Subject: "oldest new", Date: now.Add(-60 * day)}, // ok
		},
	}
	cmp := plugin.Plugin{
		Name:    "blink.cmp",
		Current: plugin.Version{SHA: "cmpCur"},
		Candidates: []plugin.Version{
			{SHA: "cmpA", Subject: "one", Date: now.Add(-40 * day)},
		},
	}

	return New([]plugin.Plugin{tel, cmp}, now, 14*day)
}

func TestDownArrowMovesPluginSelection(t *testing.T) {
	m := sample()
	if m.SelectedPlugin().Name != "telescope.nvim" {
		t.Fatalf("expected telescope selected first, got %q", m.SelectedPlugin().Name)
	}

	next, _ := m.Update(key(tea.KeyDown))
	m = next.(Model)

	if m.SelectedPlugin().Name != "blink.cmp" {
		t.Fatalf("expected blink.cmp after down, got %q", m.SelectedPlugin().Name)
	}
}

func TestPluginSelectionClampsAtEnds(t *testing.T) {
	m := sample()

	next, _ := m.Update(key(tea.KeyUp)) // already at top
	m = next.(Model)
	if m.SelectedPlugin().Name != "telescope.nvim" {
		t.Fatalf("up at top should stay on telescope, got %q", m.SelectedPlugin().Name)
	}

	for range 5 { // far past the end
		next, _ = m.Update(key(tea.KeyDown))
		m = next.(Model)
	}
	if m.SelectedPlugin().Name != "blink.cmp" {
		t.Fatalf("down past end should stay on blink.cmp, got %q", m.SelectedPlugin().Name)
	}
}

func TestVersionListHidesVersionsYoungerThanMinAge(t *testing.T) {
	m := sample()

	visible := m.VisibleVersions()
	if len(visible) != 2 {
		t.Fatalf("expected 2 installable versions, got %d", len(visible))
	}
	if visible[0].SHA != "telB" || visible[1].SHA != "telA" {
		t.Fatalf("expected [telB, telA] newest-first, got [%s, %s]", visible[0].SHA, visible[1].SHA)
	}
}

func TestRightArrowFocusesVersionsAndSelectsTop(t *testing.T) {
	m := sample()

	next, _ := m.Update(key(tea.KeyRight))
	m = next.(Model)

	sel, ok := m.SelectedVersion()
	if !ok {
		t.Fatal("expected a selected version after focusing versions")
	}
	if sel.SHA != "telB" {
		t.Fatalf("expected top installable version telB, got %q", sel.SHA)
	}
}

func TestDownArrowInVersionFocusMovesVersionSelection(t *testing.T) {
	m := sample()
	next, _ := m.Update(key(tea.KeyRight))
	m = next.(Model)

	next, _ = m.Update(key(tea.KeyDown))
	m = next.(Model)

	sel, _ := m.SelectedVersion()
	if sel.SHA != "telA" {
		t.Fatalf("expected telA after down in version focus, got %q", sel.SHA)
	}
}

func TestLeftArrowReturnsFocusToPlugins(t *testing.T) {
	m := sample()
	next, _ := m.Update(key(tea.KeyRight))
	m = next.(Model)
	next, _ = m.Update(key(tea.KeyLeft))
	m = next.(Model)

	// With focus back on plugins, down should move plugins, not versions.
	next, _ = m.Update(key(tea.KeyDown))
	m = next.(Model)
	if m.SelectedPlugin().Name != "blink.cmp" {
		t.Fatalf("expected plugin focus after left arrow, got plugin %q", m.SelectedPlugin().Name)
	}
}

func TestSelectedChangesAreCumulativeFromCurrentThroughSelected(t *testing.T) {
	m := sample()
	next, _ := m.Update(key(tea.KeyRight)) // focus versions, top = telB
	m = next.(Model)

	// telB is 30 days old; the only version between current and telB is telA.
	// Updating to telB therefore pulls in telB and telA, newest first.
	changes := m.SelectedChanges()
	if len(changes) != 2 {
		t.Fatalf("expected 2 cumulative changes for telB, got %d", len(changes))
	}
	if changes[0].SHA != "telB" || changes[1].SHA != "telA" {
		t.Fatalf("expected [telB, telA], got [%s, %s]", changes[0].SHA, changes[1].SHA)
	}

	// Moving down to telA (immediately after current) pulls in only telA.
	next, _ = m.Update(key(tea.KeyDown))
	m = next.(Model)
	changes = m.SelectedChanges()
	if len(changes) != 1 || changes[0].SHA != "telA" {
		t.Fatalf("expected only [telA] for oldest new version, got %+v", changes)
	}
}

func TestChangingPluginResetsVersionSelection(t *testing.T) {
	m := sample()
	next, _ := m.Update(key(tea.KeyRight)) // focus versions
	m = next.(Model)
	next, _ = m.Update(key(tea.KeyDown)) // move to telA
	m = next.(Model)
	next, _ = m.Update(key(tea.KeyLeft)) // back to plugins
	m = next.(Model)
	next, _ = m.Update(key(tea.KeyDown)) // to blink.cmp
	m = next.(Model)
	next, _ = m.Update(key(tea.KeyUp)) // back to telescope
	m = next.(Model)
	next, _ = m.Update(key(tea.KeyRight)) // focus versions again
	m = next.(Model)

	sel, _ := m.SelectedVersion()
	if sel.SHA != "telB" {
		t.Fatalf("expected version selection reset to top telB, got %q", sel.SHA)
	}
}
