Feature: Shared pane borders
  Adjacent panes share a single border column instead of drawing two
  side by side, so the layout wastes less horizontal space.

  Scenario: adjacent panes share one border
    Given the sample model
    And a window size of 100 by 30
    Then the rendered view has no doubled pane borders
    And every body line spans the full window width

  Scenario: the focused pane owns its shared seams
    Given the sample model
    And a window size of 100 by 30
    When I press "right"
    Then the focus is on "versions"
    And the focused pane border is highlighted on all four sides
    And the rendered view has no doubled pane borders
    And every body line spans the full window width

  Scenario: the last pane highlights its left seam when focused
    Given the sample model
    And a window size of 100 by 30
    When I press "right" 2 times
    Then the focus is on "changes"
    And the focused pane border is highlighted on all four sides
    And the rendered view has no doubled pane borders
    And every body line spans the full window width
