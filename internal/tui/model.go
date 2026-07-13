// Package tui implements the dry-dock terminal UI: a plugin list, the versions
// each plugin can update to, and the cumulative changelog for a chosen version.
package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

// Applier performs an update: it moves the named plugin to the given commit
// SHA, both on disk and in lazy.vim's lock file.
type Applier interface {
	Apply(pluginName, sha string) error
}

// applyResultMsg reports the outcome of an Applier.Apply call back into the
// update loop.
type applyResultMsg struct {
	pluginName string
	sha        string
	err        error
}

type focus int

const (
	focusPlugins focus = iota
	focusVersions
	focusChanges
)

// Model holds all TUI state. Navigation state (which plugin, which version,
// which pane has focus) lives here; the panes are derived at render time.
type Model struct {
	plugins []plugin.Plugin
	now     time.Time
	minAge  time.Duration
	applier Applier

	// status is a one-line message describing the last update attempt;
	// statusErr marks it as a failure so the footer can colour it.
	status    string
	statusErr bool

	focus      focus
	pluginIdx  int
	versionIdx int

	// scroll offsets, in lines, for panes taller than their viewport.
	pluginScroll  int
	versionScroll int
	changesScroll int

	width, height int
}

// New builds a Model over the given plugins. now and minAge drive the
// minimum-release-age filter applied to each plugin's versions, and plugins
// with no installable versions are sorted to the bottom of the list.
func New(plugins []plugin.Plugin, now time.Time, minAge time.Duration) Model {
	sortByUpdatable(plugins, now, minAge)
	return Model{plugins: plugins, now: now, minAge: minAge}
}

// sortByUpdatable stably orders plugins so those with installable versions come
// first and the up-to-date (muted) ones sink to the bottom, each group keeping
// its original order.
func sortByUpdatable(plugins []plugin.Plugin, now time.Time, minAge time.Duration) {
	sort.SliceStable(plugins, func(i, j int) bool {
		return len(plugins[i].Installable(now, minAge)) > 0 &&
			len(plugins[j].Installable(now, minAge)) == 0
	})
}

// upToDate reports whether the plugin at index i has no versions old enough to
// install — the muted plugins pinned to the bottom of the list.
func (m Model) upToDate(i int) bool {
	return len(m.plugins[i].Installable(m.now, m.minAge)) == 0
}

// WithApplier returns a copy of the model that performs updates through a.
func (m Model) WithApplier(a Applier) Model {
	m.applier = a
	return m
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
	case applyResultMsg:
		if msg.err != nil {
			// The Applier's error leads with a one-line summary (e.g. "…broke
			// nvim, rolled back to abc1234"); keep just that so the footer
			// stays a single line instead of dumping a stack trace fullscreen.
			m.status = firstLine(msg.err.Error())
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf("updated %s → %s", msg.pluginName, shortSHA(msg.sha))
		m.statusErr = false
		m.integrateUpdate(msg.sha)
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
		return m, tea.Quit
	}
	if msg.Type == tea.KeyEsc {
		// Esc dismisses the status message (restoring the key hints); it does
		// not quit — that's q.
		m.status = ""
		m.statusErr = false
		return m, nil
	}
	if len(m.plugins) == 0 {
		return m, nil
	}

	switch msg.Type {
	case tea.KeyRight:
		m.focusNext()
	case tea.KeyLeft:
		m.focusPrev()
	case tea.KeyUp:
		m.moveSelection(-1)
	case tea.KeyDown:
		m.moveSelection(1)
	case tea.KeyEnter:
		return m, m.applySelected()
	}
	return m, nil
}

// integrateUpdate refreshes the plugin list after a successful update to sha.
// Because updating pulls the plugin's ref forward, the versions newer than sha
// remain installable while sha and everything older become history. When no
// newer versions remain, the plugin is now up to date; it stays in the list
// (muted) but re-sorts to the bottom rather than dropping off.
func (m *Model) integrateUpdate(sha string) {
	idx := m.pluginIdx
	p := m.plugins[idx]

	ci := -1
	for i, c := range p.Candidates {
		if c.SHA == sha {
			ci = i
			break
		}
	}
	if ci < 0 {
		return
	}

	p.Current = p.Candidates[ci]
	p.Candidates = p.Candidates[:ci]
	m.plugins[idx] = p

	// Re-sort so a now-up-to-date plugin sinks to the bottom, then keep the
	// selection on the plugin we just updated wherever it landed.
	sortByUpdatable(m.plugins, m.now, m.minAge)
	for i := range m.plugins {
		if m.plugins[i].Name == p.Name {
			m.pluginIdx = i
			break
		}
	}

	m.versionIdx = 0
	m.versionScroll = 0
	m.pluginScroll = 0
	m.changesScroll = 0

	// A refreshed plugin may have no versions old enough to install; fall back
	// to the plugin list so focus never lands on an empty pane.
	if len(m.VisibleVersions()) == 0 {
		m.focus = focusPlugins
	}
}

// applySelected returns a command that applies the highlighted version to the
// selected plugin. It's a no-op when no version is selected or no applier is
// wired in.
func (m *Model) applySelected() tea.Cmd {
	if m.applier == nil {
		return nil
	}
	sel, ok := m.SelectedVersion()
	if !ok {
		return nil
	}
	name := m.SelectedPlugin().Name
	applier := m.applier
	m.status = fmt.Sprintf("updating %s → %s…", name, shortSHA(sel.SHA))
	return func() tea.Msg {
		err := applier.Apply(name, sel.SHA)
		return applyResultMsg{pluginName: name, sha: sel.SHA, err: err}
	}
}

// focusNext moves focus rightward (plugins → versions → changes), only
// advancing into a pane that has something to show.
func (m *Model) focusNext() {
	switch m.focus {
	case focusPlugins:
		if len(m.VisibleVersions()) > 0 {
			m.focus = focusVersions
			m.versionIdx = 0
			m.changesScroll = 0
		}
	case focusVersions:
		if len(m.SelectedChanges()) > 0 {
			m.focus = focusChanges
		}
	}
}

// focusPrev moves focus leftward (changes → versions → plugins).
func (m *Model) focusPrev() {
	switch m.focus {
	case focusChanges:
		m.focus = focusVersions
	case focusVersions:
		m.focus = focusPlugins
	}
}

// moveSelection advances the active pane's highlight (or scroll) by delta,
// clamped to its bounds. Changing plugin or version resets downstream state so
// the panes to the right always start from the top.
func (m *Model) moveSelection(delta int) {
	switch m.focus {
	case focusPlugins:
		m.pluginIdx = clamp(m.pluginIdx+delta, 0, len(m.plugins)-1)
		m.versionIdx = 0
		m.versionScroll = 0
		m.changesScroll = 0
	case focusVersions:
		m.versionIdx = clamp(m.versionIdx+delta, 0, len(m.VisibleVersions())-1)
		m.changesScroll = 0
	case focusChanges:
		m.changesScroll = clamp(m.changesScroll+delta, 0, m.maxChangesScroll())
	}
}

// maxChangesScroll is the furthest the changelog can scroll: total rendered
// lines minus the visible region, floored at zero.
func (m Model) maxChangesScroll() int {
	l := m.layout()
	over := len(m.changesBodyLines(l.changesWidth)) - l.changesViewport
	if over < 0 {
		return 0
	}
	return over
}

// firstLine returns s up to its first newline — the human-readable summary of a
// multi-line error.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
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
