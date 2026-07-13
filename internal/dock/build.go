// Package dock assembles lazy.vim lock data and git history into the plugin
// list the TUI renders.
package dock

import (
	"path/filepath"

	"github.com/royvandewater/dry-dock/internal/lazy"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

// VersionSource resolves a plugin clone's current version and the newer
// versions it can move to.
type VersionSource interface {
	Current(dir, sha string) (plugin.Version, error)
	Candidates(dir, from, ref string) ([]plugin.Version, error)
}

// Build turns locked plugins into plugin.Plugin values, dropping any that have
// no newer versions to offer.
func Build(installDir string, locked []lazy.Locked, src VersionSource) ([]plugin.Plugin, error) {
	var plugins []plugin.Plugin
	for _, l := range locked {
		dir := filepath.Join(installDir, l.Name)

		candidates, err := src.Candidates(dir, l.Commit, "origin/"+l.Branch)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			continue
		}

		current, err := src.Current(dir, l.Commit)
		if err != nil {
			return nil, err
		}

		plugins = append(plugins, plugin.Plugin{
			Name:       l.Name,
			Current:    current,
			Candidates: candidates,
		})
	}
	return plugins, nil
}
