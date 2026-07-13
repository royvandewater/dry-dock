package plugin

import (
	"testing"
	"time"
)

func TestInstallableFiltersVersionsYoungerThanMinAge(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	minAge := 14 * 24 * time.Hour

	tooYoung := Version{SHA: "aaa", Subject: "recent", Date: now.Add(-2 * 24 * time.Hour)}
	oldEnough := Version{SHA: "bbb", Subject: "seasoned", Date: now.Add(-30 * 24 * time.Hour)}

	p := Plugin{
		Name:       "telescope.nvim",
		Current:    Version{SHA: "ccc", Subject: "current", Date: now.Add(-90 * 24 * time.Hour)},
		Candidates: []Version{tooYoung, oldEnough},
	}

	got := p.Installable(now, minAge)

	if len(got) != 1 {
		t.Fatalf("expected 1 installable version, got %d", len(got))
	}
	if got[0].SHA != "bbb" {
		t.Fatalf("expected only the seasoned version, got %q", got[0].SHA)
	}
}
