Feature: Applying an update
  The updater moves a plugin's clone to a chosen commit and repins it in the
  lock file, so selecting a version in the TUI actually performs the update.

  Scenario: Apply checks out the commit and repins the lock file
    Given a plugin clone "telescope.nvim" on branch "master" with commits "first, second, third"
    And a lock file pinning "telescope.nvim" to its "first" commit
    When I apply the update for "telescope.nvim" to its "third" commit
    Then the clone "telescope.nvim" HEAD is at its "third" commit
    And the lock file pins "telescope.nvim" to its "third" commit
