package dock

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

	tmp       string
	lockPath  string
	shaBySubj map[string]string
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

func (w *buildWorld) gitRun(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %v: %w\n%s", args, err, out)
	}
	return string(out), nil
}

func (w *buildWorld) aPluginCloneWithCommits(name, branch, subjects string) error {
	tmp, err := os.MkdirTemp("", "dock-update-*")
	if err != nil {
		return err
	}
	w.tmp = tmp
	w.installDir = filepath.Join(tmp, "install")
	w.lockPath = filepath.Join(tmp, "lazy-lock.json")

	dir := filepath.Join(w.installDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if _, err := w.gitRun(dir, "init", "-q", "-b", branch); err != nil {
		return err
	}
	for i, s := range strings.Split(subjects, ",") {
		s = strings.TrimSpace(s)
		if _, err := w.gitRun(dir, "commit", "-q", "--allow-empty",
			"--date", "2020-01-0"+string(rune('1'+i))+"T00:00:00", "-m", s); err != nil {
			return err
		}
		head, err := w.gitRun(dir, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		w.shaBySubj[s] = head[:40]
	}
	return nil
}

func (w *buildWorld) aLockFilePinning(name, subject string) error {
	doc := fmt.Sprintf("{\n  %q: { \"branch\": \"master\", \"commit\": %q }\n}\n", name, w.shaBySubj[subject])
	return os.WriteFile(w.lockPath, []byte(doc), 0o644)
}

func (w *buildWorld) iApplyTheUpdate(name, subject string) error {
	cfg := Config{LockPath: w.lockPath, InstallDir: w.installDir}
	return Updater{Config: cfg}.Apply(name, w.shaBySubj[subject])
}

func (w *buildWorld) theCloneHeadIsAt(name, subject string) error {
	head, err := w.gitRun(filepath.Join(w.installDir, name), "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if got := head[:40]; got != w.shaBySubj[subject] {
		return fmt.Errorf("expected HEAD at %q, got %q", w.shaBySubj[subject], got)
	}
	return nil
}

func (w *buildWorld) theLockFilePins(name, subject string) error {
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
			if l.Commit != w.shaBySubj[subject] {
				return fmt.Errorf("expected %q pinned to %q, got %q", name, w.shaBySubj[subject], l.Commit)
			}
			return nil
		}
	}
	return fmt.Errorf("plugin %q not in lock file", name)
}

func InitializeScenario(sc *godog.ScenarioContext) {
	w := &buildWorld{}
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		*w = buildWorld{
			src: fakeSource{
				current:    map[string]plugin.Version{},
				candidates: map[string][]plugin.Version{},
			},
			shaBySubj: map[string]string{},
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
	sc.Step(`^a plugin clone "([^"]*)" on branch "([^"]*)" with commits "([^"]*)"$`, w.aPluginCloneWithCommits)
	sc.Step(`^a lock file pinning "([^"]*)" to its "([^"]*)" commit$`, w.aLockFilePinning)
	sc.Step(`^I apply the update for "([^"]*)" to its "([^"]*)" commit$`, w.iApplyTheUpdate)
	sc.Step(`^the clone "([^"]*)" HEAD is at its "([^"]*)" commit$`, w.theCloneHeadIsAt)
	sc.Step(`^the lock file pins "([^"]*)" to its "([^"]*)" commit$`, w.theLockFilePins)
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
