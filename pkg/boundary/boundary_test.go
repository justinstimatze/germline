package boundary_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/justinstimatze/germline/pkg/boundary"
)

// TestFirstEntryLandsUnderTheHeading covers the file `germline init` writes:
// an empty entry list followed by more prose under its own heading. The first
// projected entry used to land at the end of the file, under that prose.
func TestFirstEntryLandsUnderTheHeading(t *testing.T) {
	t.Parallel()
	const empty = "# Boundary\n\n" + boundary.Heading + "\n\n## How to read an entry\n\nProse.\n"
	d, err := boundary.Parse(strings.NewReader(empty))
	if err != nil {
		t.Fatal(err)
	}
	if got := d.String(); got != empty {
		t.Fatalf("an empty list does not round-trip:\n%s", got)
	}
	d, ps := d.Project([]boundary.Entry{{
		Observable: "key order", Since: "0.1.0", Decision: "KeyOrder", Reason: "Maps.",
	}})
	if !ps.OK() {
		t.Fatal(ps)
	}
	want := "# Boundary\n\n" + boundary.Heading + "\n\n" +
		"- **key order** — 0.1.0, `KeyOrder`. Maps.\n\n## How to read an entry\n\nProse.\n"
	if got := d.String(); got != want {
		t.Fatalf("first entry landed in the wrong place:\n%s", got)
	}
	again, err := boundary.Parse(strings.NewReader(want))
	if err != nil {
		t.Fatal(err)
	}
	if got := again.String(); got != want {
		t.Fatalf("the projected file does not round-trip:\n%s", got)
	}
}

// TestThisRepoBoundaryRoundTrips renders the repository's own boundary file
// and compares it byte for byte. germline is the first system germline
// governs; if the projection cannot reproduce this file, it cannot be trusted
// to rewrite anyone else's.
func TestThisRepoBoundaryRoundTrips(t *testing.T) {
	t.Parallel()
	const path = "../../BOUNDARY.md"
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	d, err := boundary.Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	if len(d.Entries) == 0 {
		t.Fatal("parsed no entries; the repository has some")
	}
	if got := d.String(); got != string(want) {
		t.Errorf("round trip changed the file:\n--- got ---\n%s\n--- want ---\n%s",
			got, want)
	}
}

func TestParseFields(t *testing.T) {
	t.Parallel()
	const src = `# Boundary

Prose above.

## Entries

- **The exact wall-clock time a replay takes** — 0.3,
  ` + "`GermlineExample`" + `. Timing is the machine's, not the system's.

Prose below.
`
	d, err := boundary.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(d.Entries))
	}
	e := d.Entries[0]
	if e.Observable != "The exact wall-clock time a replay takes" {
		t.Errorf("Observable = %q", e.Observable)
	}
	if e.Since != "0.3" {
		t.Errorf("Since = %q", e.Since)
	}
	if e.Decision != "GermlineExample" {
		t.Errorf("Decision = %q", e.Decision)
	}
	if e.Reason != "Timing is the machine's, not the system's." {
		t.Errorf("Reason = %q", e.Reason)
	}
	if got := len(d.Epilogue); got == 0 {
		t.Error("epilogue was dropped; prose after the list must survive")
	}
}

func TestParseRejectsMalformedEntry(t *testing.T) {
	t.Parallel()
	const src = "## Entries\n\n- no observable, no decision, no reason\n"
	if _, err := boundary.Parse(strings.NewReader(src)); err == nil {
		t.Error("Parse accepted an entry with no decision and no reason")
	}
}

func TestParseNeedsHeading(t *testing.T) {
	t.Parallel()
	const src = "# Boundary\n\nNothing here.\n"
	_, err := boundary.Parse(strings.NewReader(src))
	if !errors.Is(err, boundary.ErrNoEntries) {
		t.Errorf("err = %v, want ErrNoEntries", err)
	}
}

func TestParseEmptyList(t *testing.T) {
	t.Parallel()
	const src = "# Boundary\n\n## Entries\n\nNone yet.\n"
	d, err := boundary.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Entries) != 0 {
		t.Errorf("got %d entries, want 0", len(d.Entries))
	}
	if d.String() != src {
		t.Errorf("empty list did not round trip:\n%q", d.String())
	}
}

func TestAppendRefusesDuplicate(t *testing.T) {
	t.Parallel()
	d := &boundary.Document{Preamble: []string{boundary.Heading, ""}}
	e := boundary.Entry{
		Observable: "Iteration order of the input map",
		Since:      "0.1",
		Decision:   "GermlineExample",
		Reason:     "Go randomizes it and nothing should lean on it.",
	}
	if err := d.Append(e); err != nil {
		t.Fatalf("first Append: %v", err)
	}
	if err := d.Append(e); err == nil {
		t.Error("second Append succeeded; a duplicate double-counts the gauge")
	}
	if len(d.Entries) != 1 {
		t.Errorf("got %d entries, want 1", len(d.Entries))
	}
}

// The laws cover the direction that matters, which is that a projection never
// comes back with fewer promises declined. These two cover the direction the
// feature exists for.

func TestProjectionTakesTheRecordsWordingAndOrder(t *testing.T) {
	t.Parallel()
	d := &boundary.Document{
		Preamble: []string{"# Boundary", "", boundary.Heading, ""},
		Entries: []boundary.Entry{
			{Observable: "First", Since: "0.1", Decision: "A", Reason: "As published."},
			{Observable: "Second", Since: "0.1", Decision: "B", Reason: "As published."},
		},
		Epilogue: []string{""},
	}
	projected, ps := d.Project([]boundary.Entry{
		{Observable: "Second", Since: "0.1", Decision: "B", Reason: "As recorded."},
		{Observable: "First", Since: "0.2", Decision: "A", Reason: "As published."},
		{Observable: "Third", Since: "0.3", Decision: "C", Reason: "Recorded and never published."},
	})
	if !ps.OK() {
		t.Fatalf("Project: %v", ps.Sorted())
	}
	want := []boundary.Entry{
		{Observable: "Second", Since: "0.1", Decision: "B", Reason: "As recorded."},
		{Observable: "First", Since: "0.2", Decision: "A", Reason: "As published."},
		{Observable: "Third", Since: "0.3", Decision: "C", Reason: "Recorded and never published."},
	}
	if !reflect.DeepEqual(projected.Entries, want) {
		t.Errorf("entries = %v, want %v", projected.Entries, want)
	}
	if len(d.Entries) != 2 {
		t.Errorf("the document being projected was modified: %v", d.Entries)
	}
}

func TestProjectionRefusesToDropAnUnrecordedEntry(t *testing.T) {
	t.Parallel()
	d := &boundary.Document{
		Preamble: []string{"# Boundary", "", boundary.Heading, ""},
		Entries: []boundary.Entry{
			{Observable: "Recorded", Since: "0.1", Decision: "A", Reason: "Has a memory."},
			{Observable: "Forgotten", Since: "0.1", Decision: "B", Reason: "Memory deleted."},
		},
		Epilogue: []string{""},
	}
	before := d.String()
	projected, ps := d.Project([]boundary.Entry{
		{Observable: "Recorded", Since: "0.1", Decision: "A", Reason: "Has a memory."},
	})
	if ps.OK() {
		t.Fatal("Project accepted a record missing an entry")
	}
	if len(ps) != 1 || ps[0].Where != "Forgotten" {
		t.Errorf("problems = %v, want one naming Forgotten", ps.Sorted())
	}
	if projected.String() != before {
		t.Error("the document changed despite the refusal")
	}
}
