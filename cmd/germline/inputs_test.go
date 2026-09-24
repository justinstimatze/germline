package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justinstimatze/germline/pkg/corpus"
)

// TestCheckNoticesALostRecording covers an input whose data file is gone
// while the rest of the corpus is here: check used to pass, and the next
// replay failed on it. Recording the same bytes again restores it, and a
// corpus with no data on this machine at all is left alone.
func TestCheckNoticesALostRecording(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustRun := func(args ...string) {
		t.Helper()
		e, out, _ := newEnv(t, dir)
		if err := run(e, args); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out.String())
		}
	}
	mustRun("init")
	src := t.TempDir()
	bodies := []string{"alpha\n", "beta\n"}
	files := make([]string, 0, len(bodies))
	for _, body := range bodies {
		f := filepath.Join(src, body[:4])
		if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	mustRun(append([]string{"record", "-weight", "0.5"}, files...)...)
	mustRun("check")

	m, err := corpus.Load(filepath.Join(dir, "corpus", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	lost := filepath.Join(dir, "corpus", "inputs", m.Inputs[0].SHA256)
	if err = os.Remove(lost); err != nil {
		t.Fatal(err)
	}
	e, out, _ := newEnv(t, dir)
	if err = run(e, []string{"check"}); !errors.Is(err, errFailed) {
		t.Fatalf("check passed with an active input's data gone: %v\n%s", err, out.String())
	}

	mustRun("record", "-weight", "0.5", files[0])
	if _, err = os.Stat(lost); err != nil {
		t.Fatalf("recording the same bytes again did not restore the data: %v", err)
	}
	mustRun("check")

	for _, in := range m.Inputs {
		if err = os.Remove(filepath.Join(dir, "corpus", "inputs", in.SHA256)); err != nil {
			t.Fatal(err)
		}
	}
	mustRun("check")
}
