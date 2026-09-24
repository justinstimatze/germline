package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justinstimatze/germline/pkg/replay"
)

// TestReplayRefusesToPassOnNothing covers the two ways replay used to report
// "no open differences" having compared nothing: no manifest at all, and a
// manifest whose inputs the replay could not see. Both exited 0 with a subject
// that fails every input, so the Stop gate passed too.
func TestReplayRefusesToPassOnNothing(t *testing.T) {
	t.Parallel()
	replayArgs := []string{"replay", "-was", "1=/bin/cat", "-now", "2=false"}

	t.Run("no manifest", func(t *testing.T) {
		t.Parallel()
		e, out, _ := newEnv(t, t.TempDir())
		if err := run(e, replayArgs); err == nil {
			t.Fatalf("replay passed with no manifest:\n%s", out.String())
		}
	})

	t.Run("manifest the replay cannot read", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		if e, out, _ := newEnv(t, dir); run(e, []string{"init"}) != nil {
			t.Fatalf("init:\n%s", out.String())
		}
		input := filepath.Join(dir, "input.txt")
		if err := os.WriteFile(input, []byte("alpha\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if e, out, _ := newEnv(t, dir); run(e, []string{"record", "-weight", "1", input}) != nil {
			t.Fatalf("record:\n%s", out.String())
		}
		manifest := filepath.Join(dir, "corpus", "manifest.json")
		body, err := os.ReadFile(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), `"active"`) {
			t.Fatalf("recorded manifest has no active input to break:\n%s", body)
		}
		body = []byte(strings.ReplaceAll(string(body), `"active"`, `"Active"`))
		if err = os.WriteFile(manifest, body, 0o600); err != nil {
			t.Fatal(err)
		}
		e, out, _ := newEnv(t, dir)
		if err = run(e, replayArgs); err == nil {
			t.Fatalf("replay passed on a manifest whose inputs it could not see:\n%s", out.String())
		}
	})
}

// TestReplaySaysWhyEveryInputFailed covers a replay whose subject cannot run
// at all: it used to stop at "none replayed" before printing a single
// failure, so the one line that named the cause never reached the reader.
func TestReplaySaysWhyEveryInputFailed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if e, out, _ := newEnv(t, dir); run(e, []string{"init"}) != nil {
		t.Fatalf("init:\n%s", out.String())
	}
	input := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(input, []byte("alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if e, out, _ := newEnv(t, dir); run(e, []string{"record", "-weight", "1", input}) != nil {
		t.Fatalf("record:\n%s", out.String())
	}
	missing := filepath.Join(dir, "no-such-subject")
	e, out, _ := newEnv(t, dir)
	if err := run(e, []string{"replay", "-was", "1=/bin/cat", "-now", "2=" + missing}); err == nil {
		t.Fatalf("replay passed with a subject that cannot run:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "no-such-subject") {
		t.Fatalf("replay failed without naming the subject that could not run:\n%s", out.String())
	}
}

// TestSubjectPathIsTheCallersNotTheProjects covers a relative subject path:
// it resolves against the directory germline was run from, as -store does,
// so `-C examples/x -now '2=bin/rfc'` finds the binary the caller just built.
func TestSubjectPathIsTheCallersNotTheProjects(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for spec, want := range map[string]string{
		"1=bin/form -x":  filepath.Join(wd, "bin", "form"),
		"1=/bin/cat":     "/bin/cat",
		"1=python3 a.py": "python3",
	} {
		s, subErr := subject(spec, t.TempDir())
		if subErr != nil {
			t.Fatal(subErr)
		}
		c, ok := s.(*replay.Command)
		if !ok {
			t.Fatalf("subject(%q) is a %T, not a command", spec, s)
		}
		if got := c.Path; got != want {
			t.Errorf("subject(%q) runs %q, want %q", spec, got, want)
		}
	}
}
