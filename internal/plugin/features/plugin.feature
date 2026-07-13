Feature: Plugin changes and installability
  A plugin exposes the changes pulled in by updating to a chosen version and
  filters out candidate versions that are too young to install.

  Scenario: Changes accumulate from current through the selected version
    Given a plugin whose candidates newest-first are "v3, v2, v1"
    When I request the changes up to index 1
    Then the resulting shas are "v2, v1"

  Scenario: A Conventional Commits bang marks a breaking change
    Given a commit with subject "feat(keymap)!: replace expr keymaps"
    Then the commit is breaking

  Scenario: A bang on a scopeless type still marks a breaking change
    Given a commit with subject "refactor!: drop legacy config"
    Then the commit is breaking

  Scenario: An ordinary commit is not breaking
    Given a commit with subject "fix: os should be system_info"
    Then the commit is not breaking

  Scenario: Updating to a version includes any breaking change it pulls in
    Given a plugin with candidates:
      | sha | subject                | age_days |
      | e   | feat: newest           | 10       |
      | d   | fix: safe              | 20       |
      | c   | feat!: breaking change | 30       |
      | b   | fix: safe              | 40       |
      | a   | fix: oldest            | 50       |
    Then updating to "e" includes a breaking change
    And updating to "c" includes a breaking change
    And updating to "b" does not include a breaking change

  Scenario: Installable versions exclude those younger than the minimum age
    Given the current time is "2026-07-13"
    And a minimum release age of 14 days
    And a plugin with candidates:
      | sha | subject  | age_days |
      | aaa | recent   | 2        |
      | bbb | seasoned | 30       |
    When I compute the installable versions
    Then the resulting shas are "bbb"
