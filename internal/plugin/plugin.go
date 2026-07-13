// Package plugin models lazy.vim plugins and the versions they can be updated
// to, independent of where that data comes from (git, fixtures, etc).
package plugin

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
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

	// Tag is the semver tag at this commit, when the version comes from a
	// release tag rather than a bare commit (empty otherwise).
	Tag string
}

// SelectInRange partitions release tags against a lazy.vim version constraint,
// relative to the version the plugin is currently on. It returns the tags that
// satisfy the constraint and are strictly newer than current (newest-first by
// semver), and the count of newer tags that fall outside the constraint — the
// releases dry-dock hides but still wants to hint at. Tags that are not valid
// semver are ignored.
//
// The current version is taken from the highest semver tag at currentSHA;
// commits carry several tags (e.g. "v1.17", "v1.17.0"), so matching by SHA and
// picking the semver-valid one avoids tripping over a non-semver sibling. When
// the pinned commit is untagged, currentTagHint (e.g. the nearest ancestor tag
// from `git describe`) stands in.
func SelectInRange(tags []Version, constraint, currentSHA, currentTagHint string) ([]Version, int, error) {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return nil, 0, err
	}

	current := highestSemverAt(tags, currentSHA)
	if current == nil && currentTagHint != "" {
		if hint, err := semver.NewVersion(currentTagHint); err == nil {
			current = hint
		}
	}

	type tagged struct {
		version Version
		semver  *semver.Version
	}
	var newer []tagged
	outsideSHAs := map[string]bool{}
	for _, v := range tags {
		if v.SHA == currentSHA {
			continue
		}
		sv, err := semver.NewVersion(v.Tag)
		if err != nil {
			continue
		}
		if current != nil && !sv.GreaterThan(current) {
			continue
		}
		if c.Check(sv) {
			newer = append(newer, tagged{v, sv})
		} else {
			outsideSHAs[v.SHA] = true
		}
	}

	// A commit often carries several equivalent tags (e.g. "v1.18" and
	// "v1.18.0"); sort newest-first, most-specific tag first, then keep one
	// entry per commit so a single release shows up once.
	sort.Slice(newer, func(i, j int) bool {
		if !newer[i].semver.Equal(newer[j].semver) {
			return newer[i].semver.GreaterThan(newer[j].semver)
		}
		return len(newer[i].version.Tag) > len(newer[j].version.Tag)
	})
	seen := map[string]bool{}
	var inRange []Version
	for _, t := range newer {
		if seen[t.version.SHA] {
			continue
		}
		seen[t.version.SHA] = true
		inRange = append(inRange, t.version)
	}
	return inRange, len(outsideSHAs), nil
}

// highestSemverAt returns the greatest semver tag pointing at sha, or nil when
// the commit carries no semver-valid tag.
func highestSemverAt(tags []Version, sha string) *semver.Version {
	var highest *semver.Version
	for _, v := range tags {
		if v.SHA != sha {
			continue
		}
		sv, err := semver.NewVersion(v.Tag)
		if err != nil {
			continue
		}
		if highest == nil || sv.GreaterThan(highest) {
			highest = sv
		}
	}
	return highest
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

	// Constraint is the lazy.vim version matcher (e.g. "1.*") when the plugin
	// pins a release range, empty otherwise. OutOfScope counts the newer
	// releases hidden because they fall outside that constraint.
	Constraint string
	OutOfScope int
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

// TooNew counts the candidate versions younger than the minimum release age —
// the ones filtered out of Installable for being too fresh to trust yet.
func (p Plugin) TooNew(now time.Time, minAge time.Duration) int {
	cutoff := now.Add(-minAge)

	count := 0
	for _, v := range p.Candidates {
		if v.Date.After(cutoff) {
			count++
		}
	}
	return count
}
