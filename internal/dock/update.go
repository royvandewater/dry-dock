package dock

import (
	"path/filepath"

	"github.com/royvandewater/dry-dock/internal/git"
	"github.com/royvandewater/dry-dock/internal/lazy"
)

// Updater performs a plugin update against the files described by Config: it
// checks out the chosen commit in the plugin's clone and repins it in the lock
// file. It satisfies the tui.Applier interface.
type Updater struct {
	Config Config
}

// Apply moves pluginName to sha on disk and in the lock file.
func (u Updater) Apply(pluginName, sha string) error {
	if err := git.Checkout(filepath.Join(u.Config.InstallDir, pluginName), sha); err != nil {
		return err
	}
	return lazy.UpdateFile(u.Config.LockPath, pluginName, sha)
}
