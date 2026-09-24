package laws

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/justinstimatze/germline"
	"github.com/justinstimatze/germline/pkg/boundary"
)

// The boundary file is the one artifact a program rewrites and a person also
// hand-edits. Both laws below exist so that neither destroys the other's
// work: the projection settles after one pass, and no entry is lost in it.

// TestBoundaryProjectionIsIdempotent quantifies over generated documents:
// rendering a document, parsing it back and rendering again produces the same
// bytes. Stated on bytes rather than on the struct because the split between
// preamble and epilogue is internal, and an empty entry list legitimately
// moves the boundary between them.
func TestBoundaryProjectionIsIdempotent(t *testing.T) {
	t.Parallel()
	prop := func(g genDoc) bool {
		once := g.doc.String()
		reparsed, err := boundary.Parse(strings.NewReader(once))
		if err != nil {
			return false
		}
		return reparsed.String() == once
	}
	if err := quick.Check(prop, nil); err != nil {
		t.Errorf("BoundaryProjectionIsIdempotent violated: %v", err)
	}
}

// TestBoundaryEntriesSurviveProjection quantifies over generated documents:
// every entry comes back from a render-and-parse with all four fields intact.
// A projection that silently dropped an entry would shrink the drift gauge
// and read as progress.
func TestBoundaryEntriesSurviveProjection(t *testing.T) {
	t.Parallel()
	prop := func(g genDoc) bool {
		reparsed, err := boundary.Parse(strings.NewReader(g.doc.String()))
		if err != nil {
			return false
		}
		return reflect.DeepEqual(reparsed.Entries, g.doc.Entries)
	}
	if err := quick.Check(prop, nil); err != nil {
		t.Errorf("BoundaryEntriesSurviveProjection violated: %v", err)
	}
}

// The boundary is generated from the decision record where the record
// carries the entries, which puts a program between a person's declared
// promise and the file that publishes it. The two laws below are what make
// that safe: the projection can add and it can reorder, and it can never come
// back with fewer promises declined than it was given.

// TestProjectionNeverDropsADeclinedPromise quantifies over a document and a
// record that may have forgotten some of it: every observable the document
// declined is still declined afterwards. A boundary that shrinks is a promise
// nobody meant to make, and the drift gauge reads the loss as progress.
func TestProjectionNeverDropsADeclinedPromise(t *testing.T) {
	t.Parallel()
	prop := func(g genProjection) bool {
		projected, _ := g.doc.Project(g.recorded)
		have := make(map[string]bool, len(projected.Entries))
		for _, e := range projected.Entries {
			have[e.Observable] = true
		}
		for _, e := range g.doc.Entries {
			if !have[e.Observable] {
				return false
			}
		}
		return true
	}
	if err := quick.Check(prop, nil); err != nil {
		t.Errorf("ProjectionNeverDropsADeclinedPromise violated: %v", err)
	}
}

// TestProjectionRefusesByNaming quantifies over the same pairs: when the
// record does not carry every entry, the projection reports one problem per
// unrecorded entry, naming it, and returns the document byte-identical.
//
// Refusing is not enough on its own. The reader has to close the gap by hand,
// and a refusal that said only "the record is incomplete" would leave them
// diffing two lists to find out which one.
func TestProjectionRefusesByNaming(t *testing.T) {
	t.Parallel()
	prop := func(g genProjection) bool {
		recorded := make(map[string]bool, len(g.recorded))
		for _, e := range g.recorded {
			recorded[e.Observable] = true
		}
		var unrecorded []string
		for _, e := range g.doc.Entries {
			if !recorded[e.Observable] {
				unrecorded = append(unrecorded, e.Observable)
			}
		}
		before := g.doc.String()
		projected, ps := g.doc.Project(g.recorded)
		if len(unrecorded) == 0 {
			return len(ps) == 0
		}
		if len(ps) != len(unrecorded) || projected.String() != before {
			return false
		}
		for _, obs := range unrecorded {
			if !namedIn(ps, obs) {
				return false
			}
		}
		return true
	}
	if err := quick.Check(prop, nil); err != nil {
		t.Errorf("ProjectionRefusesByNaming violated: %v", err)
	}
}

// namedIn reports whether some problem is about this observable.
func namedIn(ps germline.Problems, observable string) bool {
	for _, p := range ps {
		if p.Where == observable {
			return true
		}
	}
	return false
}

// genProjection is a document paired with a record that may disagree with it:
// some entries dropped, some reworded, some the document has never seen, and
// the order shuffled. Every one of those is a thing a record does — a decline
// recorded and not yet published, a reason improved, an entry whose memory
// was never written — and the laws have to hold across all of them.
type genProjection struct {
	doc      *boundary.Document
	recorded []boundary.Entry
}

// Generate builds the pair. Bounds are literals rather than functions of
// size, for the reason genDoc gives: testing/quick passes a constant 50.
func (genProjection) Generate(rnd *rand.Rand, size int) reflect.Value {
	base, ok := genDoc{}.Generate(rnd, size).Interface().(genDoc)
	if !ok {
		panic("genDoc.Generate returned something other than a genDoc")
	}
	doc := base.doc
	recorded := make([]boundary.Entry, 0, len(doc.Entries)+2)
	for _, e := range doc.Entries {
		if rnd.Intn(4) == 0 {
			continue // the record has no memory for this one
		}
		if rnd.Intn(3) == 0 {
			e.Reason = words(rnd, 1+rnd.Intn(20)) // the reason was improved
		}
		recorded = append(recorded, e)
	}
	for range rnd.Intn(3) { // recorded and not yet published
		recorded = append(recorded, boundary.Entry{
			Observable: words(rnd, 1+rnd.Intn(8)),
			Since:      version(rnd),
			Decision:   identifier(rnd),
			Reason:     words(rnd, 1+rnd.Intn(20)),
		})
	}
	dedupe(recorded)
	rnd.Shuffle(len(recorded), func(i, j int) {
		recorded[i], recorded[j] = recorded[j], recorded[i]
	})
	return reflect.ValueOf(genProjection{doc: doc, recorded: recorded})
}

// genDoc wraps a document so testing/quick can generate one. The generator
// avoids the four delimiters the format reserves rather than escaping them:
// an observable containing "**", a version containing a comma, a decision
// containing a backtick, and leading or repeated spaces, which the wrapper
// normalizes away by design.
type genDoc struct{ doc *boundary.Document }

// Generate builds a document with zero to five entries. The bound is a
// literal and not a function of size: testing/quick passes a constant 50, so
// an expression like rnd.Intn(size%6+1) collapses to a near-constant and the
// laws quantify over almost nothing.
func (genDoc) Generate(rnd *rand.Rand, _ int) reflect.Value {
	n := rnd.Intn(6)
	entries := make([]boundary.Entry, 0, n)
	for range n {
		entries = append(entries, boundary.Entry{
			Observable: words(rnd, 1+rnd.Intn(8)),
			Since:      version(rnd),
			Decision:   identifier(rnd),
			Reason:     words(rnd, 1+rnd.Intn(20)),
		})
	}
	dedupe(entries)
	epilogue := make([]string, 0, 4)
	epilogue = append(epilogue, "")
	for range rnd.Intn(3) {
		epilogue = append(epilogue, words(rnd, 1+rnd.Intn(10)))
	}
	return reflect.ValueOf(genDoc{&boundary.Document{
		Preamble: []string{"# Boundary", "", boundary.Heading, ""},
		Entries:  entries,
		Epilogue: epilogue,
	}})
}

// dedupe makes observables unique in place. Two entries for one observable is
// a document the format refuses, not a case the laws quantify over.
func dedupe(entries []boundary.Entry) {
	seen := make(map[string]int, len(entries))
	for i := range entries {
		obs := entries[i].Observable
		seen[obs]++
		if seen[obs] > 1 {
			entries[i].Observable = obs + " " + string(rune('a'+seen[obs]))
		}
	}
}

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func words(rnd *rand.Rand, n int) string {
	var sb strings.Builder
	for i := range n {
		if i > 0 {
			sb.WriteByte(' ')
		}
		for range 1 + rnd.Intn(9) {
			sb.WriteByte(alphabet[rnd.Intn(len(alphabet))])
		}
	}
	return sb.String()
}

func version(rnd *rand.Rand) string {
	var sb strings.Builder
	for i := range 2 + rnd.Intn(2) {
		if i > 0 {
			sb.WriteByte('.')
		}
		sb.WriteByte(byte('0' + rnd.Intn(10)))
	}
	return sb.String()
}

func identifier(rnd *rand.Rand) string {
	var sb strings.Builder
	for range 1 + rnd.Intn(4) {
		sb.WriteByte(byte('A' + rnd.Intn(26)))
		for range 1 + rnd.Intn(6) {
			sb.WriteByte(alphabet[rnd.Intn(len(alphabet))])
		}
	}
	return sb.String()
}
