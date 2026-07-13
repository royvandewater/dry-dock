package tui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cucumber/godog"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

// sample builds a model with two plugins. telescope has three candidates, the
// newest of which is too young to install under a 14-day minimum age.
func sample() Model {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	tel := plugin.Plugin{
		Name:    "telescope.nvim",
		Current: plugin.Version{SHA: "telCur", Subject: "current"},
		Candidates: []plugin.Version{
			{SHA: "telC", Subject: "youngest", Date: now.Add(-2 * day)},    // too young
			{SHA: "telB", Subject: "middle", Date: now.Add(-30 * day)},     // ok
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

var keys = map[string]tea.KeyType{
	"up":    tea.KeyUp,
	"down":  tea.KeyDown,
	"left":  tea.KeyLeft,
	"right": tea.KeyRight,
}

var focusNames = map[string]focus{
	"plugins":  focusPlugins,
	"versions": focusVersions,
	"changes":  focusChanges,
}

type tuiWorld struct {
	model Model
}

func (w *tuiWorld) send(msg tea.Msg) {
	next, _ := w.model.Update(msg)
	w.model = next.(Model)
}

func (w *tuiWorld) theSampleModel() error {
	w.model = sample()
	return nil
}

func (w *tuiWorld) theLongChangelogModel() error {
	w.model = longChangelog()
	return nil
}

func (w *tuiWorld) aWindowSizeOf(width, height int) error {
	w.send(tea.WindowSizeMsg{Width: width, Height: height})
	return nil
}

func (w *tuiWorld) iPress(name string) error {
	return w.iPressTimes(name, 1)
}

func (w *tuiWorld) iPressTimes(name string, n int) error {
	kt, ok := keys[name]
	if !ok {
		return fmt.Errorf("unknown key %q", name)
	}
	for range n {
		w.send(tea.KeyMsg{Type: kt})
	}
	return nil
}

func (w *tuiWorld) theSelectedPluginIs(name string) error {
	if got := w.model.SelectedPlugin().Name; got != name {
		return fmt.Errorf("expected selected plugin %q, got %q", name, got)
	}
	return nil
}

func (w *tuiWorld) thereAreVisibleVersions(n int) error {
	if got := len(w.model.VisibleVersions()); got != n {
		return fmt.Errorf("expected %d visible versions, got %d", n, got)
	}
	return nil
}

func shaList(versions []plugin.Version) string {
	var out []string
	for _, v := range versions {
		out = append(out, v.SHA)
	}
	return strings.Join(out, ", ")
}

func (w *tuiWorld) theVisibleVersionShasAre(list string) error {
	if got := shaList(w.model.VisibleVersions()); got != list {
		return fmt.Errorf("expected visible version shas %q, got %q", list, got)
	}
	return nil
}

func (w *tuiWorld) aVersionIsSelected() error {
	if _, ok := w.model.SelectedVersion(); !ok {
		return fmt.Errorf("expected a selected version, got none")
	}
	return nil
}

func (w *tuiWorld) theSelectedVersionShaIs(sha string) error {
	sel, ok := w.model.SelectedVersion()
	if !ok {
		return fmt.Errorf("expected a selected version, got none")
	}
	if sel.SHA != sha {
		return fmt.Errorf("expected selected version sha %q, got %q", sha, sel.SHA)
	}
	return nil
}

func (w *tuiWorld) theSelectedChangesShasAre(list string) error {
	if got := shaList(w.model.SelectedChanges()); got != list {
		return fmt.Errorf("expected selected changes shas %q, got %q", list, got)
	}
	return nil
}

func (w *tuiWorld) theFocusIsOn(name string) error {
	want, ok := focusNames[name]
	if !ok {
		return fmt.Errorf("unknown focus %q", name)
	}
	if w.model.focus != want {
		return fmt.Errorf("expected focus %q, got %v", name, w.model.focus)
	}
	return nil
}

func (w *tuiWorld) theChangesScrollIs(n int) error {
	if w.model.changesScroll != n {
		return fmt.Errorf("expected changes scroll %d, got %d", n, w.model.changesScroll)
	}
	return nil
}

func (w *tuiWorld) theChangesScrollIsGreaterThan(n int) error {
	if !(w.model.changesScroll > n) {
		return fmt.Errorf("expected changes scroll greater than %d, got %d", n, w.model.changesScroll)
	}
	return nil
}

func (w *tuiWorld) theChangesScrollIsAtTheMaximum() error {
	max := w.model.maxChangesScroll()
	if w.model.changesScroll != max {
		return fmt.Errorf("expected changes scroll clamped to %d, got %d", max, w.model.changesScroll)
	}
	return nil
}

func (w *tuiWorld) theMaxChangesScrollIsPositive() error {
	if max := w.model.maxChangesScroll(); max <= 0 {
		return fmt.Errorf("expected a positive max changes scroll, got %d", max)
	}
	return nil
}

func InitializeScenario(sc *godog.ScenarioContext) {
	w := &tuiWorld{}
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		*w = tuiWorld{}
		return ctx, nil
	})

	sc.Step(`^the sample model$`, w.theSampleModel)
	sc.Step(`^the long-changelog model$`, w.theLongChangelogModel)
	sc.Step(`^a window size of (\d+) by (\d+)$`, w.aWindowSizeOf)
	sc.Step(`^I press "([^"]*)" (\d+) times$`, w.iPressTimes)
	sc.Step(`^I press "([^"]*)"$`, w.iPress)
	sc.Step(`^the selected plugin is "([^"]*)"$`, w.theSelectedPluginIs)
	sc.Step(`^there are (\d+) visible versions$`, w.thereAreVisibleVersions)
	sc.Step(`^the visible version shas are "([^"]*)"$`, w.theVisibleVersionShasAre)
	sc.Step(`^a version is selected$`, w.aVersionIsSelected)
	sc.Step(`^the selected version sha is "([^"]*)"$`, w.theSelectedVersionShaIs)
	sc.Step(`^the selected changes shas are "([^"]*)"$`, w.theSelectedChangesShasAre)
	sc.Step(`^the focus is on "([^"]*)"$`, w.theFocusIsOn)
	sc.Step(`^the changes scroll is (\d+)$`, w.theChangesScrollIs)
	sc.Step(`^the changes scroll is greater than (\d+)$`, w.theChangesScrollIsGreaterThan)
	sc.Step(`^the changes scroll is at the maximum$`, w.theChangesScrollIsAtTheMaximum)
	sc.Step(`^the max changes scroll is positive$`, w.theMaxChangesScrollIsPositive)
}
