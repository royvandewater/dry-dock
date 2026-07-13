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

// constrained builds a model with one versioned plugin pinned to v1.10.0 under a
// "1.*" constraint. It has an installable in-range release, an in-range release
// that is still too new, and two out-of-range releases (v2.x) hidden by the
// constraint — exercising both collapsible headers.
func constrained() Model {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	p := plugin.Plugin{
		Name:       "vim-pinned",
		Current:    plugin.Version{SHA: "pinCur", Tag: "v1.10.0"},
		Constraint: "1.*",
		Candidates: []plugin.Version{
			{SHA: "c112", Tag: "v1.12.0", Subject: "in-range fresh", Date: now.Add(-2 * day)}, // too new
			{SHA: "c111", Tag: "v1.11.0", Subject: "in-range ripe", Date: now.Add(-30 * day)}, // installable
		},
		OutOfScope: []plugin.Version{
			{SHA: "c210", Tag: "v2.1.0", Subject: "major two-one", Date: now.Add(-3 * day)},
			{SHA: "c200", Tag: "v2.0.0", Subject: "major two-oh", Date: now.Add(-20 * day)},
		},
	}
	return New([]plugin.Plugin{p}, now, 14*day)
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

// allTooNew builds a model with one plugin whose every candidate is younger
// than the 14-day minimum age, so nothing is installable.
func allTooNew() Model {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	p := plugin.Plugin{
		Name:    "fresh.nvim",
		Current: plugin.Version{SHA: "cur"},
		Candidates: []plugin.Version{
			{SHA: "aaa", Subject: "one", Date: now.Add(-1 * day)},
			{SHA: "bbb", Subject: "two", Date: now.Add(-2 * day)},
			{SHA: "ccc", Subject: "three", Date: now.Add(-3 * day)},
		},
	}
	return New([]plugin.Plugin{p}, now, 14*day)
}

// mixed builds a model whose plugins arrive in an order that must be re-sorted:
// an up-to-date plugin, an updatable one, and one whose only versions are too
// new. Only the updatable plugin has installable versions, so it should rise to
// the top while the other two sink to the bottom in their original order.
func mixed() Model {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	stale := plugin.Plugin{
		Name:    "stale.nvim",
		Current: plugin.Version{SHA: "staleCur"},
	}
	updatable := plugin.Plugin{
		Name:    "updatable.nvim",
		Current: plugin.Version{SHA: "upCur"},
		Candidates: []plugin.Version{
			{SHA: "upA", Subject: "one", Date: now.Add(-40 * day)},
		},
	}
	fresh := plugin.Plugin{
		Name:    "fresh.nvim",
		Current: plugin.Version{SHA: "freshCur"},
		Candidates: []plugin.Version{
			{SHA: "fA", Subject: "one", Date: now.Add(-2 * day)},
		},
	}

	return New([]plugin.Plugin{stale, updatable, fresh}, now, 14*day)
}

var keys = map[string]tea.KeyType{
	"up":    tea.KeyUp,
	"down":  tea.KeyDown,
	"left":  tea.KeyLeft,
	"right": tea.KeyRight,
	"enter": tea.KeyEnter,
	"esc":   tea.KeyEsc,
}

// recordingUpdater is a test double for the Applier the model calls to perform
// an update. It records the last request and returns a configurable error.
type recordingUpdater struct {
	plugin, sha string
	called      bool
	err         error
}

func (u *recordingUpdater) Apply(plugin, sha string) error {
	u.plugin, u.sha, u.called = plugin, sha, true
	return u.err
}

var focusNames = map[string]focus{
	"plugins":  focusPlugins,
	"versions": focusVersions,
	"changes":  focusChanges,
}

type tuiWorld struct {
	model   Model
	updater *recordingUpdater
	lastCmd tea.Cmd
}

func (w *tuiWorld) send(msg tea.Msg) {
	next, cmd := w.model.Update(msg)
	w.model = next.(Model)
	w.lastCmd = cmd
}

func (w *tuiWorld) theSampleModel() error {
	w.model = sample()
	return nil
}

func (w *tuiWorld) theLongChangelogModel() error {
	w.model = longChangelog()
	return nil
}

func (w *tuiWorld) theMixedModel() error {
	w.model = mixed()
	return nil
}

func (w *tuiWorld) theConstrainedModel() error {
	w.model = constrained()
	return nil
}

func (w *tuiWorld) theConstrainedModelWithARecordingUpdater() error {
	w.updater = &recordingUpdater{}
	w.model = constrained().WithApplier(w.updater)
	return nil
}

func (w *tuiWorld) noVersionIsSelected() error {
	if _, ok := w.model.SelectedVersion(); ok {
		return fmt.Errorf("expected no selected version, got one")
	}
	return nil
}

func (w *tuiWorld) theUpdaterWasNotCalled() error {
	if w.updater.called {
		return fmt.Errorf("expected the updater not to be called")
	}
	return nil
}

func (w *tuiWorld) pluginIsMuted(n int) error {
	if !w.model.upToDate(n - 1) {
		return fmt.Errorf("expected plugin %d to be muted", n)
	}
	return nil
}

func (w *tuiWorld) pluginIsNotMuted(n int) error {
	if w.model.upToDate(n - 1) {
		return fmt.Errorf("expected plugin %d not to be muted", n)
	}
	return nil
}

func (w *tuiWorld) theHeaderShows(substr string) error {
	if got := w.model.renderHeader(); !strings.Contains(got, substr) {
		return fmt.Errorf("expected header to contain %q, got %q", substr, got)
	}
	return nil
}

func (w *tuiWorld) aModelWhoseOnlyPluginHasVersionsAllTooNew(n int) error {
	w.model = allTooNew()
	return nil
}

func (w *tuiWorld) theVersionsPaneShows(substr string) error {
	if got := w.model.versionContent(w.model.layout()); !strings.Contains(got, substr) {
		return fmt.Errorf("expected versions pane to contain %q, got %q", substr, got)
	}
	return nil
}

func (w *tuiWorld) theSampleModelWithARecordingUpdater() error {
	w.updater = &recordingUpdater{}
	w.model = sample().WithApplier(w.updater)
	return nil
}

func (w *tuiWorld) theSampleModelWithAFailingUpdater() error {
	w.updater = &recordingUpdater{err: fmt.Errorf("boom")}
	w.model = sample().WithApplier(w.updater)
	return nil
}

func (w *tuiWorld) iProcessPendingCommands() error {
	if w.lastCmd == nil {
		return fmt.Errorf("no pending command to process")
	}
	w.send(w.lastCmd())
	return nil
}

func (w *tuiWorld) theUpdaterApplied(plugin, sha string) error {
	if !w.updater.called {
		return fmt.Errorf("expected the updater to be called")
	}
	if w.updater.plugin != plugin || w.updater.sha != sha {
		return fmt.Errorf("expected apply(%q, %q), got apply(%q, %q)",
			plugin, sha, w.updater.plugin, w.updater.sha)
	}
	return nil
}

func (w *tuiWorld) theStatusContains(substr string) error {
	if !strings.Contains(w.model.status, substr) {
		return fmt.Errorf("expected status to contain %q, got %q", substr, w.model.status)
	}
	return nil
}

func (w *tuiWorld) theStatusIsEmpty() error {
	if w.model.status != "" {
		return fmt.Errorf("expected empty status, got %q", w.model.status)
	}
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

func (w *tuiWorld) thereAreNPlugins(n int) error {
	if got := len(w.model.plugins); got != n {
		return fmt.Errorf("expected %d plugins, got %d", n, got)
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
	sc.Step(`^the sample model with a recording updater$`, w.theSampleModelWithARecordingUpdater)
	sc.Step(`^the sample model with a failing updater$`, w.theSampleModelWithAFailingUpdater)
	sc.Step(`^I process pending commands$`, w.iProcessPendingCommands)
	sc.Step(`^the updater applied "([^"]*)" at "([^"]*)"$`, w.theUpdaterApplied)
	sc.Step(`^the status contains "([^"]*)"$`, w.theStatusContains)
	sc.Step(`^the status is empty$`, w.theStatusIsEmpty)
	sc.Step(`^the long-changelog model$`, w.theLongChangelogModel)
	sc.Step(`^the mixed model$`, w.theMixedModel)
	sc.Step(`^the constrained model$`, w.theConstrainedModel)
	sc.Step(`^the constrained model with a recording updater$`, w.theConstrainedModelWithARecordingUpdater)
	sc.Step(`^no version is selected$`, w.noVersionIsSelected)
	sc.Step(`^the updater was not called$`, w.theUpdaterWasNotCalled)
	sc.Step(`^plugin (\d+) is muted$`, w.pluginIsMuted)
	sc.Step(`^plugin (\d+) is not muted$`, w.pluginIsNotMuted)
	sc.Step(`^the header shows "([^"]*)"$`, w.theHeaderShows)
	sc.Step(`^a model whose only plugin has (\d+) versions all too new$`, w.aModelWhoseOnlyPluginHasVersionsAllTooNew)
	sc.Step(`^the versions pane shows "([^"]*)"$`, w.theVersionsPaneShows)
	sc.Step(`^a window size of (\d+) by (\d+)$`, w.aWindowSizeOf)
	sc.Step(`^I press "([^"]*)" (\d+) times$`, w.iPressTimes)
	sc.Step(`^I press "([^"]*)"$`, w.iPress)
	sc.Step(`^the selected plugin is "([^"]*)"$`, w.theSelectedPluginIs)
	sc.Step(`^there is (\d+) plugin$`, w.thereAreNPlugins)
	sc.Step(`^there are (\d+) plugins$`, w.thereAreNPlugins)
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
