// Package lazy reads lazy.vim's on-disk state: the lazy-lock.json manifest and
// the plugin clones it pins.
package lazy

import (
	"encoding/json"
	"io"
	"sort"
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
