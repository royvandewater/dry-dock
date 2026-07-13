package lazy

import (
	"strings"
	"testing"
)

func TestParseLockReadsNameBranchAndCommit(t *testing.T) {
	raw := `{
	  "telescope.nvim": { "branch": "master", "commit": "3333a52ff548ba0a68af6d8da1e54f9cd96e9179" },
	  "blink.cmp": { "branch": "main", "commit": "78336bc89ee5365633bcf754d93df01678b5c08f" }
	}`

	locked, err := ParseLock(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(locked) != 2 {
		t.Fatalf("expected 2 locked plugins, got %d", len(locked))
	}

	byName := map[string]Locked{}
	for _, l := range locked {
		byName[l.Name] = l
	}

	tel, ok := byName["telescope.nvim"]
	if !ok {
		t.Fatal("telescope.nvim missing")
	}
	if tel.Branch != "master" {
		t.Fatalf("expected branch master, got %q", tel.Branch)
	}
	if tel.Commit != "3333a52ff548ba0a68af6d8da1e54f9cd96e9179" {
		t.Fatalf("unexpected commit %q", tel.Commit)
	}
}
