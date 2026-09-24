package approve_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/justinstimatze/germline/internal/approve"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func repo(t *testing.T) (dir, file string) {
	t.Helper()
	dir = t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "test")
	git(t, dir, "config", "commit.gpgsign", "false")
	file = filepath.Join(dir, "record.go")
	write(t, file, "package record\n\nvar A = 1\n")
	git(t, dir, "add", "record.go")
	git(t, dir, "commit", "-q", "-m", "first")
	return dir, file
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestUncommittedLinesAreNotApproved covers the cold-read bypass: a decline
// written into the record and cited in the same breath, with no commit
// between.
func TestUncommittedLinesAreNotApproved(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	dir, file := repo(t)
	if _, err := approve.Committed(ctx, file, 3, 3); err != nil {
		t.Fatalf("a committed line was refused: %v", err)
	}
	write(t, file, "package record\n\nvar A = 1\n\nvar B = 2\n")
	if _, err := approve.Committed(ctx, file, 5, 5); !errors.Is(err, approve.ErrNotApproved) {
		t.Fatalf("an uncommitted line was approved: %v", err)
	}
	git(t, dir, "add", "record.go")
	if _, err := approve.Committed(ctx, file, 5, 5); !errors.Is(err, approve.ErrNotApproved) {
		t.Fatalf("a staged line was approved: %v", err)
	}
	git(t, dir, "commit", "-q", "-m", "second")
	commits, err := approve.Committed(ctx, file, 3, 5)
	if err != nil {
		t.Fatalf("committed lines were refused: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("span shaped by two commits reported %d", len(commits))
	}
}

func TestOutsideGitIsNotApproved(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "record.go")
	write(t, file, "package record\n")
	if _, err := approve.Committed(t.Context(), file, 1, 1); !errors.Is(err, approve.ErrNotApproved) {
		t.Fatalf("a file outside git was approved: %v", err)
	}
}
