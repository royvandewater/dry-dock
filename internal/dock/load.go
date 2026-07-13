package dock

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/royvandewater/dry-dock/internal/git"
	"github.com/royvandewater/dry-dock/internal/lazy"
	"github.com/royvandewater/dry-dock/internal/plugin"
)

// Config points the loader at lazy.vim's files.
type Config struct {
	LockPath   string // path to lazy-lock.json
	InstallDir string // directory holding plugin clones
}

// DefaultConfig resolves the standard lazy.vim locations under $HOME, honoring
// $XDG_CONFIG_HOME and $XDG_DATA_HOME when set.
func DefaultConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}

	return Config{
		LockPath:   filepath.Join(configHome, "nvim", "lazy-lock.json"),
		InstallDir: filepath.Join(dataHome, "nvim", "lazy"),
	}, nil
}

// Load reads the lock file, fetches each plugin's remote so newer commits are
// visible, and assembles the updatable plugins.
func Load(cfg Config) ([]plugin.Plugin, error) {
	f, err := os.Open(cfg.LockPath)
	if err != nil {
		return nil, fmt.Errorf("opening lock file: %w", err)
	}
	defer f.Close()

	locked, err := lazy.ParseLock(f)
	if err != nil {
		return nil, fmt.Errorf("parsing lock file: %w", err)
	}

	fetchAll(cfg.InstallDir, locked)

	return Build(cfg.InstallDir, locked, GitSource{})
}

// fetchAll refreshes remote-tracking refs concurrently. Fetch failures (e.g. a
// plugin cloned from a now-unreachable remote) are ignored; the plugin simply
// shows whatever versions are already local.
func fetchAll(installDir string, locked []lazy.Locked) {
	const maxConcurrent = 8
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, l := range locked {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			_ = git.Fetch(filepath.Join(installDir, name))
		}(l.Name)
	}
	wg.Wait()
}
