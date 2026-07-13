// Package lazy reads lazy.vim's on-disk state: the lazy-lock.json manifest and
// the plugin clones it pins.
package lazy

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Locked is a single plugin as pinned in lazy-lock.json.
type Locked struct {
	Name   string
	Branch string
	Commit string
}

// ParseLock decodes a lazy-lock.json document into a name-sorted slice of
// locked plugins.
func ParseLock(r io.Reader) ([]Locked, error) {
	var entries map[string]struct {
		Branch string `json:"branch"`
		Commit string `json:"commit"`
	}
	if err := json.NewDecoder(r).Decode(&entries); err != nil {
		return nil, err
	}

	locked := make([]Locked, 0, len(entries))
	for name, e := range entries {
		locked = append(locked, Locked{Name: name, Branch: e.Branch, Commit: e.Commit})
	}
	sort.Slice(locked, func(i, j int) bool { return locked[i].Name < locked[j].Name })
	return locked, nil
}

// SetCommit rewrites doc so that name's commit becomes commit, leaving every
// other entry and field untouched. The output matches lazy.vim's own format:
// name-sorted entries, each on one line with alphabetically-ordered fields, so
// dry-dock's edits produce the same diff lazy.vim would.
func SetCommit(doc, name, commit string) (string, error) {
	var entries map[string]map[string]string
	if err := json.Unmarshal([]byte(doc), &entries); err != nil {
		return "", err
	}

	entry, ok := entries[name]
	if !ok {
		return "", fmt.Errorf("plugin %q not found in lock file", name)
	}
	entry["commit"] = commit

	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("{\n")
	for i, n := range names {
		b.WriteString("  " + encodeString(n) + ": " + encodeEntry(entries[n]))
		if i < len(names)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("}")
	return b.String(), nil
}

// CommitFor reads the commit that name is pinned to in the lock file at path.
func CommitFor(path, name string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	locked, err := ParseLock(f)
	if err != nil {
		return "", err
	}
	for _, l := range locked {
		if l.Name == name {
			return l.Commit, nil
		}
	}
	return "", fmt.Errorf("plugin %q not found in lock file", name)
}

// UpdateFile rewrites the lock file at path so name's commit becomes commit,
// leaving every other entry untouched, and writes it back with a trailing
// newline the way lazy.vim does.
func UpdateFile(path, name, commit string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	updated, err := SetCommit(string(data), name, commit)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(updated+"\n"), 0o644)
}

// encodeEntry renders a single lock entry on one line: { "k": "v", ... } with
// alphabetically-ordered keys.
func encodeEntry(entry map[string]string) string {
	keys := make([]string, 0, len(entry))
	for k := range entry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = encodeString(k) + ": " + encodeString(entry[k])
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func encodeString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
