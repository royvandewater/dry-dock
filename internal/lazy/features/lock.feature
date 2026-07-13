Feature: Parsing lazy-lock.json
  ParseLock decodes lazy.vim's lock manifest into a name-sorted slice of
  locked plugins.

  Scenario: Reading name, branch, and commit for each entry
    Given a lazy-lock.json document:
      """
      {
        "telescope.nvim": { "branch": "master", "commit": "3333a52ff548ba0a68af6d8da1e54f9cd96e9179" },
        "blink.cmp": { "branch": "main", "commit": "78336bc89ee5365633bcf754d93df01678b5c08f" }
      }
      """
    When I parse the lock document
    Then there are 2 locked plugins
    And the locked plugin "telescope.nvim" has branch "master"
    And the locked plugin "telescope.nvim" has commit "3333a52ff548ba0a68af6d8da1e54f9cd96e9179"
