package decide_test

import (
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/justinstimatze/germline/pkg/decide"
)

func TestWinzeReadsExportedMemoriesOnly(t *testing.T) {
	t.Parallel()
	w := &decide.Winze{Dir: filepath.Join("testdata", "store")}
	got, err := w.Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	want := []string{
		"GermlineCorpusInputsAppendOnlyOutputsPerVersion",
		"GermlineThreeWayClose",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

func TestWinzeCachesAcrossCalls(t *testing.T) {
	t.Parallel()
	w := &decide.Winze{Dir: filepath.Join("testdata", "store")}
	first, err := w.Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	second, err := w.Names()
	if err != nil {
		t.Fatalf("Names again: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("second read differs: %v vs %v", first, second)
	}
}

func TestWinzeOnAMissingDirectory(t *testing.T) {
	t.Parallel()
	w := &decide.Winze{Dir: filepath.Join(t.TempDir(), "absent")}
	if _, err := w.Names(); !errors.Is(err, decide.ErrNoStore) {
		t.Errorf("err = %v, want ErrNoStore", err)
	}
}

func TestMissingNamesTheUnrecordedCitations(t *testing.T) {
	t.Parallel()
	store := decide.Set{"GermlineThreeWayClose"}
	got, err := decide.Missing(store, []string{
		"GermlineThreeWayClose",
		"GermlineNeverWritten",
		"GermlineNeverWritten", // cited twice, reported once
		"",                     // an empty citation is a different problem
	})
	if err != nil {
		t.Fatalf("Missing: %v", err)
	}
	if want := []string{"GermlineNeverWritten"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Missing() = %v, want %v", got, want)
	}
}

func TestMissingIsEmptyWhenEverythingIsRecorded(t *testing.T) {
	t.Parallel()
	store := decide.Set{"A", "B"}
	got, err := decide.Missing(store, []string{"A", "B"})
	if err != nil {
		t.Fatalf("Missing: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Missing() = %v, want none", got)
	}
}

// The reader recognizes a decline by its shape rather than its type name,
// because the sample system's record is a different package from the real
// store and both have to be readable by the same code. These cover the shapes
// a hand-written record actually gets written in.
func TestDeclinesReadsEveryShapeAStoreWritesThemIn(t *testing.T) {
	t.Parallel()
	w := &decide.Winze{Dir: filepath.Join("testdata", "declining")}
	got, err := w.Declines()
	if err != nil {
		t.Fatalf("Declines: %v", err)
	}
	memory := filepath.Join("testdata", "declining", "memory.go")
	want := []decide.Declined{{
		System:     "querystring",
		Observable: "How `+` is decoded",
		Since:      "0.1.0",
		Reason: "Form submission reads it as a space and RFC 3986 reads it " +
			"literally; both ship.",
		Decision: "PlusDecodingIsNotPromised",
		File:     memory, Line: 47, EndLine: 56,
	}, {
		System:     "querystring",
		Observable: "The case of hex digits in a rendered escape",
		Since:      "0.1.0",
		Reason:     "RFC 3986 makes the two equivalent.",
		Decision:   "EscapeCaseIsNotPromised",
		File:       memory, Line: 60, EndLine: 68,
	}, {
		System:     "somethingelse",
		Observable: "How `+` is decoded",
		Since:      "2.0",
		Reason:     "A different system, and the same observable means something else in it.",
		Decision:   "PlusDecodingIsNotPromised",
		File:       memory, Line: 72, EndLine: 80,
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Declines() =\n%#v\nwant\n%#v", got, want)
	}
}

// Declaration order is the boundary's append order, so the reader must not
// sort. Names are sorted and declines are not, out of the same walk.
func TestDeclinesKeepsDeclarationOrderWhileNamesAreSorted(t *testing.T) {
	t.Parallel()
	w := &decide.Winze{Dir: filepath.Join("testdata", "declining")}
	declined, err := w.Declines()
	if err != nil {
		t.Fatalf("Declines: %v", err)
	}
	if declined[0].Observable != "How `+` is decoded" {
		t.Errorf("first decline is %q; declaration order was not kept",
			declined[0].Observable)
	}
	names, err := w.Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("Names() = %v, want sorted", names)
	}
	for _, unexported := range names {
		if unexported == "declinesDraft" {
			t.Error("an unexported draft was read as a memory")
		}
	}
}

// A decline with no system reaches no boundary file. Reporting it is the
// point: it is a promise withdrawn in the record and still published in the
// file, which is the one direction this must not fail quietly in.
func TestADeclineWithNoSystemIsAnError(t *testing.T) {
	t.Parallel()
	w := &decide.Winze{Dir: filepath.Join("testdata", "incomplete")}
	_, err := w.Declines()
	if err == nil {
		t.Fatal("Declines() on a record with no system: want an error")
	}
	for _, want := range []string{"DeclinesWithoutASystem", "no system"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %q, want it to name %q", err, want)
		}
	}
}

// A record that carries no declines is one nobody has moved a boundary into.
// It is not an error and it is not an empty boundary: it is the file staying
// the source.
func TestAStoreWithNoDeclinesReportsNone(t *testing.T) {
	t.Parallel()
	w := &decide.Winze{Dir: filepath.Join("testdata", "store")}
	got, err := w.Declines()
	if err != nil {
		t.Fatalf("Declines: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Declines() = %v, want none", got)
	}
}
