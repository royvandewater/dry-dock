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

  Scenario: Setting a plugin's commit rewrites only that entry
    Given a lazy-lock.json document:
      """
      {
        "blink.cmp": { "branch": "main", "commit": "old" },
        "telescope.nvim": { "branch": "master", "commit": "keepme" }
      }
      """
    When I set "blink.cmp" commit to "newsha"
    Then the resulting document is:
      """
      {
        "blink.cmp": { "branch": "main", "commit": "newsha" },
        "telescope.nvim": { "branch": "master", "commit": "keepme" }
      }
      """

  Scenario: Reading a single plugin's pinned commit from a file
    Given a lock file containing:
      """
      {
        "blink.cmp": { "branch": "main", "commit": "abc123" }
      }
      """
    When I read the commit for "blink.cmp" from the file
    Then the commit read is "abc123"

  Scenario: Updating a plugin's commit on disk
    Given a lock file containing:
      """
      {
        "blink.cmp": { "branch": "main", "commit": "old" }
      }
      """
    When I update "blink.cmp" commit to "newsha" in the file
    And I parse the lock file
    Then the locked plugin "blink.cmp" has commit "newsha"
