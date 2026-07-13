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

  Scenario: A version matcher keeps in-range tags and counts newer ones outside it
    Given tags "v2.0.0, v1.11.0, v1.10.2, v1.9.0"
    And the current tag is "v1.10.2"
    And the version constraint is "1.*"
    When I select the in-range versions
    Then the resulting shas are "v1.11.0"
    And there is 1 newer release outside the range

  Scenario: Duplicate tags on one commit collapse to a single version
    Given tag releases:
      | tag     | sha  |
      | v2.0.0  | c200 |
      | v2.0    | c200 |
      | v1.11.0 | c110 |
      | v1.11   | c110 |
      | v1.10.2 | c102 |
    And the current commit is "c102"
    And the version constraint is "1.*"
    When I select the in-range versions
    Then the resulting shas are "c110"
    And there is 1 newer release outside the range

  Scenario: A star matcher offers every newer tag and flags nothing as outside
    Given tags "v2.0.0, v1.11.0, v1.10.2"
    And the current tag is "v1.10.2"
    And the version constraint is "*"
    When I select the in-range versions
    Then the resulting shas are "v2.0.0, v1.11.0"
    And there are 0 newer releases outside the range

  Scenario: Installable versions exclude those younger than the minimum age
    Given the current time is "2026-07-13"
    And a minimum release age of 14 days
    And a plugin with candidates:
      | sha | subject  | age_days |
      | aaa | recent   | 2        |
      | bbb | seasoned | 30       |
    When I compute the installable versions
    Then the resulting shas are "bbb"

  Scenario: Too-new count reports the candidates younger than the minimum age
    Given the current time is "2026-07-13"
    And a minimum release age of 14 days
    And a plugin with candidates:
      | sha | subject | age_days |
      | aaa | fresh   | 1        |
      | bbb | recent  | 5        |
      | ccc | ripe    | 30       |
    When I count the versions too new to install
    Then 2 versions are too new
