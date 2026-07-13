Feature: Scrolling the changelog pane
  With a viewport set, the changes pane scrolls independently and clamps to the
  changelog's bounds, resetting whenever the selected version changes.

  Scenario: Right arrow from versions focuses the changes pane
    Given the sample model
    And a window size of 120 by 30
    When I press "right"
    And I press "right"
    Then the focus is on "changes"

  Scenario: Down in changes focus scrolls the changelog
    Given the long-changelog model
    And a window size of 120 by 30
    When I press "right"
    And I press "right"
    And I press "down"
    Then the changes scroll is 1

  Scenario: Changes scroll clamps to the bottom
    Given the long-changelog model
    And a window size of 120 by 30
    When I press "right"
    And I press "right"
    Then the max changes scroll is positive
    When I press "down" 1000 times
    Then the changes scroll is at the maximum

  Scenario: A short changelog does not scroll
    Given the sample model
    And a window size of 120 by 30
    When I press "right"
    And I press "right"
    And I press "down"
    Then the changes scroll is 0

  Scenario: Changing version resets the changes scroll
    Given the long-changelog model
    And a window size of 120 by 30
    When I press "right"
    And I press "right"
    And I press "down"
    And I press "down"
    Then the changes scroll is greater than 0
    When I press "left"
    And I press "down"
    Then the changes scroll is 0

  Scenario: Left arrow steps back through the panes
    Given the sample model
    And a window size of 120 by 30
    When I press "right"
    And I press "right"
    And I press "left"
    Then the focus is on "versions"
    When I press "left"
    Then the focus is on "plugins"
