Feature: Shared pane borders
  Adjacent panes share a single border column instead of drawing two
  side by side, so the layout wastes less horizontal space.

  Scenario: adjacent panes share one border
    Given the sample model
    And a window size of 100 by 30
    Then the rendered view has no doubled pane borders
    And every body line spans the full window width
