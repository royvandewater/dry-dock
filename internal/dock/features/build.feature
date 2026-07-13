Feature: Building the plugin list
  Build turns locked plugins plus a version source into plugin.Plugin values,
  dropping any plugin that has no newer versions to offer.

  Scenario: Assembles plugins and skips those without candidates
    Given the install dir "/plugins"
    And a locked plugin "telescope.nvim" on branch "master" at commit "curTel"
    And a locked plugin "blink.cmp" on branch "main" at commit "curCmp"
    And the source reports current version "curTel" for "telescope.nvim"
    And the source reports current version "curCmp" for "blink.cmp"
    And the source offers candidates "newTel" for "telescope.nvim"
    And the source offers no candidates for "blink.cmp"
    When I build the plugin list
    Then there is 1 plugin
    And plugin 1 is named "telescope.nvim"
    And plugin 1 has current sha "curTel"
    And plugin 1 has candidate shas "newTel"

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
