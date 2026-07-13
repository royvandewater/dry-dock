Feature: Building the plugin list
  Build turns locked plugins plus a version source into plugin.Plugin values,
  keeping even those with no newer versions so the TUI can show them as up to
  date.

  Scenario: Assembles plugins, keeping those without candidates
    Given the install dir "/plugins"
    And a locked plugin "telescope.nvim" on branch "master" at commit "curTel"
    And a locked plugin "blink.cmp" on branch "main" at commit "curCmp"
    And the source reports current version "curTel" for "telescope.nvim"
    And the source reports current version "curCmp" for "blink.cmp"
    And the source offers candidates "newTel" for "telescope.nvim"
    And the source offers no candidates for "blink.cmp"
    When I build the plugin list
    Then there are 2 plugins
    And plugin 1 is named "telescope.nvim"
    And plugin 1 has current sha "curTel"
    And plugin 1 has candidate shas "newTel"
    And plugin 2 is named "blink.cmp"
    And plugin 2 has current sha "curCmp"
    And plugin 2 has no candidates

  Scenario: Keeps a versioned plugin with no newer tags
    Given the install dir "/plugins"
    And a locked plugin "blink.cmp" on branch "main" at commit "v1cur"
    And the source reports current version "v1cur" for "blink.cmp"
    And the source offers tags for "blink.cmp":
      | tag    | sha   |
      | v1.0.0 | v1cur |
    And "blink.cmp" has version constraint "1.*"
    When I build the plugin list
    Then there is 1 plugin
    And plugin 1 is named "blink.cmp"
    And plugin 1 has no candidates
    And plugin 1 has current tag "v1.0.0"

  Scenario: A versioned plugin lists only in-range tags and notes the rest
    Given the install dir "/plugins"
    And a locked plugin "blink.cmp" on branch "main" at commit "v1cur"
    And the source reports current version "v1cur" for "blink.cmp"
    And the source offers tags for "blink.cmp":
      | tag    | sha   |
      | v2.0.0 | c200  |
      | v1.1.0 | c110  |
      | v1.0.0 | v1cur |
    And "blink.cmp" has version constraint "1.*"
    When I build the plugin list
    Then there is 1 plugin
    And plugin 1 has candidate shas "c110"
    And plugin 1 has 1 release outside its constraint
