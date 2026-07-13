package lazy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cucumber/godog"
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

type lockWorld struct {
	doc    string
	locked []Locked
	result string
}

func (w *lockWorld) find(name string) (Locked, error) {
	for _, l := range w.locked {
		if l.Name == name {
			return l, nil
		}
	}
	return Locked{}, fmt.Errorf("locked plugin %q not found", name)
}

func (w *lockWorld) aLazyLockDocument(doc *godog.DocString) error {
	w.doc = doc.Content
	return nil
}

func (w *lockWorld) iParseTheLockDocument() error {
	locked, err := ParseLock(strings.NewReader(w.doc))
	if err != nil {
		return err
	}
	w.locked = locked
	return nil
}

func (w *lockWorld) iSetCommitTo(name, commit string) error {
	result, err := SetCommit(w.doc, name, commit)
	if err != nil {
		return err
	}
	w.result = result
	return nil
}

func (w *lockWorld) theResultingDocumentIs(doc *godog.DocString) error {
	if got := strings.TrimSpace(w.result); got != strings.TrimSpace(doc.Content) {
		return fmt.Errorf("expected document:\n%s\ngot:\n%s", doc.Content, got)
	}
	return nil
}

func (w *lockWorld) thereAreLockedPlugins(n int) error {
	if len(w.locked) != n {
		return fmt.Errorf("expected %d locked plugins, got %d", n, len(w.locked))
	}
	return nil
}

func (w *lockWorld) theLockedPluginHasBranch(name, branch string) error {
	l, err := w.find(name)
	if err != nil {
		return err
	}
	if l.Branch != branch {
		return fmt.Errorf("expected branch %q, got %q", branch, l.Branch)
	}
	return nil
}

func (w *lockWorld) theLockedPluginHasCommit(name, commit string) error {
	l, err := w.find(name)
	if err != nil {
		return err
	}
	if l.Commit != commit {
		return fmt.Errorf("expected commit %q, got %q", commit, l.Commit)
	}
	return nil
}

func InitializeScenario(sc *godog.ScenarioContext) {
	w := &lockWorld{}
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		*w = lockWorld{}
		return ctx, nil
	})

	sc.Step(`^a lazy-lock\.json document:$`, w.aLazyLockDocument)
	sc.Step(`^I parse the lock document$`, w.iParseTheLockDocument)
	sc.Step(`^I set "([^"]*)" commit to "([^"]*)"$`, w.iSetCommitTo)
	sc.Step(`^the resulting document is:$`, w.theResultingDocumentIs)
	sc.Step(`^there are (\d+) locked plugins$`, w.thereAreLockedPlugins)
	sc.Step(`^the locked plugin "([^"]*)" has branch "([^"]*)"$`, w.theLockedPluginHasBranch)
	sc.Step(`^the locked plugin "([^"]*)" has commit "([^"]*)"$`, w.theLockedPluginHasCommit)
}
