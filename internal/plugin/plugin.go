// Package plugin models lazy.vim plugins and the versions they can be updated
// to, independent of where that data comes from (git, fixtures, etc).
package plugin

import (
	"regexp"
	"strings"
	"time"
)

// breakingType matches a Conventional Commits type/scope that carries a "!"
// breaking-change marker, e.g. "feat!:" or "feat(keymap)!:".
var breakingType = regexp.MustCompile(`^[a-zA-Z]+(\([^)]*\))?!:`)

// Version is a single point a plugin can move to. For lazy.vim plugins this
// maps to a git commit: SHA identifies it, Subject describes the change, and
// Date is when it was published (used to enforce a minimum release age).
type Version struct {
	SHA     string
	Subject string
	Date    time.Time
}

// Breaking reports whether the commit announces a breaking change, per the
// Conventional Commits convention: a "!" after the type/scope or a
// "BREAKING CHANGE" marker in the subject.
func (v Version) Breaking() bool {
	return breakingType.MatchString(v.Subject) ||
		strings.Contains(v.Subject, "BREAKING CHANGE") ||
		strings.Contains(v.Subject, "BREAKING-CHANGE")
}

// Plugin is an installed lazy.vim plugin together with the newer versions it
// could be updated to. Candidates are ordered most-recent first.
type Plugin struct {
	Name       string
	Current    Version
	Candidates []Version
}

// ChangesUpTo returns every change from the current version through the
// candidate at index i (into the most-recent-first Candidates slice). Because
// updating to a version pulls in all the intervening versions, the result spans
// candidates[i:], preserving the most-recent-first ordering.
func (p Plugin) ChangesUpTo(i int) []Version {
	return p.Candidates[i:]
}

// IncludesBreaking reports whether updating to the candidate identified by sha
// pulls in a breaking change. Updating moves the ref forward through every
// candidate at or older than sha, so a breaking commit taints sha and every
// newer candidate that would carry it along.
func (p Plugin) IncludesBreaking(sha string) bool {
	for i, c := range p.Candidates {
		if c.SHA != sha {
			continue
		}
		for _, pulled := range p.Candidates[i:] {
			if pulled.Breaking() {
				return true
			}
		}
		return false
	}
	return false
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
