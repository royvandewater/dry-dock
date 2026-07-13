package dock

import (
	"github.com/royvandewater/dry-dock/internal/git"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

// GitSource is the production VersionSource, backed by the git CLI.
type GitSource struct{}

func (GitSource) Current(dir, sha string) (plugin.Version, error) {
	return git.Commit(dir, sha)
}

func (GitSource) Candidates(dir, from, ref string) ([]plugin.Version, error) {
	return git.LogBetween(dir, from, ref)
}

func (GitSource) Tags(dir string) ([]plugin.Version, error) {
	return git.Tags(dir)
}

func (GitSource) Describe(dir, sha string) (string, error) {
	return git.DescribeTag(dir, sha)
}
