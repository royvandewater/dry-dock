package plugin

import (
	"testing"
	"time"
)

func TestChangesUpToAccumulatesFromCurrentThroughSelected(t *testing.T) {
	v3 := Version{SHA: "v3", Subject: "newest"}
	v2 := Version{SHA: "v2", Subject: "middle"}
	v1 := Version{SHA: "v1", Subject: "just after current"}

	p := Plugin{
		Name:       "telescope.nvim",
		Current:    Version{SHA: "cur"},
		Candidates: []Version{v3, v2, v1}, // most-recent first
	}

	// Selecting v2 (index 1) should include everything from current up to and
	// including v2: v2 and v1, still most-recent first.
	got := p.ChangesUpTo(1)

	want := []string{"v2", "v1"}
	if len(got) != len(want) {
		t.Fatalf("expected %d changes, got %d", len(want), len(got))
	}
	for i, sha := range want {
		if got[i].SHA != sha {
			t.Fatalf("change %d: expected %q, got %q", i, sha, got[i].SHA)
		}
	}
}

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
