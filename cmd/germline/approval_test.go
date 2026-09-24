package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justinstimatze/germline/pkg/corpus"
)

// TestAnAgentCannotLicenseItsOwnDifference is the cold-read bypass: declare a
// decline, project it into the boundary, and close a real difference against
// it, with nobody asked. The close is kept as a proposal, the replay waits on
// a human rather than failing, and only a commit of the decline lets the close
// through.
func TestAnAgentCannotLicenseItsOwnDifference(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := filepath.Join(t.TempDir(), "svc")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "t@example.com")
	git(t, dir, "config", "user.name", "t")
	git(t, dir, "config", "commit.gpgsign", "false")
	commit := func(msg string, paths ...string) {
		git(t, dir, append([]string{"add", "--"}, paths...)...)
		git(t, dir, "commit", "-qm", msg)
	}
	mustRun := func(args ...string) string {
		t.Helper()
		e, out, _ := newEnv(t, dir)
		if err := run(e, args); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out.String())
		}
		return out.String()
	}

	mustRun("init")
	commit("init", ".")

	declineIn(t, dir, "svc") // the agent writes the decline; nobody commits it
	input := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(input, []byte("alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRun("record", "-weight", "1", input)
	mustRun("boundary", "-write")
	commit("recorded", "corpus", "BOUNDARY.md")

	m, err := corpus.Load(filepath.Join(dir, "corpus", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	id := m.Inputs[0].ID
	closeArgs := []string{
		"close", "-input", id, "-from", "1", "-to", "2", "-as", "boundary",
		"-decision", "HeaderOrderIsNotPromised", "-entry", "Header order", "-reason", "incidental",
	}
	replayArgs := []string{"replay", "-was", "1=cat", "-now", "2=tr a-z A-Z"}

	e, out, _ := newEnv(t, dir)
	if err = run(e, closeArgs); !errors.Is(err, errAwaiting) {
		t.Fatalf("close against an uncommitted decline: %v, want awaiting\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Stop here and ask") {
		t.Errorf("a refused close did not say what to do:\n%s", out.String())
	}

	e, out, _ = newEnv(t, dir)
	if err = run(e, replayArgs); !errors.Is(err, errAwaiting) {
		t.Fatalf("replay with only a proposed close open: %v, want awaiting\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"ALPHA\n"`) {
		t.Errorf("an abstracted input's outputs were not shown beside its difference:\n%s", out.String())
	}

	commit("a human approves the decline", "decisions")
	closed := mustRun(closeArgs...)
	if !strings.Contains(closed, "approved: committed ") {
		t.Errorf("the close did not record what approved it:\n%s", closed)
	}
	m, err = corpus.Load(filepath.Join(dir, "corpus", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.ProposedCloses) != 0 || len(m.AcceptedDiffs) != 1 {
		t.Errorf("after approval: %d proposed, %d accepted; want 0 and 1",
			len(m.ProposedCloses), len(m.AcceptedDiffs))
	}
	mustRun(replayArgs...)
}
