// Package git is a thin adapter over the git CLI that turns a plugin's clone
// into plugin.Version values.
package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/royvandewater/dry-dock/internal/plugin"
)

// fields is the null-separated log format: SHA, commit unix time, subject.
const fields = "%H%x00%ct%x00%s"

// LogBetween returns the commits reachable from ref but not from `from`,
// newest first — the versions a plugin can be updated to.
func LogBetween(dir, from, ref string) ([]plugin.Version, error) {
	out, err := gitOutput(dir, "log", "--format="+fields, from+".."+ref)
	if err != nil {
		return nil, err
	}
	return parseVersions(out)
}

// Commit reads a single commit as a plugin.Version.
func Commit(dir, sha string) (plugin.Version, error) {
	out, err := gitOutput(dir, "log", "-1", "--format="+fields, sha)
	if err != nil {
		return plugin.Version{}, err
	}
	versions, err := parseVersions(out)
	if err != nil {
		return plugin.Version{}, err
	}
	if len(versions) != 1 {
		return plugin.Version{}, fmt.Errorf("expected 1 commit for %s, got %d", sha, len(versions))
	}
	return versions[0], nil
}

// Fetch updates remote-tracking refs so newer commits become visible.
func Fetch(dir string) error {
	_, err := gitOutput(dir, "fetch", "--quiet", "origin")
	return err
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

func parseVersions(out string) ([]plugin.Version, error) {
	var versions []plugin.Version
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("malformed log line: %q", line)
		}
		unix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing commit time %q: %w", parts[1], err)
		}
		versions = append(versions, plugin.Version{
			SHA:     parts[0],
			Subject: parts[2],
			Date:    time.Unix(unix, 0),
		})
	}
	return versions, nil
}
