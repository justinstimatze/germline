package law_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/justinstimatze/germline/pkg/law"
)

type config struct {
	Name    string
	Workers int
}

func TestLensFamilyHoldsForAWellBehavedLens(t *testing.T) {
	t.Parallel()
	s := &law.Set{}
	s.Register(law.Lens("Workers",
		func(c config) int { return c.Workers },
		func(c config, v int) config { c.Workers = v; return c },
	)...)
	if len(s.Laws()) != 3 {
		t.Fatalf("Lens gave %d laws, want 3", len(s.Laws()))
	}
	if ps := s.Check(nil); !ps.OK() {
		t.Errorf("the lens laws do not hold:\n%s", ps.String())
	}
}

// TestLensFamilyCatchesABadLens: a setter that clamps breaks PutGet, because
// what you get back is not what you put. A law family that could not catch
// that would be decoration.
func TestLensFamilyCatchesABadLens(t *testing.T) {
	t.Parallel()
	s := &law.Set{}
	s.Register(law.Lens("Clamped",
		func(c config) int { return c.Workers },
		func(c config, v int) config {
			if v > 8 {
				v = 8
			}
			c.Workers = v
			return c
		},
	)...)
	ps := s.Check(nil)
	if ps.OK() {
		t.Fatal("a clamping setter passed the lens laws")
	}
	if !strings.Contains(ps.String(), "ClampedPutGet") {
		t.Errorf("the violation does not name PutGet:\n%s", ps.String())
	}
}

func TestRoundtripAndIdempotentAndInvolution(t *testing.T) {
	t.Parallel()
	s := &law.Set{}
	s.Register(
		law.Roundtrip("IntFormat", strconv.Itoa, strconv.Atoi),
		law.Idempotent("TrimSpace", strings.TrimSpace),
		law.Involution("NegateInt", func(i int) int { return -i }),
		law.Deterministic("StringLen", func(s string) int { return len(s) }),
	)
	if ps := s.Check(nil); !ps.OK() {
		t.Errorf("a standard family does not hold:\n%s", ps.String())
	}
}

func TestRoundtripCatchesALossyFormat(t *testing.T) {
	t.Parallel()
	s := &law.Set{}
	s.Register(law.Roundtrip("LossyFormat",
		func(i int) string { return strconv.Itoa(i % 10) },
		strconv.Atoi,
	))
	if ps := s.Check(nil); ps.OK() {
		t.Error("a format that drops all but the last digit passed the roundtrip law")
	}
}

func TestMetamorphicRelation(t *testing.T) {
	t.Parallel()
	s := &law.Set{}
	// No oracle for "how long is this string" — but appending must not shorten it.
	s.Register(law.Metamorphic("LengthGrowsWithAppend",
		"appending a character never shortens the result",
		func(in string) (string, bool) { return in + "x", true },
		func(in string) int { return len(in) },
		func(before, after int) bool { return after > before },
	))
	if ps := s.Check(nil); !ps.OK() {
		t.Errorf("the metamorphic relation does not hold:\n%s", ps.String())
	}
}

func TestALawWithoutAStatementIsRefused(t *testing.T) {
	t.Parallel()
	s := &law.Set{}
	s.Register(law.Law{Name: "Nameless", Prop: func(int) bool { return true }})
	if ps := s.Check(nil); !strings.Contains(ps.String(), "no statement") {
		t.Errorf("a law with no statement was accepted:\n%s", ps.String())
	}
}

func TestDuplicateLawNamesAreRefused(t *testing.T) {
	t.Parallel()
	s := &law.Set{}
	l := law.Idempotent("Same", strings.TrimSpace)
	s.Register(l, l)
	if ps := s.Check(nil); !strings.Contains(ps.String(), "share this name") {
		t.Errorf("two laws with one name were accepted:\n%s", ps.String())
	}
}

func TestUncheckedFindsProseLaws(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := "package x\n\n// " + law.Marker + " replaying n atoms costs O(n) parse steps.\nvar _ = 1\n"
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.bin"), []byte{0, 1, 2}, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := law.Unchecked(dir)
	if err != nil {
		t.Fatalf("Unchecked: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d prose laws, want 1: %+v", len(got), got)
	}
	if got[0].Line != 3 {
		t.Errorf("Line = %d, want 3", got[0].Line)
	}
	if got[0].Statement != "replaying n atoms costs O(n) parse steps." {
		t.Errorf("Statement = %q", got[0].Statement)
	}
}

// TestUncheckedReadsPastAnOverlongLine: a minified or generated file in the
// tree used to stop the scan with "token too long", and the laws metric then
// read zero for the whole project. A law after the long line, and one in the
// next file, are both still found, with the line numbers they really have.
func TestUncheckedReadsPastAnOverlongLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	long := strings.Repeat("x", 2<<20)
	src := long + "\n// " + law.Marker + " after the long line.\n"
	if err := os.WriteFile(filepath.Join(dir, "a.js"), []byte(src), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	other := "// " + law.Marker + " in the next file.\n"
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte(other), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := law.Unchecked(dir)
	if err != nil {
		t.Fatalf("Unchecked: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d prose laws, want 2: %+v", len(got), got)
	}
	if got[0].Line != 2 {
		t.Errorf("Line = %d, want 2", got[0].Line)
	}
}

// TestUncheckedIgnoresProseAboutTheMarker: documentation that mentions the
// marker mid-sentence, and the constant declaring it, are not laws. Before
// this anchor the scanner found four laws in its own documentation.
func TestUncheckedIgnoresProseAboutTheMarker(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lines := []string{
		"A law with no executable form is written `" + law.Marker + "` so it can be found.",
		`const Marker = "` + law.Marker + `"`,
		"return fmt.Errorf(\"write it as a " + law.Marker + " comment instead\")",
		"",
		"<!-- " + law.Marker + " the boundary never shrinks without a decision -->",
		"# " + law.Marker + " retirement is append-only",
	}
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "doc.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := law.Unchecked(dir)
	if err != nil {
		t.Fatalf("Unchecked: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d prose laws, want 2 (only the two comment lines): %+v", len(got), got)
	}
	if got[0].Statement != "the boundary never shrinks without a decision -->" {
		t.Errorf("Statement = %q", got[0].Statement)
	}
	if got[1].Statement != "retirement is append-only" {
		t.Errorf("Statement = %q", got[1].Statement)
	}
}
