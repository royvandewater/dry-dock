package dock

import (
	"path/filepath"
	"testing"

	"github.com/royvandewater/dry-dock/internal/lazy"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

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

func TestBuildAssemblesPluginsAndSkipsThoseWithoutCandidates(t *testing.T) {
	installDir := "/plugins"
	telDir := filepath.Join(installDir, "telescope.nvim")
	cmpDir := filepath.Join(installDir, "blink.cmp")

	locked := []lazy.Locked{
		{Name: "telescope.nvim", Branch: "master", Commit: "curTel"},
		{Name: "blink.cmp", Branch: "main", Commit: "curCmp"},
	}

	src := fakeSource{
		current: map[string]plugin.Version{
			telDir: {SHA: "curTel", Subject: "current tel"},
			cmpDir: {SHA: "curCmp", Subject: "current cmp"},
		},
		candidates: map[string][]plugin.Version{
			telDir: {{SHA: "newTel", Subject: "newer tel"}},
			cmpDir: {}, // up to date
		},
	}

	plugins, err := Build(installDir, locked, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plugins) != 1 {
		t.Fatalf("expected only plugins with candidates, got %d", len(plugins))
	}
	if plugins[0].Name != "telescope.nvim" {
		t.Fatalf("expected telescope.nvim, got %q", plugins[0].Name)
	}
	if plugins[0].Current.SHA != "curTel" {
		t.Fatalf("expected current curTel, got %q", plugins[0].Current.SHA)
	}
	if len(plugins[0].Candidates) != 1 || plugins[0].Candidates[0].SHA != "newTel" {
		t.Fatalf("unexpected candidates: %+v", plugins[0].Candidates)
	}
}
