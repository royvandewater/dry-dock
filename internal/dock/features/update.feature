Feature: Applying an update
  The updater repins the plugin in lazy.vim's lock file and then asks lazy.vim
  itself to check the commit out, so dependency installs and build steps run
  through lazy's own pipeline instead of a raw git checkout. If the update
  breaks nvim, it rolls back to the previous commit.

  Scenario: Apply repins the lock file and drives lazy.vim to restore
    Given a lock file pinning "telescope.nvim" to commit "oldsha"
    When I apply the update for "telescope.nvim" to commit "newsha"
    Then the lock file pins "telescope.nvim" to commit "newsha"
    And lazy.vim was asked to restore "telescope.nvim"

  Scenario: A broken update rolls back to the previous commit
    Given a lock file pinning "telescope.nvim" to commit "oldsha"
    And commit "newsha" breaks nvim
    When I apply the update for "telescope.nvim" to commit "newsha"
    Then the update fails
    And the lock file pins "telescope.nvim" to commit "oldsha"

  Scenario: A lock file write failure skips the restore
    Given no lock file exists
    When I apply the update for "telescope.nvim" to commit "newsha"
    Then the update fails
    And lazy.vim was not asked to restore

  Scenario: A successful update is committed to the config repo
    Given a lock file pinning "telescope.nvim" to commit "oldsha"
    When I apply the update for "telescope.nvim" to commit "newsha"
    Then the config repo is committed with message "Update telescope.nvim to newsha"

  Scenario: A broken update is not committed to the config repo
    Given a lock file pinning "telescope.nvim" to commit "oldsha"
    And commit "newsha" breaks nvim
    When I apply the update for "telescope.nvim" to commit "newsha"
    Then the update fails
    And the config repo is not committed
