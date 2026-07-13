package dock

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/royvandewater/dry-dock/internal/lazy"
)

// Updater performs a plugin update against the files described by Config. It
// repins the plugin in lazy.vim's lock file and then asks lazy.vim to check the
// commit out, so lazy's own pipeline (dependency installs, build steps) runs
// rather than a raw git checkout that would leave the clone half-updated. If
// the new version breaks nvim, it rolls back to the previous commit. It
// satisfies the tui.Applier interface.
type Updater struct {
	Config Config

	// Restore applies the lock file's pinned commit for one plugin through
	// lazy.vim. Defaults to running nvim headless when nil.
	Restore func(pluginName string) error

	// HealthCheck returns a non-nil error when nvim fails to load the plugin
	// after a restore. Defaults to loading it in a headless nvim when nil.
	HealthCheck func(pluginName string) error
}

// Apply repins pluginName to sha and has lazy.vim restore it. If that leaves
// nvim broken, it rolls the plugin back to the commit it was on before.
func (u Updater) Apply(pluginName, sha string) error {
	previous, err := lazy.CommitFor(u.Config.LockPath, pluginName)
	if err != nil {
		return fmt.Errorf("reading current commit for %s: %w", pluginName, err)
	}

	if err := u.moveTo(pluginName, sha); err != nil {
		if rbErr := u.moveTo(pluginName, previous); rbErr != nil {
			return fmt.Errorf("update of %s to %s failed (%v); rollback to %s also failed: %w",
				pluginName, shortSHA(sha), err, shortSHA(previous), rbErr)
		}
		return fmt.Errorf("update of %s to %s broke nvim; rolled back to %s: %w",
			pluginName, shortSHA(sha), shortSHA(previous), err)
	}
	return nil
}

// moveTo repins name to sha, has lazy.vim restore it, and verifies nvim still
// loads it.
func (u Updater) moveTo(name, sha string) error {
	if err := lazy.UpdateFile(u.Config.LockPath, name, sha); err != nil {
		return err
	}
	if err := u.restore()(name); err != nil {
		return err
	}
	return u.healthCheck()(name)
}

func (u Updater) restore() func(string) error {
	if u.Restore != nil {
		return u.Restore
	}
	return restoreViaNvim
}

func (u Updater) healthCheck() func(string) error {
	if u.HealthCheck != nil {
		return u.HealthCheck
	}
	return healthViaNvim
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

// healthViaNvim loads the plugin in a headless nvim and reports an error when
// the output carries a Lua/Vim error signature. nvim exits 0 even when a plugin
// fails to load, so we inspect the output rather than the exit code.
func healthViaNvim(pluginName string) error {
	cmd := exec.Command("nvim", "--headless", "+Lazy! load "+pluginName, "+qa")
	out, _ := cmd.CombinedOutput()
	if looksBroken(string(out)) {
		return fmt.Errorf("nvim reported errors loading %s:\n%s", pluginName, strings.TrimSpace(string(out)))
	}
	return nil
}

// brokenMarkers are substrings nvim emits when a plugin fails to load.
var brokenMarkers = []string{"E5113", "stack traceback", "Failed to source", "Error executing", "Error detected"}

func looksBroken(out string) bool {
	for _, m := range brokenMarkers {
		if strings.Contains(out, m) {
			return true
		}
	}
	return false
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
