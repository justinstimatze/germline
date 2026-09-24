package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestAppendOnlyHoldsForAProjectInASubdirectory is the layout a pilot uses: the
// project is a directory inside someone else's repository, and the repository
// root carries a manifest of its own. git resolves `HEAD:path` from the
// repository root, so the check used to compare the project's manifest against
// the root's, find nothing to object to, and pass with an input deleted.
func TestAppendOnlyHoldsForAProjectInASubdirectory(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	sub := filepath.Join(repo, "sub")
	if err := os.Mkdir(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{repo, sub} {
		if e, out, _ := newEnv(t, dir); run(e, []string{"init"}) != nil {
			t.Fatalf("init %s:\n%s", dir, out.String())
		}
	}
	input := filepath.Join(sub, "input.txt")
	if err := os.WriteFile(input, []byte("alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if e, out, _ := newEnv(t, sub); run(e, []string{"record", "-weight", "1", input}) != nil {
		t.Fatalf("record:\n%s", out.String())
	}
	git(t, repo, "add", "-A")
	git(t, repo, "-c", "user.email=t@example.com", "-c", "user.name=t", "commit", "-qm", "recorded")

	if e, out, _ := newEnv(t, sub); run(e, []string{"check"}) != nil {
		t.Fatalf("check before any edit:\n%s", out.String())
	}

	manifest := filepath.Join(sub, "corpus", "manifest.json")
	body, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if jsonErr := json.Unmarshal(body, &m); jsonErr != nil {
		t.Fatal(jsonErr)
	}
	m["inputs"] = []any{}
	body, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(manifest, body, 0o600); err != nil {
		t.Fatal(err)
	}

	e, out, _ := newEnv(t, sub)
	if err = run(e, []string{"check"}); !errors.Is(err, errFailed) {
		t.Fatalf("check passed with a committed input deleted from a subdirectory project: %v\n%s",
			err, out.String())
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
