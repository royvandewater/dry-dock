package dock

import (
	"fmt"
	"os/exec"

	"github.com/royvandewater/dry-dock/internal/lazy"
)

// Updater performs a plugin update against the files described by Config. It
// repins the plugin in lazy.vim's lock file and then asks lazy.vim to check the
// commit out, so lazy's own pipeline (dependency installs, build steps) runs
// rather than a raw git checkout that would leave the clone half-updated. It
// satisfies the tui.Applier interface.
type Updater struct {
	Config Config

	// Restore applies the lock file's pinned commit for one plugin through
	// lazy.vim. Defaults to running nvim headless when nil.
	Restore func(pluginName string) error
}

// Apply repins pluginName to sha in the lock file, then has lazy.vim restore it.
func (u Updater) Apply(pluginName, sha string) error {
	if err := lazy.UpdateFile(u.Config.LockPath, pluginName, sha); err != nil {
		return err
	}
	restore := u.Restore
	if restore == nil {
		restore = restoreViaNvim
	}
	return restore(pluginName)
}

// restoreViaNvim drives `:Lazy restore <plugin>` in a headless nvim, which reads
// the lock file we just wrote and moves the plugin to the pinned commit.
func restoreViaNvim(pluginName string) error {
	cmd := exec.Command("nvim", "--headless", "+Lazy! restore "+pluginName, "+qa")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("nvim Lazy restore %s: %w\n%s", pluginName, err, out)
	}
	return nil
}
