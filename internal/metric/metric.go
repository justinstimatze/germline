// Package metric computes the numbers the method says to watch. Each one
// tells you something a green build does not, and each is gameable in a way
// worth naming, so the doc comment for every metric says how.
//
// Trend them. A single reading of any of these says almost nothing; the shape
// over quarters is the whole signal.
package metric

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/justinstimatze/germline/pkg/boundary"
	"github.com/justinstimatze/germline/pkg/corpus"
	"github.com/justinstimatze/germline/pkg/law"
)

// A Reading is one metric with its value and the one line that says how to
// read it.
type Reading struct {
	Name  string
	Value string
	Note  string
}

// Set is every metric computable from the artifacts on disk.
type Set struct {
	Boundary *boundary.Document
	Corpus   *corpus.Manifest
	// LawNames are the laws the project's laws directory declares, from
	// law.Declared. Names rather than values, because a checker running
	// outside a project cannot hold that project's laws: this field used to
	// be a []law.Law taken from the checker's own registry, which was empty
	// for every project including this one, and the reading had never
	// measured anything.
	LawNames []string
	Prose    []law.Prose
}

// Readings computes every metric, in the order the method lists them.
func (s Set) Readings() []Reading {
	return []Reading{
		s.lawCount(),
		s.uncheckedLaws(),
		s.boundarySize(),
		s.corpusCoverage(),
		s.closeMix(),
		s.boundaryReopenRate(),
	}
}

// lawCount is the size of the law set. Append-mostly by design: the metric
// that matters is places per law, and the cheapest way to improve any
// per-law metric is to delete a law, which is why removal is a decision.
func (s Set) lawCount() Reading {
	return Reading{
		Name:  "laws",
		Value: strconv.Itoa(len(s.LawNames)),
		Note:  "append-mostly; removing one is a decision, not an edit",
	}
}

// uncheckedLaws counts laws stated in prose and not yet executable. Not a
// failure. A rising count with a flat law count means the set is accumulating
// intentions.
func (s Set) uncheckedLaws() Reading {
	return Reading{
		Name:  "laws stated but unchecked",
		Value: strconv.Itoa(len(s.Prose)),
		Note:  "prose laws are legitimate; a rising count against a flat law count is not",
	}
}

// boundarySize is the drift gauge. Growth without pruning is the escape hatch
// becoming the program.
func (s Set) boundarySize() Reading {
	n := 0
	if s.Boundary != nil {
		n = len(s.Boundary.Entries)
	}
	return Reading{
		Name:  "boundary entries",
		Value: strconv.Itoa(n),
		Note:  "the drift gauge; growth without pruning is the escape hatch becoming the program",
	}
}

// corpusCoverage is the share of the operational profile the active inputs
// carry. Gameable by weighting what was recorded rather than what happens,
// which is why choosing what to record stays a human job.
func (s Set) corpusCoverage() Reading {
	active, total := 0, 0
	var weight float64
	if s.Corpus != nil {
		total = len(s.Corpus.Inputs)
		weight = s.Corpus.ActiveWeight()
		for _, in := range s.Corpus.Inputs {
			if in.Status == corpus.Active {
				active++
			}
		}
	}
	return Reading{
		Name:  "corpus coverage",
		Value: fmt.Sprintf("%.2f of the profile across %d active inputs (%d recorded)", weight, active, total),
		Note:  "gameable by weighting what was recorded rather than what happens",
	}
}

// closeMix is how the accepted diffs split across the three closes. All
// boundary means the corpus is decaying into the profile; all implementation
// means the laws are learning nothing.
func (s Set) closeMix() Reading {
	counts := map[corpus.Close]int{}
	if s.Corpus != nil {
		counts = s.Corpus.Closes()
	}
	parts := make([]string, 0, len(corpus.Closes()))
	for _, c := range corpus.Closes() {
		parts = append(parts, fmt.Sprintf("%s %d", c, counts[c]))
	}
	return Reading{
		Name:  "closes by kind",
		Value: strings.Join(parts, ", "),
		Note:  "all one column is a dumping ground, whichever column it is",
	}
}

// boundaryReopenRate is the share of boundary closes whose entry a later diff
// landed on again. The one calibration number for the fuzziest judgment in
// the loop: an entry many diffs keep hitting is evidence it should become a
// promise, and a human decides that.
func (s Set) boundaryReopenRate() Reading {
	if s.Corpus == nil {
		return Reading{Name: "boundary reopen rate", Value: "no corpus", Note: reopenNote}
	}
	perReason := map[string]int{}
	total := 0
	for i := range s.Corpus.AcceptedDiffs {
		d := &s.Corpus.AcceptedDiffs[i]
		if d.ClosedAs != corpus.ClosedAsBoundary {
			continue
		}
		total++
		perReason[d.Decision]++
	}
	if total == 0 {
		return Reading{Name: "boundary reopen rate", Value: "no boundary closes yet", Note: reopenNote}
	}
	repeats := 0
	hottest, hottestN := "", 0
	keys := make([]string, 0, len(perReason))
	for k := range perReason {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		n := perReason[k]
		if n > 1 {
			repeats += n - 1
		}
		if n > hottestN {
			hottest, hottestN = k, n
		}
	}
	value := fmt.Sprintf("%d of %d boundary closes reopened an entry", repeats, total)
	if hottestN > 1 {
		value += fmt.Sprintf("; %s took %d", hottest, hottestN)
	}
	return Reading{Name: "boundary reopen rate", Value: value, Note: reopenNote}
}

const reopenNote = "an entry many diffs keep hitting is evidence it should become a promise"
