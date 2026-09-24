package boundary

import (
	"errors"
	"fmt"
	"sort"
)

// A boundary entry makes a formal claim: the suite cannot tell an
// implementation that has this behavior from one that does not. That claim
// is testable, and until it is tested it is a place someone licensed
// themselves to be wrong. A Witness is what makes it testable.

// A Witness perturbs an implementation so that it differs on exactly one
// entry's observable and on nothing else. Perturb applies the difference and
// returns the function that undoes it, so a suite can be run on both sides.
//
// Writing one is the work. A witness that changes two observables at once
// proves nothing about either, and the check cannot detect that; it is the
// author's claim, the same way the entry's reason is.
type Witness struct {
	// Entry is the observable, matching an entry in the document exactly.
	Entry string
	// Perturb applies the difference and returns its undo.
	Perturb func() (restore func())
}

// Kind says what a finding is, so a caller can decide which ones stop a
// build. Only Distinguished is a lie in the document; the other two are
// places the document and the witnesses have drifted apart.
type Kind int

const (
	// Distinguished means the suite caught the perturbation. The observable
	// is pinned down by some law or replay, so it is a promise whether or
	// not anyone meant it to be, and the entry claiming otherwise is wrong.
	Distinguished Kind = iota
	// Unwitnessed means no witness perturbs this entry, so its claim has
	// never been checked. Not a lie; an unpaid bill.
	Unwitnessed
	// Unclaimed means a witness names an observable no entry declines.
	// Either the entry was dropped or the witness perturbs a real promise.
	Unclaimed
)

// String names the kind for a reader at a terminal.
func (k Kind) String() string {
	switch k {
	case Distinguished:
		return "distinguished"
	case Unwitnessed:
		return "unwitnessed"
	case Unclaimed:
		return "unclaimed"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
}

// A Finding is one entry or witness that needs attention, phrased for the
// person who has to act on it.
type Finding struct {
	Kind    Kind
	Entry   string
	Problem string
}

// Report is the result of checking a document against its witnesses.
type Report struct {
	// Confirmed lists the entries the suite genuinely cannot distinguish.
	// These are the boundary as claimed, and the only ones the drift gauge
	// should count.
	Confirmed []string
	Findings  []Finding
}

// OK reports whether every entry that was witnessed held up. Unwitnessed
// entries do not fail a build on their own; a caller that wants them to can
// read Findings.
func (r Report) OK() bool {
	for _, f := range r.Findings {
		if f.Kind == Distinguished {
			return false
		}
	}
	return true
}

// ErrSuiteAlreadyRed reports a suite that fails before any perturbation. Every
// verdict would be meaningless, so the check refuses to produce one.
var ErrSuiteAlreadyRed = errors.New("the suite fails unperturbed")

// Distinguish runs suite once as a baseline, then once per witness with that
// witness's perturbation applied, and reports which entries the suite can
// actually tell apart. A suite that returns nil under a perturbation confirms
// the entry: the difference is invisible to every law and every replay, which
// is what the entry says.
//
// suite must be independent of this package. It is the laws and the corpus
// replay, and it is never a judgment.
func Distinguish(d *Document, witnesses []Witness, suite func() error) (Report, error) {
	if err := suite(); err != nil {
		return Report{}, fmt.Errorf("%w: %w", ErrSuiteAlreadyRed, err)
	}

	byEntry := make(map[string]Witness, len(witnesses))
	var r Report
	for _, w := range witnesses {
		if _, ok := d.Find(w.Entry); !ok {
			r.Findings = append(r.Findings, Finding{
				Kind:  Unclaimed,
				Entry: w.Entry,
				Problem: "a witness perturbs this observable but no entry declines " +
					"it; either the entry was dropped or the witness changes a promise",
			})
			continue
		}
		byEntry[w.Entry] = w
	}

	for _, e := range d.Entries {
		w, ok := byEntry[e.Observable]
		if !ok {
			r.Findings = append(r.Findings, Finding{
				Kind:    Unwitnessed,
				Entry:   e.Observable,
				Problem: "no witness perturbs this observable, so the entry's claim is unchecked",
			})
			continue
		}
		if caught := runPerturbed(w, suite); caught != nil {
			r.Findings = append(r.Findings, Finding{
				Kind:  Distinguished,
				Entry: e.Observable,
				Problem: fmt.Sprintf("the suite distinguishes this observable, so it is a "+
					"promise: %v", caught),
			})
			continue
		}
		r.Confirmed = append(r.Confirmed, e.Observable)
	}
	sort.Strings(r.Confirmed)
	return r, nil
}

// runPerturbed applies a witness, runs the suite, and always restores.
func runPerturbed(w Witness, suite func() error) error {
	restore := w.Perturb()
	defer restore()
	return suite()
}
