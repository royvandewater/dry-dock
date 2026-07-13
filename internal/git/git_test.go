package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

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

type gitWorld struct {
	dir       string
	shaBySubj map[string]string
	versions  []plugin.Version
	commit    plugin.Version
}

func (w *gitWorld) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = w.dir
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

func (w *gitWorld) aRepoWithCommits(branch, subjects string) error {
	dir, err := os.MkdirTemp("", "git-feature-*")
	if err != nil {
		return err
	}
	w.dir = dir

	if _, err := w.run("init", "-q", "-b", branch); err != nil {
		return err
	}
	for i, s := range strings.Split(subjects, ",") {
		s = strings.TrimSpace(s)
		if _, err := w.run("commit", "-q", "--allow-empty",
			"--date", "2020-01-0"+string(rune('1'+i))+"T00:00:00", "-m", s); err != nil {
			return err
		}
		head, err := w.run("rev-parse", "HEAD")
		if err != nil {
			return err
		}
		w.shaBySubj[s] = head[:40]
	}
	return nil
}

func (w *gitWorld) iLogBetween(from, ref string) error {
	versions, err := LogBetween(w.dir, w.shaBySubj[from], ref)
	if err != nil {
		return err
	}
	w.versions = versions
	return nil
}

func (w *gitWorld) iReadTheCommitFor(subject string) error {
	commit, err := Commit(w.dir, w.shaBySubj[subject])
	if err != nil {
		return err
	}
	w.commit = commit
	return nil
}

func (w *gitWorld) iCheckoutCommit(subject string) error {
	return Checkout(w.dir, w.shaBySubj[subject])
}

func (w *gitWorld) headIsAtCommit(subject string) error {
	head, err := w.run("rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if got := head[:40]; got != w.shaBySubj[subject] {
		return fmt.Errorf("expected HEAD at %q, got %q", w.shaBySubj[subject], got)
	}
	return nil
}

func (w *gitWorld) thereAreVersions(n int) error {
	if len(w.versions) != n {
		return fmt.Errorf("expected %d versions, got %d", n, len(w.versions))
	}
	return nil
}

func (w *gitWorld) theVersionSubjectsAre(list string) error {
	var got []string
	for _, v := range w.versions {
		got = append(got, v.Subject)
	}
	var want []string
	for _, s := range strings.Split(list, ",") {
		want = append(want, strings.TrimSpace(s))
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		return fmt.Errorf("expected subjects %v, got %v", want, got)
	}
	return nil
}

func (w *gitWorld) versionHasTheShaOfCommit(n int, subject string) error {
	if got := w.versions[n-1].SHA; got != w.shaBySubj[subject] {
		return fmt.Errorf("expected version %d sha %q, got %q", n, w.shaBySubj[subject], got)
	}
	return nil
}

func (w *gitWorld) versionHasANonZeroDate(n int) error {
	if w.versions[n-1].Date.IsZero() {
		return fmt.Errorf("expected version %d to have a non-zero date", n)
	}
	return nil
}

func (w *gitWorld) theCommitSubjectIs(subject string) error {
	if w.commit.Subject != subject {
		return fmt.Errorf("expected commit subject %q, got %q", subject, w.commit.Subject)
	}
	return nil
}

func (w *gitWorld) theCommitShaIsTheShaOfCommit(subject string) error {
	if w.commit.SHA != w.shaBySubj[subject] {
		return fmt.Errorf("expected commit sha %q, got %q", w.shaBySubj[subject], w.commit.SHA)
	}
	return nil
}

func InitializeScenario(sc *godog.ScenarioContext) {
	w := &gitWorld{}
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		*w = gitWorld{shaBySubj: map[string]string{}}
		return ctx, nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		if w.dir != "" {
			os.RemoveAll(w.dir)
		}
		return ctx, nil
	})

	sc.Step(`^a repo with commits on "([^"]*)": "([^"]*)"$`, w.aRepoWithCommits)
	sc.Step(`^I log between commit "([^"]*)" and "([^"]*)"$`, w.iLogBetween)
	sc.Step(`^I read the commit for "([^"]*)"$`, w.iReadTheCommitFor)
	sc.Step(`^I checkout commit "([^"]*)"$`, w.iCheckoutCommit)
	sc.Step(`^HEAD is at commit "([^"]*)"$`, w.headIsAtCommit)
	sc.Step(`^there are (\d+) versions$`, w.thereAreVersions)
	sc.Step(`^the version subjects are "([^"]*)"$`, w.theVersionSubjectsAre)
	sc.Step(`^version (\d+) has the sha of commit "([^"]*)"$`, w.versionHasTheShaOfCommit)
	sc.Step(`^version (\d+) has a non-zero date$`, w.versionHasANonZeroDate)
	sc.Step(`^the commit subject is "([^"]*)"$`, w.theCommitSubjectIs)
	sc.Step(`^the commit sha is the sha of commit "([^"]*)"$`, w.theCommitShaIsTheShaOfCommit)
}
