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

  Scenario: Enter applies the selected version to the plugin
    Given the sample model with a recording updater
    When I press "right"
    And I press "down"
    And I press "enter"
    And I process pending commands
    Then the updater applied "telescope.nvim" at "telA"
    And the status contains "telescope.nvim"

  Scenario: A successful update refreshes the plugin's versions in place
    Given the sample model with a recording updater
    When I press "right"
    And I press "down"
    And I press "enter"
    And I process pending commands
    Then the selected plugin is "telescope.nvim"
    And the visible version shas are "telB"

  Scenario: A plugin with no more versions drops off the list after updating
    Given the sample model with a recording updater
    When I press "down"
    And I press "right"
    And I press "enter"
    And I process pending commands
    Then the selected plugin is "telescope.nvim"
    And there is 1 plugin

  Scenario: A failed update surfaces its error in the status
    Given the sample model with a failing updater
    When I press "right"
    And I press "enter"
    And I process pending commands
    Then the status contains "boom"

  Scenario: Esc dismisses the status without quitting
    Given the sample model with a failing updater
    When I press "right"
    And I press "enter"
    And I process pending commands
    Then the status contains "boom"
    When I press "esc"
    Then the status is empty
    And the selected plugin is "telescope.nvim"

  Scenario: The versions pane reports how many releases are too new to install
    Given a model whose only plugin has 3 versions all too new
    And a window size of 120 by 40
    Then the versions pane shows "3 releases too new to install"

  Scenario: Changing plugin resets version selection
    Given the sample model
    When I press "right"
    And I press "down"
    And I press "left"
    And I press "down"
    And I press "up"
    And I press "right"
    Then the selected version sha is "telB"
