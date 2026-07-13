package dock

import (
	"context"
	"fmt"
	"os"
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
	tags       map[string][]plugin.Version
}

func (f fakeSource) Current(dir, sha string) (plugin.Version, error) {
	return f.current[dir], nil
}

func (f fakeSource) Candidates(dir, from, ref string) ([]plugin.Version, error) {
	return f.candidates[dir], nil
}

func (f fakeSource) Tags(dir string) ([]plugin.Version, error) {
	return f.tags[dir], nil
}

func (f fakeSource) Describe(dir, sha string) (string, error) {
	return "", nil
}

type buildWorld struct {
	installDir string
	locked     []lazy.Locked
	src        fakeSource
	matchers   map[string]string
	plugins    []plugin.Plugin

	tmp      string
	lockPath string

	restored       []string
	breakingCommit string
	applyErr       error
	commitMessages []string
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

func (w *buildWorld) offersTagsFor(name string, table *godog.Table) error {
	var tags []plugin.Version
	for _, row := range table.Rows[1:] {
		tags = append(tags, plugin.Version{SHA: row.Cells[1].Value, Tag: row.Cells[0].Value})
	}
	w.src.tags[w.dir(name)] = tags
	return nil
}

func (w *buildWorld) hasVersionConstraint(name, constraint string) error {
	w.matchers[name] = constraint
	return nil
}

func (w *buildWorld) pluginHasReleasesOutsideItsConstraint(n, count int) error {
	if got := w.plugins[n-1].OutOfScope; got != count {
		return fmt.Errorf("expected plugin %d to have %d releases outside its constraint, got %d", n, count, got)
	}
	return nil
}

func (w *buildWorld) iBuildThePluginList() error {
	plugins, err := Build(w.installDir, w.locked, w.matchers, w.src)
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

func (w *buildWorld) pluginHasNoCandidates(n int) error {
	if got := len(w.plugins[n-1].Candidates); got != 0 {
		return fmt.Errorf("expected plugin %d to have no candidates, got %d", n, got)
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

func (w *buildWorld) tmpDir() (string, error) {
	if w.tmp == "" {
		tmp, err := os.MkdirTemp("", "dock-update-*")
		if err != nil {
			return "", err
		}
		w.tmp = tmp
		w.lockPath = filepath.Join(tmp, "lazy-lock.json")
	}
	return w.tmp, nil
}

func (w *buildWorld) aLockFilePinningToCommit(name, commit string) error {
	if _, err := w.tmpDir(); err != nil {
		return err
	}
	doc := fmt.Sprintf("{\n  %q: { \"branch\": \"master\", \"commit\": %q }\n}\n", name, commit)
	return os.WriteFile(w.lockPath, []byte(doc), 0o644)
}

func (w *buildWorld) noLockFileExists() error {
	_, err := w.tmpDir()
	return err
}

func (w *buildWorld) commitBreaksNvim(commit string) error {
	w.breakingCommit = commit
	return nil
}

func (w *buildWorld) iApplyTheUpdateToCommit(name, commit string) error {
	if _, err := w.tmpDir(); err != nil {
		return err
	}
	u := Updater{
		Config:  Config{LockPath: w.lockPath},
		Restore: func(pluginName string) error { w.restored = append(w.restored, pluginName); return nil },
		// HealthCheck reads whatever the lock file currently pins and treats the
		// designated commit as broken, mirroring "the new version breaks nvim".
		HealthCheck: func(pluginName string) error {
			c, err := lazy.CommitFor(w.lockPath, pluginName)
			if err != nil {
				return err
			}
			if w.breakingCommit != "" && c == w.breakingCommit {
				return fmt.Errorf("nvim broke on %s", c)
			}
			return nil
		},
		Commit: func(message string) error {
			w.commitMessages = append(w.commitMessages, message)
			return nil
		},
	}
	w.applyErr = u.Apply(name, commit)
	return nil
}

func (w *buildWorld) theLockFilePinsCommit(name, commit string) error {
	f, err := os.Open(w.lockPath)
	if err != nil {
		return err
	}
	defer f.Close()
	locked, err := lazy.ParseLock(f)
	if err != nil {
		return err
	}
	for _, l := range locked {
		if l.Name == name {
			if l.Commit != commit {
				return fmt.Errorf("expected %q pinned to %q, got %q", name, commit, l.Commit)
			}
			return nil
		}
	}
	return fmt.Errorf("plugin %q not in lock file", name)
}

func (w *buildWorld) lazyWasAskedToRestore(name string) error {
	for _, r := range w.restored {
		if r == name {
			return nil
		}
	}
	return fmt.Errorf("expected lazy.vim to restore %q, got %v", name, w.restored)
}

func (w *buildWorld) lazyWasNotAskedToRestore() error {
	if len(w.restored) != 0 {
		return fmt.Errorf("expected no restore, got %v", w.restored)
	}
	return nil
}

func (w *buildWorld) theConfigRepoIsCommittedWithMessage(message string) error {
	for _, m := range w.commitMessages {
		if m == message {
			return nil
		}
	}
	return fmt.Errorf("expected a commit with message %q, got %v", message, w.commitMessages)
}

func (w *buildWorld) theConfigRepoIsNotCommitted() error {
	if len(w.commitMessages) != 0 {
		return fmt.Errorf("expected no commit, got %v", w.commitMessages)
	}
	return nil
}

func (w *buildWorld) theUpdateFails() error {
	if w.applyErr == nil {
		return fmt.Errorf("expected the update to fail, got nil error")
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
				tags:       map[string][]plugin.Version{},
			},
			matchers: map[string]string{},
		}
		return ctx, nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		if w.tmp != "" {
			os.RemoveAll(w.tmp)
		}
		return ctx, nil
	})

	sc.Step(`^the install dir "([^"]*)"$`, w.theInstallDir)
	sc.Step(`^a lock file pinning "([^"]*)" to commit "([^"]*)"$`, w.aLockFilePinningToCommit)
	sc.Step(`^commit "([^"]*)" breaks nvim$`, w.commitBreaksNvim)
	sc.Step(`^no lock file exists$`, w.noLockFileExists)
	sc.Step(`^I apply the update for "([^"]*)" to commit "([^"]*)"$`, w.iApplyTheUpdateToCommit)
	sc.Step(`^the lock file pins "([^"]*)" to commit "([^"]*)"$`, w.theLockFilePinsCommit)
	sc.Step(`^lazy\.vim was asked to restore "([^"]*)"$`, w.lazyWasAskedToRestore)
	sc.Step(`^lazy\.vim was not asked to restore$`, w.lazyWasNotAskedToRestore)
	sc.Step(`^the update fails$`, w.theUpdateFails)
	sc.Step(`^the config repo is committed with message "([^"]*)"$`, w.theConfigRepoIsCommittedWithMessage)
	sc.Step(`^the config repo is not committed$`, w.theConfigRepoIsNotCommitted)
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
	sc.Step(`^plugin (\d+) has no candidates$`, w.pluginHasNoCandidates)
	sc.Step(`^the source offers tags for "([^"]*)":$`, w.offersTagsFor)
	sc.Step(`^"([^"]*)" has version constraint "([^"]*)"$`, w.hasVersionConstraint)
	sc.Step(`^plugin (\d+) has (\d+) release outside its constraint$`, w.pluginHasReleasesOutsideItsConstraint)
	sc.Step(`^plugin (\d+) has (\d+) releases outside its constraint$`, w.pluginHasReleasesOutsideItsConstraint)
}
