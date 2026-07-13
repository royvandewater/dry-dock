// Package plugin models lazy.vim plugins and the versions they can be updated
// to, independent of where that data comes from (git, fixtures, etc).
package plugin

import "time"

// Version is a single point a plugin can move to. For lazy.vim plugins this
// maps to a git commit: SHA identifies it, Subject describes the change, and
// Date is when it was published (used to enforce a minimum release age).
type Version struct {
	SHA     string
	Subject string
	Date    time.Time
}

// Plugin is an installed lazy.vim plugin together with the newer versions it
// could be updated to. Candidates are ordered most-recent first.
type Plugin struct {
	Name       string
	Current    Version
	Candidates []Version
}

// Installable returns the candidate versions old enough to install given a
// minimum release age, preserving the most-recent-first ordering.
func (p Plugin) Installable(now time.Time, minAge time.Duration) []Version {
	cutoff := now.Add(-minAge)

	installable := make([]Version, 0, len(p.Candidates))
	for _, v := range p.Candidates {
		if !v.Date.After(cutoff) {
			installable = append(installable, v)
		}
	}
	return installable
}
