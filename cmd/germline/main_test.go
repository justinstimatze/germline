package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newEnv(t *testing.T, dir string) (e *env, stdout, stderr *bytes.Buffer) {
	t.Helper()
	var out, errOut bytes.Buffer
	return &env{stdout: &out, stderr: &errOut, dir: dir, base: t.Context()}, &out, &errOut
}

func TestVersionPrintsSomething(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"version"}, {"--version"}} {
		e, out, _ := newEnv(t, t.TempDir())
		if err := run(e, args); err != nil {
			t.Fatalf("run(%v) error = %v", args, err)
		}
		if strings.TrimSpace(out.String()) == "" {
			t.Errorf("run(%v) printed nothing", args)
		}
	}
}

func TestUsageOnNoArgsAndUnknownCommand(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{}, {"bogus"}} {
		e, _, errOut := newEnv(t, t.TempDir())
		if err := run(e, args); err == nil {
			t.Errorf("run(%v) error = nil, want non-nil", args)
		}
		if !strings.Contains(errOut.String(), "usage:") {
			t.Errorf("run(%v) did not print usage to stderr", args)
		}
	}
}

func TestHelpIsNotAnError(t *testing.T) {
	t.Parallel()
	e, out, _ := newEnv(t, t.TempDir())
	if err := run(e, []string{"help"}); err != nil {
		t.Fatalf("run(help) error = %v", err)
	}
	for _, want := range []string{"check", "record", "replay", "close"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("usage does not mention %q", want)
		}
	}
}

// TestInitThenCheckIsGreen is the day-one path: a project adopts germline and
// the gate passes with nothing recorded and nothing declined. If adoption
// starts red, nobody adopts.
func TestInitThenCheckIsGreen(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	e, out, _ := newEnv(t, dir)
	if err := run(e, []string{"init"}); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, want := range []string{"BOUNDARY.md", "corpus/manifest.json", "decisions/decisions.go", "laws/README.md"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("init did not write %s:\n%s", want, out.String())
		}
	}
	e, out, _ = newEnv(t, dir)
	if err := run(e, []string{"check"}); err != nil {
		t.Fatalf("check after init: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "ok") {
		t.Errorf("check after init was not ok:\n%s", out.String())
	}
	if strings.Contains(out.String(), "no record configured") {
		t.Errorf("init left the project with no decision record:\n%s", out.String())
	}
}

// TestScaffoldedRecordCarriesADecline is the second day: the first decline goes
// into the record init wrote, and the boundary projects from it. The scaffold's
// types are a copy of a shape germline reads by structure, so this is what
// fails if the two drift apart.
func TestScaffoldedRecordCarriesADecline(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "myservice")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	declineIn(t, dir, "myservice")
	e, out, _ := newEnv(t, dir)
	if err := run(e, []string{"boundary", "-write"}); err != nil {
		t.Fatalf("boundary -write: %v\n%s", err, out.String())
	}
	e, out, _ = newEnv(t, dir)
	if err := run(e, []string{"check"}); err != nil {
		t.Fatalf("check: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "1 declined promise") {
		t.Errorf("the scaffolded record's decline did not reach the boundary:\n%s", out.String())
	}
}

// TestOwnRecordRefusesAForeignSystem is the silent version of the same day: a
// decline in the project's own record under a name other than the directory's
// used to be skipped, so the boundary published without it and check stayed
// green.
func TestOwnRecordRefusesAForeignSystem(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "myservice")
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	declineIn(t, dir, "otherservice")
	e, out, _ := newEnv(t, dir)
	if err := run(e, []string{"check"}); err == nil {
		t.Fatalf("check passed with a decline recorded under another system's name:\n%s", out.String())
	}
}

// declineIn runs init in dir and appends one decline for system to the record
// it wrote.
func declineIn(t *testing.T, dir, system string) {
	t.Helper()
	e, out, _ := newEnv(t, dir)
	if err := run(e, []string{"init"}); err != nil {
		t.Fatalf("init: %v\n%s", err, out.String())
	}
	record := filepath.Join(dir, "decisions", "decisions.go")
	f, err := os.OpenFile(record, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(`
var HeaderOrderIsNotPromised = Concept{&Entity{Name: "Header Order", Brief: "why"}}

var DeclinesHeaderOrder = Declines{
	Subject: HeaderOrderIsNotPromised,
	Object: &DeclinedPromise{
		System: "` + system + `", Observable: "Header order", Since: "0.1.0", Reason: "incidental",
	},
}
`)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}
}

// TestInitKeepsWhatIsAlreadyThere: adopting germline in a repository that
// already has a BOUNDARY.md must not lose it.
func TestInitKeepsWhatIsAlreadyThere(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const mine = "# Boundary\n\nMy own words.\n\n## Entries\n"
	if err := os.WriteFile(filepath.Join(dir, "BOUNDARY.md"), []byte(mine), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	e, out, _ := newEnv(t, dir)
	if err := run(e, []string{"init"}); err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(out.String(), "kept  BOUNDARY.md") {
		t.Errorf("init did not report keeping the existing file:\n%s", out.String())
	}
	got, err := os.ReadFile(filepath.Join(dir, "BOUNDARY.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != mine {
		t.Errorf("init overwrote a hand-written boundary:\n%s", got)
	}
}

// TestRecordThenReplayFindsTheDifference walks the loop the way a session
// does: record two inputs, replay two versions that disagree on one, and get
// a difference that nothing closes.
func TestRecordThenReplayFindsTheDifference(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	e, _, _ := newEnv(t, dir)
	if err := run(e, []string{"init"}); err != nil {
		t.Fatalf("init: %v", err)
	}
	same := filepath.Join(dir, "same.txt")
	if err := os.WriteFile(same, []byte("alpha\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	e, out, _ := newEnv(t, dir)
	if err := run(e, []string{"record", "-weight", "1", same}); err != nil {
		t.Fatalf("record: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "in-000001") {
		t.Errorf("record did not name the input:\n%s", out.String())
	}

	e, out, _ = newEnv(t, dir)
	err := run(e, []string{"replay", "-was", "0.1.0=/bin/cat", "-now", "0.2.0=/bin/echo"})
	if !errors.Is(err, errFailed) {
		t.Fatalf("replay error = %v, want errFailed\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "1 difference with nothing closing them") {
		t.Errorf("replay did not report one open difference:\n%s", out.String())
	}
}

func TestCloseRefusesAFourthWay(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	e, _, _ := newEnv(t, dir)
	if err := run(e, []string{"init"}); err != nil {
		t.Fatalf("init: %v", err)
	}
	e, _, _ = newEnv(t, dir)
	err := run(e, []string{
		"close", "-input", "in-000001", "-from", "0.1.0", "-to", "0.2.0",
		"-as", "edited the corpus", "-decision", "X", "-reason", "easier",
	})
	if err == nil || !strings.Contains(err.Error(), "the only three closes") {
		t.Errorf("err = %v, want a refusal naming the three closes", err)
	}
}
