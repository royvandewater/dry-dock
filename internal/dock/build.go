// Package dock assembles lazy.vim lock data and git history into the plugin
// list the TUI renders.
package dock

import (
	"path/filepath"

	"github.com/royvandewater/dry-dock/internal/lazy"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

// VersionSource resolves a plugin clone's current version, the newer commits it
// can move to, and its release tags.
type VersionSource interface {
	Current(dir, sha string) (plugin.Version, error)
	Candidates(dir, from, ref string) ([]plugin.Version, error)
	Tags(dir string) ([]plugin.Version, error)
	Describe(dir, sha string) (string, error)
}

// Build turns locked plugins into plugin.Plugin values, keeping even those with
// no newer versions so the TUI can list them as up to date. matchers maps a
// plugin name to its lazy.vim version constraint (e.g. "1.*"); a plugin with a
// constraint is offered only the release tags that satisfy it, and records how
// many newer releases fall outside it.
func Build(installDir string, locked []lazy.Locked, matchers map[string]string, src VersionSource) ([]plugin.Plugin, error) {
	var plugins []plugin.Plugin
	for _, l := range locked {
		dir := filepath.Join(installDir, l.Name)

		p, err := buildOne(dir, l, matchers[l.Name], src)
		if err != nil {
			return nil, err
		}
		if p != nil {
			plugins = append(plugins, *p)
		}
	}
	return plugins, nil
}

// buildOne assembles a single plugin. A plugin with no newer versions is still
// returned (with an empty Candidates slice) so the TUI can show it as up to
// date. Plugins with a version constraint are tag-based; the rest track their
// branch commit by commit.
func buildOne(dir string, l lazy.Locked, constraint string, src VersionSource) (*plugin.Plugin, error) {
	current, err := src.Current(dir, l.Commit)
	if err != nil {
		return nil, err
	}

	if constraint == "" {
		candidates, err := src.Candidates(dir, l.Commit, "origin/"+l.Branch)
		if err != nil {
			return nil, err
		}
		return &plugin.Plugin{Name: l.Name, Current: current, Candidates: candidates}, nil
	}

	tags, err := src.Tags(dir)
	if err != nil {
		return nil, err
	}
	hint, err := src.Describe(dir, l.Commit)
	if err != nil {
		return nil, err
	}
	inRange, outOfScope, err := plugin.SelectInRange(tags, constraint, l.Commit, hint)
	if err != nil {
		return nil, err
	}
	// A versioned plugin is pinned to a release tag; surface it so the TUI can
	// show the version number rather than the bare commit SHA.
	current.Tag = plugin.CurrentTag(tags, l.Commit, hint)
	return &plugin.Plugin{
		Name:       l.Name,
		Current:    current,
		Candidates: inRange,
		Constraint: constraint,
		OutOfScope: outOfScope,
	}, nil
}
