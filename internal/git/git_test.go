package git

import (
	"os/exec"
	"testing"
)

// makeRepo builds a throwaway git repo with three sequential commits on the
// default branch and returns the repo dir plus the SHAs in order (A, B, C).
func makeRepo(t *testing.T) (dir string, shas [3]string) {
	t.Helper()
	dir = t.TempDir()

	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(out)
	}

	run("init", "-q", "-b", "master")
	subjects := []string{"first", "second", "third"}
	for i, s := range subjects {
		run("commit", "-q", "--allow-empty",
			"--date", "2020-01-0"+string(rune('1'+i))+"T00:00:00",
			"-m", s)
		shas[i] = run("rev-parse", "HEAD")[:40]
	}
	return dir, shas
}

func TestLogBetweenReturnsCommitsAfterFromNewestFirst(t *testing.T) {
	dir, shas := makeRepo(t)

	got, err := LogBetween(dir, shas[0], "master")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 commits after first, got %d", len(got))
	}
	if got[0].Subject != "third" || got[1].Subject != "second" {
		t.Fatalf("expected [third, second], got [%s, %s]", got[0].Subject, got[1].Subject)
	}
	if got[0].SHA != shas[2] {
		t.Fatalf("expected newest SHA %q, got %q", shas[2], got[0].SHA)
	}
	if got[0].Date.IsZero() {
		t.Fatal("expected a commit date, got zero")
	}
}

func TestCommitReadsSubjectForSHA(t *testing.T) {
	dir, shas := makeRepo(t)

	got, err := Commit(dir, shas[1])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Subject != "second" {
		t.Fatalf("expected subject %q, got %q", "second", got.Subject)
	}
	if got.SHA != shas[1] {
		t.Fatalf("expected SHA %q, got %q", shas[1], got.SHA)
	}
}
