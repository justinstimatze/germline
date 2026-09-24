package replay

import (
	"errors"
	"fmt"
	"slices"

	"github.com/justinstimatze/germline/pkg/boundary"
	"github.com/justinstimatze/germline/pkg/corpus"
	"github.com/justinstimatze/germline/pkg/decide"
	"github.com/justinstimatze/germline/pkg/law"
)

// There are three ways to close a difference and there is no fourth. The
// implementation was wrong, the law was missing a condition, or the behavior
// was never a promise. Editing the corpus is not on the list, and the reason
// this file is a type rather than a paragraph is that a paragraph cannot
// refuse.
//
// Each close demands the evidence its own kind needs, checked here rather
// than trusted: the decision exists in the record, the law is one that is
// registered, the boundary entry is published. A close that cannot produce
// its evidence is refused, and a refused close leaves the difference open,
// which is the correct state for a difference nobody can justify.

// A Ledger closes differences against the artifacts that must back them.
type Ledger struct {
	// Boundary is the published boundary. A boundary close must cite an
	// entry that is already in it: publishing comes before licensing, or the
	// entry is written to excuse the diff that prompted it.
	Boundary *boundary.Document
	// LawNames is every law that exists. Use LawNamesOf for a Go law set, or
	// list them from a config when the laws are in another language.
	LawNames []string
	// Store is the decision record. Every close is a decision.
	Store decide.Store
	// Approve reports whether a human approved the boundary entry for an
	// observable, returning the evidence to record or an error saying what is
	// missing. Publishing is not approving: the file and the record are both
	// things an agent can write. Nil licenses every published entry, for a
	// caller that establishes approval some other way.
	Approve func(observable string) (evidence string, err error)
}

// LawNamesOf returns the names in a law set, for a Ledger's LawNames.
func LawNamesOf(s *law.Set) []string {
	laws := s.Laws()
	names := make([]string, 0, len(laws))
	for _, l := range laws {
		names = append(names, l.Name)
	}
	return names
}

// The five ways a close is refused. Each names what is missing, because the
// fix is always to produce the missing thing and never to weaken the check.
// ErrAwaitingHuman is the one whose missing thing is a person.
var (
	ErrAwaitingHuman       = errors.New("a human has not approved this boundary entry")
	ErrNoReason            = errors.New("a close with no reason licenses being wrong silently")
	ErrUnrecordedDecision  = errors.New("the decision record does not carry this decision")
	ErrUnknownLaw          = errors.New("no law by that name is registered")
	ErrUnpublishedBoundary = errors.New(
		"the boundary does not declare this observable; publish the entry first")
)

// AsImplementation closes a difference as a defect in the new version: the
// output was wrong and the code was fixed. Recording it says the difference
// was seen and understood, so a replay that shows it again is a regression
// and not a fresh discovery.
func (l *Ledger) AsImplementation(d Difference, decision, reason string) (corpus.AcceptedDiff, error) {
	return l.close(d, corpus.ClosedAsImplementation, decision, reason, nil)
}

// AsLaw closes a difference as a law that was missing a condition: the new
// output is right, the law now says so, and the next regeneration cannot lose
// the condition again. The named law must exist, because a close citing a law
// nobody wrote is the failure the three-way close was built to prevent.
func (l *Ledger) AsLaw(d Difference, decision, lawName, reason string) (corpus.AcceptedDiff, error) {
	return l.close(d, corpus.ClosedAsLaw, decision, reason, func() error {
		if !slices.Contains(l.LawNames, lawName) {
			return fmt.Errorf("%w: %s", ErrUnknownLaw, lawName)
		}
		return nil
	})
}

// AsBoundary closes a difference as licensed: the behavior was never a
// promise. The observable must already be an entry in the published boundary.
// Publishing after the fact is how a boundary becomes a dumping ground, and
// it is the one thing this method is here to make awkward.
//
// A published entry must also be approved, when an Approve is set. Refused for
// that reason alone, the close still comes back filled in beside
// ErrAwaitingHuman, so a caller can keep it as a proposal for the human.
func (l *Ledger) AsBoundary(d Difference, decision, observable, reason string) (corpus.AcceptedDiff, error) {
	var evidence string
	ad, err := l.close(d, corpus.ClosedAsBoundary, decision, reason, func() error {
		if l.Boundary == nil {
			return fmt.Errorf("%w: no boundary document is loaded", ErrUnpublishedBoundary)
		}
		if _, ok := l.Boundary.Find(observable); !ok {
			return fmt.Errorf("%w: %q", ErrUnpublishedBoundary, observable)
		}
		if l.Approve == nil {
			return nil
		}
		var err error
		if evidence, err = l.Approve(observable); err != nil {
			return fmt.Errorf("%w: %q: %w", ErrAwaitingHuman, observable, err)
		}
		return nil
	})
	if err != nil && !errors.Is(err, ErrAwaitingHuman) {
		return corpus.AcceptedDiff{}, err
	}
	ad.Entry, ad.Approval = observable, evidence
	return ad, err
}

// close is the shared half: a reason, a decision that exists, and then
// whatever else the kind demands.
func (l *Ledger) close(d Difference, as corpus.Close, decision, reason string,
	extra func() error,
) (corpus.AcceptedDiff, error) {
	if reason == "" {
		return corpus.AcceptedDiff{}, ErrNoReason
	}
	if err := l.checkDecision(decision); err != nil {
		return corpus.AcceptedDiff{}, err
	}
	ad := corpus.AcceptedDiff{
		From:     d.From,
		To:       d.To,
		Input:    d.Input,
		ClosedAs: as,
		Decision: decision,
		Reason:   reason,
	}
	if extra != nil {
		if err := extra(); err != nil {
			return ad, err
		}
	}
	return ad, nil
}

func (l *Ledger) checkDecision(decision string) error {
	if decision == "" {
		return fmt.Errorf("%w: accepting a diff is a decision", ErrUnrecordedDecision)
	}
	if l.Store == nil {
		return fmt.Errorf("%w: no decision record is configured", ErrUnrecordedDecision)
	}
	missing, err := decide.Missing(l.Store, []string{decision})
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrUnrecordedDecision, decision)
	}
	return nil
}

// Accept appends a closed difference to the manifest. Closing the same
// difference twice is refused: the first close is the record, and a second
// one would let a later reader see two reasons and pick.
func Accept(m *corpus.Manifest, ad corpus.AcceptedDiff) error {
	for i := range m.AcceptedDiffs {
		existing := &m.AcceptedDiffs[i]
		if existing.Input == ad.Input && existing.From == ad.From && existing.To == ad.To {
			return fmt.Errorf("%s %s→%s is already closed as %q for %q; supersede that "+
				"decision rather than adding a second close",
				ad.Input, ad.From, ad.To, existing.ClosedAs, existing.Reason)
		}
	}
	m.AcceptedDiffs = append(m.AcceptedDiffs, ad)
	m.ProposedCloses = slices.DeleteFunc(m.ProposedCloses, func(p corpus.AcceptedDiff) bool {
		return sameDifference(p, ad)
	})
	return nil
}

// Propose keeps a boundary close that is waiting on a human, replacing any
// earlier proposal for the same difference. It is how stopping to ask leaves
// something behind: the human sees exactly what the close would license, and
// a replay can tell a difference nobody has looked at from one that is only
// waiting for approval.
func Propose(m *corpus.Manifest, ad corpus.AcceptedDiff) {
	m.ProposedCloses = slices.DeleteFunc(m.ProposedCloses, func(p corpus.AcceptedDiff) bool {
		return sameDifference(p, ad)
	})
	m.ProposedCloses = append(m.ProposedCloses, ad)
}

// Awaiting splits open differences into those with a proposed close, which
// wait on a human, and those nobody has proposed anything for, which are
// still work.
func Awaiting(m *corpus.Manifest, open []Difference) (awaiting, unproposed []Difference) {
	proposed := make(map[[3]string]bool, len(m.ProposedCloses))
	for i := range m.ProposedCloses {
		p := &m.ProposedCloses[i]
		proposed[[3]string{p.Input, p.From, p.To}] = true
	}
	for _, d := range open {
		if proposed[[3]string{d.Input, d.From, d.To}] {
			awaiting = append(awaiting, d)
		} else {
			unproposed = append(unproposed, d)
		}
	}
	return awaiting, unproposed
}

func sameDifference(a, b corpus.AcceptedDiff) bool {
	return a.Input == b.Input && a.From == b.From && a.To == b.To
}

// Open returns the differences in a result that no accepted diff in the
// manifest discharges. A replay is finished when this is empty, and until it
// is, the session has work left.
func Open(m *corpus.Manifest, r Result) []Difference {
	closed := make(map[[3]string]bool, len(m.AcceptedDiffs))
	for i := range m.AcceptedDiffs {
		ad := &m.AcceptedDiffs[i]
		closed[[3]string{ad.Input, ad.From, ad.To}] = true
	}
	var open []Difference
	for _, d := range r.Differences {
		if !closed[[3]string{d.Input, d.From, d.To}] {
			open = append(open, d)
		}
	}
	return open
}
