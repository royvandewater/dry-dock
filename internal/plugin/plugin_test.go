package plugin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

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

type pluginWorld struct {
	now         time.Time
	minAge      time.Duration
	plugin      Plugin
	result      []Version
	version     Version
	tagVersions []Version
	currentTag  string
	constraint  string
	outside     int
	tooNew      int
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func shas(versions []Version) []string {
	out := make([]string, len(versions))
	for i, v := range versions {
		out[i] = v.SHA
	}
	return out
}

func (w *pluginWorld) theCurrentTimeIs(date string) error {
	now, err := time.Parse("2006-01-02", date)
	if err != nil {
		return err
	}
	w.now = now
	return nil
}

func (w *pluginWorld) aMinimumReleaseAgeOfDays(days int) error {
	w.minAge = time.Duration(days) * 24 * time.Hour
	return nil
}

func (w *pluginWorld) aPluginWhoseCandidatesNewestFirstAre(list string) error {
	var candidates []Version
	for _, sha := range splitList(list) {
		candidates = append(candidates, Version{SHA: sha})
	}
	w.plugin = Plugin{Name: "test.nvim", Current: Version{SHA: "cur"}, Candidates: candidates}
	return nil
}

func (w *pluginWorld) aPluginWithCandidates(table *godog.Table) error {
	var candidates []Version
	for _, row := range table.Rows[1:] {
		ageDays, err := strconv.Atoi(row.Cells[2].Value)
		if err != nil {
			return err
		}
		candidates = append(candidates, Version{
			SHA:     row.Cells[0].Value,
			Subject: row.Cells[1].Value,
			Date:    w.now.Add(-time.Duration(ageDays) * 24 * time.Hour),
		})
	}
	w.plugin = Plugin{Name: "test.nvim", Current: Version{SHA: "cur"}, Candidates: candidates}
	return nil
}

func (w *pluginWorld) aCommitWithSubject(subject string) error {
	w.version = Version{Subject: subject}
	return nil
}

func (w *pluginWorld) theCommitIsBreaking() error {
	if !w.version.Breaking() {
		return fmt.Errorf("expected %q to be breaking", w.version.Subject)
	}
	return nil
}

func (w *pluginWorld) theCommitIsNotBreaking() error {
	if w.version.Breaking() {
		return fmt.Errorf("expected %q not to be breaking", w.version.Subject)
	}
	return nil
}

func (w *pluginWorld) tags(list string) error {
	for _, t := range splitList(list) {
		w.tagVersions = append(w.tagVersions, Version{SHA: t, Tag: t})
	}
	return nil
}

func (w *pluginWorld) tagReleases(table *godog.Table) error {
	for _, row := range table.Rows[1:] {
		w.tagVersions = append(w.tagVersions, Version{SHA: row.Cells[1].Value, Tag: row.Cells[0].Value})
	}
	return nil
}

func (w *pluginWorld) theCurrentTagIs(tag string) error {
	w.currentTag = tag
	return nil
}

func (w *pluginWorld) theCurrentCommitIs(sha string) error {
	w.currentTag = sha
	return nil
}

func (w *pluginWorld) theVersionConstraintIs(constraint string) error {
	w.constraint = constraint
	return nil
}

func (w *pluginWorld) iSelectTheInRangeVersions() error {
	inRange, outside, err := SelectInRange(w.tagVersions, w.constraint, w.currentTag, "")
	if err != nil {
		return err
	}
	w.result = inRange
	w.outside = outside
	return nil
}

func (w *pluginWorld) thereAreNewerReleasesOutsideTheRange(n int) error {
	if w.outside != n {
		return fmt.Errorf("expected %d newer releases outside the range, got %d", n, w.outside)
	}
	return nil
}

func (w *pluginWorld) updatingToIncludesABreakingChange(sha string) error {
	if !w.plugin.IncludesBreaking(sha) {
		return fmt.Errorf("expected updating to %q to include a breaking change", sha)
	}
	return nil
}

func (w *pluginWorld) updatingToDoesNotIncludeABreakingChange(sha string) error {
	if w.plugin.IncludesBreaking(sha) {
		return fmt.Errorf("expected updating to %q not to include a breaking change", sha)
	}
	return nil
}

func (w *pluginWorld) iRequestTheChangesUpToIndex(i int) error {
	w.result = w.plugin.ChangesUpTo(i)
	return nil
}

func (w *pluginWorld) iComputeTheInstallableVersions() error {
	w.result = w.plugin.Installable(w.now, w.minAge)
	return nil
}

func (w *pluginWorld) iCountTheVersionsTooNewToInstall() error {
	w.tooNew = w.plugin.TooNew(w.now, w.minAge)
	return nil
}

func (w *pluginWorld) versionsAreTooNew(n int) error {
	if w.tooNew != n {
		return fmt.Errorf("expected %d versions too new, got %d", n, w.tooNew)
	}
	return nil
}

func (w *pluginWorld) theResultingShasAre(list string) error {
	want := splitList(list)
	got := shas(w.result)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		return fmt.Errorf("expected shas %v, got %v", want, got)
	}
	return nil
}

func InitializeScenario(sc *godog.ScenarioContext) {
	w := &pluginWorld{}
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		*w = pluginWorld{}
		return ctx, nil
	})

	sc.Step(`^the current time is "([^"]*)"$`, w.theCurrentTimeIs)
	sc.Step(`^a minimum release age of (\d+) days$`, w.aMinimumReleaseAgeOfDays)
	sc.Step(`^a plugin whose candidates newest-first are "([^"]*)"$`, w.aPluginWhoseCandidatesNewestFirstAre)
	sc.Step(`^a plugin with candidates:$`, w.aPluginWithCandidates)
	sc.Step(`^a commit with subject "([^"]*)"$`, w.aCommitWithSubject)
	sc.Step(`^the commit is breaking$`, w.theCommitIsBreaking)
	sc.Step(`^the commit is not breaking$`, w.theCommitIsNotBreaking)
	sc.Step(`^updating to "([^"]*)" includes a breaking change$`, w.updatingToIncludesABreakingChange)
	sc.Step(`^updating to "([^"]*)" does not include a breaking change$`, w.updatingToDoesNotIncludeABreakingChange)
	sc.Step(`^tags "([^"]*)"$`, w.tags)
	sc.Step(`^tag releases:$`, w.tagReleases)
	sc.Step(`^the current tag is "([^"]*)"$`, w.theCurrentTagIs)
	sc.Step(`^the current commit is "([^"]*)"$`, w.theCurrentCommitIs)
	sc.Step(`^the version constraint is "([^"]*)"$`, w.theVersionConstraintIs)
	sc.Step(`^I select the in-range versions$`, w.iSelectTheInRangeVersions)
	sc.Step(`^there is (\d+) newer release outside the range$`, w.thereAreNewerReleasesOutsideTheRange)
	sc.Step(`^there are (\d+) newer releases outside the range$`, w.thereAreNewerReleasesOutsideTheRange)
	sc.Step(`^I request the changes up to index (\d+)$`, w.iRequestTheChangesUpToIndex)
	sc.Step(`^I compute the installable versions$`, w.iComputeTheInstallableVersions)
	sc.Step(`^I count the versions too new to install$`, w.iCountTheVersionsTooNewToInstall)
	sc.Step(`^(\d+) versions are too new$`, w.versionsAreTooNew)
	sc.Step(`^the resulting shas are "([^"]*)"$`, w.theResultingShasAre)
}
