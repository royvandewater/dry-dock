package dock

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/royvandewater/dry-dock/internal/lazy"
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

type fakeSource struct {
	current    map[string]plugin.Version
	candidates map[string][]plugin.Version
}

func (f fakeSource) Current(dir, sha string) (plugin.Version, error) {
	return f.current[dir], nil
}

func (f fakeSource) Candidates(dir, from, ref string) ([]plugin.Version, error) {
	return f.candidates[dir], nil
}

type buildWorld struct {
	installDir string
	locked     []lazy.Locked
	src        fakeSource
	plugins    []plugin.Plugin
}

func (w *buildWorld) dir(name string) string {
	return filepath.Join(w.installDir, name)
}

func (w *buildWorld) theInstallDir(dir string) error {
	w.installDir = dir
	return nil
}

func (w *buildWorld) aLockedPlugin(name, branch, commit string) error {
	w.locked = append(w.locked, lazy.Locked{Name: name, Branch: branch, Commit: commit})
	return nil
}

func (w *buildWorld) currentVersionFor(sha, name string) error {
	w.src.current[w.dir(name)] = plugin.Version{SHA: sha, Subject: "current " + name}
	return nil
}

func (w *buildWorld) offersCandidates(list, name string) error {
	var versions []plugin.Version
	for _, sha := range strings.Split(list, ",") {
		versions = append(versions, plugin.Version{SHA: strings.TrimSpace(sha), Subject: "newer " + name})
	}
	w.src.candidates[w.dir(name)] = versions
	return nil
}

func (w *buildWorld) offersNoCandidates(name string) error {
	w.src.candidates[w.dir(name)] = []plugin.Version{}
	return nil
}

func (w *buildWorld) iBuildThePluginList() error {
	plugins, err := Build(w.installDir, w.locked, w.src)
	if err != nil {
		return err
	}
	w.plugins = plugins
	return nil
}

func (w *buildWorld) thereArePlugins(n int) error {
	if len(w.plugins) != n {
		return fmt.Errorf("expected %d plugins, got %d", n, len(w.plugins))
	}
	return nil
}

func (w *buildWorld) pluginIsNamed(n int, name string) error {
	if got := w.plugins[n-1].Name; got != name {
		return fmt.Errorf("expected plugin %d named %q, got %q", n, name, got)
	}
	return nil
}

func (w *buildWorld) pluginHasCurrentSha(n int, sha string) error {
	if got := w.plugins[n-1].Current.SHA; got != sha {
		return fmt.Errorf("expected plugin %d current sha %q, got %q", n, sha, got)
	}
	return nil
}

func (w *buildWorld) pluginHasCandidateShas(n int, list string) error {
	var got []string
	for _, c := range w.plugins[n-1].Candidates {
		got = append(got, c.SHA)
	}
	var want []string
	for _, sha := range strings.Split(list, ",") {
		want = append(want, strings.TrimSpace(sha))
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		return fmt.Errorf("expected plugin %d candidate shas %v, got %v", n, want, got)
	}
	return nil
}

func InitializeScenario(sc *godog.ScenarioContext) {
	w := &buildWorld{}
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		*w = buildWorld{
			src: fakeSource{
				current:    map[string]plugin.Version{},
				candidates: map[string][]plugin.Version{},
			},
		}
		return ctx, nil
	})

	sc.Step(`^the install dir "([^"]*)"$`, w.theInstallDir)
	sc.Step(`^a locked plugin "([^"]*)" on branch "([^"]*)" at commit "([^"]*)"$`, w.aLockedPlugin)
	sc.Step(`^the source reports current version "([^"]*)" for "([^"]*)"$`, w.currentVersionFor)
	sc.Step(`^the source offers candidates "([^"]*)" for "([^"]*)"$`, w.offersCandidates)
	sc.Step(`^the source offers no candidates for "([^"]*)"$`, w.offersNoCandidates)
	sc.Step(`^I build the plugin list$`, w.iBuildThePluginList)
	sc.Step(`^there is (\d+) plugin$`, w.thereArePlugins)
	sc.Step(`^there are (\d+) plugins$`, w.thereArePlugins)
	sc.Step(`^plugin (\d+) is named "([^"]*)"$`, w.pluginIsNamed)
	sc.Step(`^plugin (\d+) has current sha "([^"]*)"$`, w.pluginHasCurrentSha)
	sc.Step(`^plugin (\d+) has candidate shas "([^"]*)"$`, w.pluginHasCandidateShas)
}
