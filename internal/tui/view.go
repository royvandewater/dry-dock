package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

const (
	pluginPaneWidth  = 26
	versionPaneWidth = 34
)

var (
	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212"))
	blurredBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("212"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	shaStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	warningStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))
)

// layout holds the derived pane geometry for the current window size.
type layout struct {
	bodyHeight      int // height of the pane row, including borders
	innerH          int // content height inside a pane's border
	listRows        int // scrollable rows in the plugin/version panes (below the title)
	changesViewport int // scrollable rows in the changes pane (below its header)
	changesWidth    int
}

func (m Model) layout() layout {
	bodyHeight := m.height - 2 // header + footer
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	innerH := bodyHeight - 2 // pane top+bottom border

	changesWidth := m.width - pluginPaneWidth - versionPaneWidth - 6 // three borders
	if changesWidth < 10 {
		changesWidth = 10
	}

	return layout{
		bodyHeight:      bodyHeight,
		innerH:          innerH,
		listRows:        max(1, innerH-1), // one line for the title
		changesViewport: max(1, innerH-2), // header line + blank
		changesWidth:    changesWidth,
	}
}

func (m Model) render() string {
	if m.width == 0 || m.height == 0 {
		return "loading…"
	}
	if len(m.plugins) == 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			titleStyle.Render("dry-dock")+"\n\nAll plugins are up to date. 🚢")
	}

	l := m.layout()
	header := m.renderHeader()
	footer := m.renderFooter()

	plugins := m.pane(m.pluginContent(l), pluginPaneWidth, l, m.focus == focusPlugins)
	versions := m.pane(m.versionContent(l), versionPaneWidth, l, m.focus == focusVersions)
	changes := m.pane(m.changesContent(l), l.changesWidth, l, m.focus == focusChanges)

	body := lipgloss.JoinHorizontal(lipgloss.Top, plugins, versions, changes)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) pane(content string, width int, l layout, focused bool) string {
	style := blurredBorder
	if focused {
		style = focusedBorder
	}
	return style.Width(width).Height(l.innerH).Render(content)
}

// pluginContent renders the plugin pane: a sticky title above a windowed list
// that follows the selection.
func (m Model) pluginContent(l layout) string {
	lines := m.pluginBodyLines()
	offset := follow(m.pluginIdx, len(lines), l.listRows)
	body := window(lines, offset, l.listRows)
	return titleStyle.Render(scrollTitle("Plugins", offset, len(lines), l.listRows)) + "\n" +
		strings.Join(body, "\n")
}

func (m Model) versionContent(l layout) string {
	p := m.SelectedPlugin()

	// Show which version the plugin is currently pinned to. The installable
	// list only holds versions newer than this, so without the marker there's
	// nothing anchoring the changes against what's already installed. The line
	// costs one row of the list window.
	installedID := shortSHA(p.Current.SHA)
	if p.Current.Tag != "" {
		installedID = p.Current.Tag
	}
	installed := dimStyle.Render("installed " + installedID)

	rows := max(1, l.listRows-1) // installed line
	lines, starts := m.versionBodyLines()
	if len(lines) == 0 {
		return titleStyle.Render("Versions") + "\n" + installed + "\n" + dimStyle.Render("(up to date)")
	}

	// Keep the selected entry's first line in view; a header occupies one line,
	// a version two, so the offset comes from the entry→line map.
	selStart := 0
	if m.versionIdx >= 0 && m.versionIdx < len(starts) {
		selStart = starts[m.versionIdx]
	}
	offset := follow(selStart, len(lines), rows)
	body := window(lines, offset, rows)
	return titleStyle.Render(scrollTitle("Versions", offset, len(lines), rows)) + "\n" + installed + "\n" +
		strings.Join(body, "\n")
}

func (m Model) changesContent(l layout) string {
	p := m.SelectedPlugin()
	sel, ok := m.SelectedVersion()
	if !ok {
		return titleStyle.Render("Changes") + "\n\n" +
			dimStyle.Render("Press → to pick a version for "+p.Name)
	}

	header := titleStyle.Render(fmt.Sprintf("Changes: %s → %s", shortSHA(p.Current.SHA), shortSHA(sel.SHA))) +
		dimStyle.Render(fmt.Sprintf("  (%d commits)", len(m.SelectedChanges())))

	lines := m.changesBodyLines(l.changesWidth)
	offset := clamp(m.changesScroll, 0, m.maxChangesScroll())
	body := window(lines, offset, l.changesViewport)
	return header + "\n\n" + strings.Join(body, "\n")
}

// --- body line builders (title-free, one slice element per rendered row) ---

func (m Model) pluginBodyLines() []string {
	lines := make([]string, len(m.plugins))
	for i, p := range m.plugins {
		line := truncate(p.Name, pluginPaneWidth-2)
		switch {
		case i == m.pluginIdx:
			line = selectedStyle.Render(padRight(line, pluginPaneWidth-2))
		case m.upToDate(i):
			line = dimStyle.Render(line)
		}
		lines[i] = line
	}
	return lines
}

// versionBodyLines renders the versions pane's entries into display lines,
// returning the lines and a map from entry index to its first line so the pane
// can scroll the selection into view. Headers render as one line, versions as
// two (id/date, then subject).
func (m Model) versionBodyLines() ([]string, []int) {
	p := m.SelectedPlugin()
	entries := m.versionEntries()
	var lines []string
	starts := make([]int, len(entries))
	for i, e := range entries {
		starts[i] = len(lines)
		selected := m.focus == focusVersions && i == m.versionIdx
		switch e.kind {
		case entryOutOfScopeHeader:
			text := fmt.Sprintf("%d newer %s outside %s", e.count, plural(e.count, "release", "releases"), p.Constraint)
			lines = append(lines, headerLine(text, m.expandOutOfScope, selected, warningStyle))
		case entryTooNewHeader:
			text := fmt.Sprintf("%d %s too new to install", e.count, plural(e.count, "release", "releases"))
			lines = append(lines, headerLine(text, m.expandTooNew, selected, dimStyle))
		default:
			label, subject := m.versionRow(p, e.version, selected, e.kind != entryInstallable)
			lines = append(lines, label, subject)
		}
	}
	return lines, starts
}

// headerLine renders a collapsible warning header: a ▶/▼ marker plus its text,
// highlighted when selected and drawn in style otherwise.
func headerLine(text string, expanded, selected bool, style lipgloss.Style) string {
	marker := "▶"
	if expanded {
		marker = "▼"
	}
	full := truncate(marker+" "+text, versionPaneWidth-2)
	if selected {
		return selectedStyle.Render(padRight(full, versionPaneWidth-2))
	}
	return style.Render(full)
}

// versionRow renders a version's two lines (id/date, then subject). Revealed
// releases (out-of-scope, too-new) are indented and muted to sit under their
// header; the selected row is highlighted.
func (m Model) versionRow(p plugin.Plugin, v plugin.Version, selected, revealed bool) (string, string) {
	id := shortSHA(v.SHA)
	if v.Tag != "" {
		id = v.Tag
	}
	prefix := ""
	if revealed {
		prefix = "  "
	}
	text := fmt.Sprintf("%s%s  %s", prefix, id, relativeDate(v.Date, m.now))
	breaking := p.IncludesBreaking(v.SHA)
	subject := truncate(v.Subject, versionPaneWidth-2)
	if selected {
		if breaking {
			text += "  ⚠"
		}
		label := selectedStyle.Render(padRight(truncate(text, versionPaneWidth-2), versionPaneWidth-2))
		return label, selectedStyle.Render(padRight("  "+subject, versionPaneWidth-2))
	}
	style := shaStyle
	if revealed {
		style = dimStyle
	}
	label := style.Render(truncate(text, versionPaneWidth-2))
	if breaking {
		label += "  " + warningStyle.Render("⚠")
	}
	return label, "  " + dimStyle.Render(subject)
}

// plural picks the singular or plural noun for n.
func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func (m Model) changesBodyLines(width int) []string {
	var lines []string
	for _, c := range m.SelectedChanges() {
		head := shaStyle.Render(fmt.Sprintf("%s  %s", shortSHA(c.SHA), c.Date.Format("2006-01-02")))
		if c.Breaking() {
			head += "  " + warningStyle.Render("⚠ BREAKING")
		}
		lines = append(lines, head)
		for _, line := range wrap(c.Subject, width-2) {
			lines = append(lines, "  "+line)
		}
		lines = append(lines, "")
	}
	return lines
}

func (m Model) renderHeader() string {
	age := formatDuration(m.minAge)
	updatable := 0
	for i := range m.plugins {
		if !m.upToDate(i) {
			updatable++
		}
	}
	return titleStyle.Render("dry-dock") +
		dimStyle.Render(fmt.Sprintf("  ·  minimum release age: %s  ·  %d plugin(s) with updates", age, updatable))
}

// renderFooter shows the last update status when there is one, otherwise the
// context-sensitive key hints. The status is truncated to the window width so a
// long (or multi-line) error can never overrun the single-line footer.
func (m Model) renderFooter() string {
	if m.status != "" {
		style := titleStyle
		if m.statusErr {
			style = warningStyle
		}
		hint := "  ·  esc dismiss"
		text := truncate(m.status, m.width-lipgloss.Width(hint))
		return style.Render(text) + helpStyle.Render(hint)
	}
	return helpStyle.Render(m.helpText())
}

func (m Model) helpText() string {
	switch m.focus {
	case focusChanges:
		return "↑/↓ scroll changes  ·  enter update  ·  ← versions  ·  q quit"
	case focusVersions:
		action := "enter update"
		if e, ok := m.selectedEntry(); ok {
			switch {
			case e.isHeader():
				action = "enter expand"
			case e.kind != entryInstallable:
				action = "" // view-only release; nothing to apply
			}
		}
		hint := "↑/↓ version"
		if action != "" {
			hint += "  ·  " + action
		}
		return hint + "  ·  → changes  ·  ← plugins  ·  q quit"
	default:
		return "↑/↓ plugin  ·  → versions  ·  q quit"
	}
}

// --- windowing helpers ---

// window returns exactly height lines starting at offset, padding with blanks
// when the source runs short so panes stay a fixed height.
func window(lines []string, offset, height int) []string {
	out := make([]string, height)
	for i := range height {
		src := offset + i
		if src >= 0 && src < len(lines) {
			out[i] = lines[src]
		}
	}
	return out
}

// follow returns a scroll offset that keeps sel visible within a height-line
// window, biased to center it, clamped to the content bounds.
func follow(sel, total, height int) int {
	if total <= height {
		return 0
	}
	return clamp(sel-height/2, 0, total-height)
}

// scrollTitle appends a position hint to a pane title when its list overflows.
func scrollTitle(name string, offset, total, height int) string {
	if total <= height {
		return name
	}
	return fmt.Sprintf("%s  ↕ %d–%d/%d", name, offset+1, min(offset+height, total), total)
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return string([]rune(s)[:max-1]) + "…"
}

func padRight(s string, width int) string {
	gap := width - lipgloss.Width(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

func wrap(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	line := words[0]
	for _, w := range words[1:] {
		if lipgloss.Width(line)+1+lipgloss.Width(w) > width {
			lines = append(lines, line)
			line = w
		} else {
			line += " " + w
		}
	}
	return append(lines, line)
}

func relativeDate(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < 24*time.Hour:
		return "today"
	case d < 48*time.Hour:
		return "yesterday"
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/24/365))
	}
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days >= 1 {
		return fmt.Sprintf("%d days", days)
	}
	return d.String()
}
