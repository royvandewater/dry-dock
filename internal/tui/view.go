package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
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
)

func (m Model) render() string {
	if m.width == 0 || m.height == 0 {
		return "loading…"
	}
	if len(m.plugins) == 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			titleStyle.Render("dry-dock")+"\n\nAll plugins are up to date. 🚢")
	}

	header := m.renderHeader()
	footer := helpStyle.Render(m.helpText())

	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer)
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	changesWidth := m.width - pluginPaneWidth - versionPaneWidth - 6 // borders
	if changesWidth < 10 {
		changesWidth = 10
	}

	plugins := m.pane(m.renderPlugins(), pluginPaneWidth, bodyHeight, m.focus == focusPlugins)
	versions := m.pane(m.renderVersions(), versionPaneWidth, bodyHeight, m.focus == focusVersions)
	changes := m.pane(m.renderChanges(changesWidth), changesWidth, bodyHeight, false)

	body := lipgloss.JoinHorizontal(lipgloss.Top, plugins, versions, changes)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) pane(content string, width, height int, focused bool) string {
	style := blurredBorder
	if focused {
		style = focusedBorder
	}
	return style.Width(width).Height(height - 2).Render(content)
}

func (m Model) renderHeader() string {
	age := formatDuration(m.minAge)
	return titleStyle.Render("dry-dock") +
		dimStyle.Render(fmt.Sprintf("  ·  minimum release age: %s  ·  %d plugin(s) with updates", age, len(m.plugins)))
}

func (m Model) renderPlugins() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Plugins") + "\n")
	for i, p := range m.plugins {
		line := truncate(p.Name, pluginPaneWidth-2)
		if i == m.pluginIdx {
			line = selectedStyle.Render(padRight(line, pluginPaneWidth-2))
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m Model) renderVersions() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Versions") + "\n")

	visible := m.VisibleVersions()
	if len(visible) == 0 {
		b.WriteString(dimStyle.Render("(none old enough)"))
		return b.String()
	}

	for i, v := range visible {
		label := fmt.Sprintf("%s  %s", shortSHA(v.SHA), relativeDate(v.Date, m.now))
		subject := truncate(v.Subject, versionPaneWidth-2)
		if m.focus == focusVersions && i == m.versionIdx {
			label = selectedStyle.Render(padRight(label, versionPaneWidth-2))
			subject = selectedStyle.Render(padRight("  "+subject, versionPaneWidth-2))
		} else {
			label = shaStyle.Render(label)
			subject = "  " + dimStyle.Render(subject)
		}
		b.WriteString(label + "\n" + subject + "\n")
	}
	return b.String()
}

func (m Model) renderChanges(width int) string {
	var b strings.Builder
	p := m.SelectedPlugin()

	sel, ok := m.SelectedVersion()
	if !ok {
		b.WriteString(titleStyle.Render("Changes") + "\n\n")
		b.WriteString(dimStyle.Render("Press → to pick a version for " + p.Name))
		return b.String()
	}

	b.WriteString(titleStyle.Render(fmt.Sprintf("Changes: %s → %s", shortSHA(p.Current.SHA), shortSHA(sel.SHA))))
	b.WriteString(dimStyle.Render(fmt.Sprintf("  (%d commits)", len(m.SelectedChanges()))) + "\n\n")

	for _, c := range m.SelectedChanges() {
		meta := shaStyle.Render(fmt.Sprintf("%s  %s", shortSHA(c.SHA), c.Date.Format("2006-01-02")))
		b.WriteString(meta + "\n")
		for _, line := range wrap(c.Subject, width-2) {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) helpText() string {
	if m.focus == focusVersions {
		return "↑/↓ version  ·  ← plugins  ·  q quit"
	}
	return "↑/↓ plugin  ·  → versions  ·  q quit"
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
