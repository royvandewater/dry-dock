// Package tui implements the dry-dock terminal UI: a plugin list, the versions
// each plugin can update to, and the cumulative changelog for a chosen version.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

type focus int

const (
	focusPlugins focus = iota
	focusVersions
)

// Model holds all TUI state. Navigation state (which plugin, which version,
// which pane has focus) lives here; the panes are derived at render time.
type Model struct {
	plugins []plugin.Plugin
	now     time.Time
	minAge  time.Duration

	focus      focus
	pluginIdx  int
	versionIdx int

	width, height int
}

// New builds a Model over the given updatable plugins. now and minAge drive the
// minimum-release-age filter applied to each plugin's versions.
func New(plugins []plugin.Plugin, now time.Time, minAge time.Duration) Model {
	return Model{plugins: plugins, now: now, minAge: minAge}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) View() string { return m.render() }

// SelectedPlugin returns the currently highlighted plugin.
func (m Model) SelectedPlugin() plugin.Plugin {
	return m.plugins[m.pluginIdx]
}

// VisibleVersions is the selected plugin's installable versions (old enough to
// satisfy the minimum release age), newest first.
func (m Model) VisibleVersions() []plugin.Version {
	return m.SelectedPlugin().Installable(m.now, m.minAge)
}

// SelectedVersion returns the highlighted version, or ok=false when the version
// pane has nothing selected.
func (m Model) SelectedVersion() (plugin.Version, bool) {
	visible := m.VisibleVersions()
	if m.versionIdx < 0 || m.versionIdx >= len(visible) {
		return plugin.Version{}, false
	}
	return visible[m.versionIdx], true
}

// SelectedChanges returns every change pulled in by updating to the highlighted
// version: from the current version through the selected one, newest first.
// This spans versions filtered out of the list for being too young, since
// moving the plugin's ref forward necessarily includes them.
func (m Model) SelectedChanges() []plugin.Version {
	sel, ok := m.SelectedVersion()
	if !ok {
		return nil
	}
	p := m.SelectedPlugin()
	for i, c := range p.Candidates {
		if c.SHA == sel.SHA {
			return p.ChangesUpTo(i)
		}
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyRight:
		if m.focus == focusPlugins && len(m.VisibleVersions()) > 0 {
			m.focus = focusVersions
			m.versionIdx = 0
		}
	case tea.KeyLeft:
		m.focus = focusPlugins
	case tea.KeyUp:
		m.moveSelection(-1)
	case tea.KeyDown:
		m.moveSelection(1)
	default:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

// moveSelection advances the active pane's highlight by delta, clamped to its
// bounds. Moving between plugins resets the version highlight to the top.
func (m *Model) moveSelection(delta int) {
	switch m.focus {
	case focusPlugins:
		m.pluginIdx = clamp(m.pluginIdx+delta, 0, len(m.plugins)-1)
		m.versionIdx = 0
	case focusVersions:
		m.versionIdx = clamp(m.versionIdx+delta, 0, len(m.VisibleVersions())-1)
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
