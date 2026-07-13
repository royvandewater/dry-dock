Feature: Reading git history as plugin versions
  The git adapter turns a plugin clone's commit log into plugin.Version values.

  Background:
    Given a repo with commits on "master": "first", "second", "third"

  Scenario: LogBetween returns commits after a ref, newest first
    When I log between commit "first" and "master"
    Then there are 2 versions
    And the version subjects are "third, second"
    And version 1 has the sha of commit "third"
    And version 1 has a non-zero date

  Scenario: Commit reads the subject for a single sha
    When I read the commit for "second"
    Then the commit subject is "second"
    And the commit sha is the sha of commit "second"

  Scenario: Tags lists release tags with the commit they point at
    Given a tag "v1.0.0" on commit "first"
    And a tag "v1.1.0" on commit "third"
    When I list the tags
    Then there are 2 tags
    And tag "v1.0.0" points at commit "first"
    And tag "v1.1.0" points at commit "third"

  Scenario: CommitFile stages a file and records it as a commit
    Given a file "lazy-lock.json" holding "pinned"
    When I commit "lazy-lock.json" with message "Update telescope.nvim to newsha"
    Then the latest commit subject is "Update telescope.nvim to newsha"
    And there are no uncommitted changes
