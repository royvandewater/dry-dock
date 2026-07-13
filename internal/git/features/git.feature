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
