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
	now    time.Time
	minAge time.Duration
	plugin Plugin
	result []Version
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

func (w *pluginWorld) iRequestTheChangesUpToIndex(i int) error {
	w.result = w.plugin.ChangesUpTo(i)
	return nil
}

func (w *pluginWorld) iComputeTheInstallableVersions() error {
	w.result = w.plugin.Installable(w.now, w.minAge)
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
	sc.Step(`^I request the changes up to index (\d+)$`, w.iRequestTheChangesUpToIndex)
	sc.Step(`^I compute the installable versions$`, w.iComputeTheInstallableVersions)
	sc.Step(`^the resulting shas are "([^"]*)"$`, w.theResultingShasAre)
}
