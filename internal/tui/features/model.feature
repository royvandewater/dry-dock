Feature: Navigating plugins, versions, and changes
  The model tracks which plugin, version, and pane are active and derives the
  installable versions and cumulative changes from that state.

  Scenario: Down arrow moves plugin selection
    Given the sample model
    Then the selected plugin is "telescope.nvim"
    When I press "down"
    Then the selected plugin is "blink.cmp"

  Scenario: Plugin selection clamps at the ends
    Given the sample model
    When I press "up"
    Then the selected plugin is "telescope.nvim"
    When I press "down" 5 times
    Then the selected plugin is "blink.cmp"

  Scenario: Version list hides versions younger than the minimum age
    Given the sample model
    Then there are 2 visible versions
    And the visible version shas are "telB, telA"

  Scenario: Right arrow focuses versions and selects the top
    Given the sample model
    When I press "right"
    Then a version is selected
    And the selected version sha is "telB"

  Scenario: Down arrow in version focus moves version selection
    Given the sample model
    When I press "right"
    And I press "down"
    Then the selected version sha is "telA"

  Scenario: Left arrow returns focus to plugins
    Given the sample model
    When I press "right"
    And I press "left"
    And I press "down"
    Then the selected plugin is "blink.cmp"

  Scenario: Selected changes are cumulative from current through selected
    Given the sample model
    When I press "right"
    Then the selected changes shas are "telB, telA"
    When I press "down"
    Then the selected changes shas are "telA"

  Scenario: Changing plugin resets version selection
    Given the sample model
    When I press "right"
    And I press "down"
    And I press "left"
    And I press "down"
    And I press "up"
    And I press "right"
    Then the selected version sha is "telB"
