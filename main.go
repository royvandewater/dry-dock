// Command dry-dock is a TUI for managing lazy.vim plugin updates while
// enforcing a configurable minimum release age.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/royvandewater/dry-dock/internal/dock"
	"github.com/royvandewater/dry-dock/internal/tui"
)

func main() {
	minAgeDays := flag.Int("min-age-days", 14, "minimum release age in days; versions younger than this are not offered")
	lockPath := flag.String("lock", "", "path to lazy-lock.json (defaults to lazy.vim's location)")
	installDir := flag.String("install-dir", "", "directory holding plugin clones (defaults to lazy.vim's location)")
	flag.Parse()

	if err := run(*minAgeDays, *lockPath, *installDir); err != nil {
		fmt.Fprintln(os.Stderr, "dry-dock:", err)
		os.Exit(1)
	}
}

func run(minAgeDays int, lockPath, installDir string) error {
	cfg, err := dock.DefaultConfig()
	if err != nil {
		return err
	}
	if lockPath != "" {
		cfg.LockPath = lockPath
	}
	if installDir != "" {
		cfg.InstallDir = installDir
	}

	fmt.Fprintln(os.Stderr, "Fetching plugin updates…")
	plugins, err := dock.Load(cfg)
	if err != nil {
		return err
	}

	minAge := time.Duration(minAgeDays) * 24 * time.Hour
	model := tui.New(plugins, time.Now(), minAge).WithApplier(dock.Updater{Config: cfg})

	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}
