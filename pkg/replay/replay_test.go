package replay_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/justinstimatze/germline/pkg/boundary"
	"github.com/justinstimatze/germline/pkg/corpus"
	"github.com/justinstimatze/germline/pkg/decide"
	"github.com/justinstimatze/germline/pkg/replay"
)

// upper is a Subject standing in for one version of a system.
type upper struct {
	ver   string
	apply func([]byte) []byte
}

func (u upper) Version() string { return u.ver }

func (u upper) Run(_ context.Context, in []byte) ([]byte, error) { return u.apply(in), nil }

func storeWith(t *testing.T, payloads ...string) (store replay.Dir, hashes []string) {
	t.Helper()
	dir := t.TempDir()
	hashes = make([]string, 0, len(payloads))
	for _, p := range payloads {
		h := replay.Hash([]byte(p))
		if err := os.WriteFile(filepath.Join(dir, h), []byte(p), 0o600); err != nil {
			t.Fatalf("write input: %v", err)
		}
		hashes = append(hashes, h)
	}
	return replay.Dir(dir), hashes
}

func manifestOver(hashes []string) *corpus.Manifest {
	m := corpus.New("2026-09-02")
	for i, h := range hashes {
		m.Inputs = append(m.Inputs, corpus.Input{
			ID:         fmt.Sprintf("in-%06d", i+1),
			RecordedAt: "2026-09-02T00:00:00Z",
			SHA256:     h,
			Level:      corpus.LevelAbstraction,
			Weight:     0.5,
			Status:     corpus.Active,
		})
	}
	return m
}

func TestRunFindsTheDifferingInput(t *testing.T) {
	t.Parallel()
	store, hashes := storeWith(t, "alpha", "beta")
	m := manifestOver(hashes)
	was := upper{"0.1.0", func(b []byte) []byte { return b }}
	now := upper{"0.2.0", func(b []byte) []byte {
		if string(b) == "beta" {
			return []byte("BETA")
		}
		return b
	}}
	r, err := replay.Run(t.Context(), m, store, was, now)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.Ran != 2 || r.Same != 1 {
		t.Errorf("Ran = %d, Same = %d, want 2 and 1", r.Ran, r.Same)
	}
	if len(r.Differences) != 1 || r.Differences[0].Input != "in-000002" {
		t.Fatalf("Differences = %v, want just in-000002", r.Differences)
	}
	if r.OK() {
		t.Error("OK() is true with a difference outstanding")
	}
}

// TestAParallelReplayReportsWhatASequentialOneWould is the invariant that
// makes Workers safe to use: a replay is a comparison, and which worker
// happened to finish first is not part of it.
//
// The gate is what gives the test teeth. Input one is held until every other
// input has been through, so completion order is very nearly the reverse of
// input order, and an implementation that appends its results as they arrive
// cannot pass.
func TestAParallelReplayReportsWhatASequentialOneWould(t *testing.T) {
	t.Parallel()
	const n = 40
	payloads := make([]string, n)
	for i := range payloads {
		payloads[i] = fmt.Sprintf("input-%02d", i)
	}
	store, hashes := storeWith(t, payloads...)
	m := manifestOver(hashes)

	var mu sync.Mutex
	rest := 0
	released := make(chan struct{})
	was := upper{"0.1.0", func(b []byte) []byte { return b }}
	now := upper{"0.2.0", func(b []byte) []byte {
		if string(b) == payloads[0] {
			<-released
		} else {
			mu.Lock()
			rest++
			if rest == n-1 {
				close(released)
			}
			mu.Unlock()
		}
		if strings.HasSuffix(string(b), "7") {
			return []byte(strings.ToUpper(string(b)))
		}
		return b
	}}

	r, err := replay.Run(t.Context(), m, store, was, now, replay.Workers(8))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.Ran != n {
		t.Errorf("Ran = %d, want %d", r.Ran, n)
	}
	want := []string{"in-000008", "in-000018", "in-000028", "in-000038"}
	if len(r.Differences) != len(want) {
		t.Fatalf("%d differences, want %d", len(r.Differences), len(want))
	}
	for i, d := range r.Differences {
		if d.Input != want[i] {
			t.Errorf("difference %d is %s, want %s", i, d.Input, want[i])
		}
	}
	if r.Same != n-len(want) {
		t.Errorf("Same = %d, want %d", r.Same, n-len(want))
	}
	for i := range payloads {
		id := fmt.Sprintf("in-%06d", i+1)
		if _, ok := r.Outputs[id]; !ok {
			t.Errorf("%s has no recorded output", id)
		}
	}
}

// TestAParallelReplayRecordsEachSubjectsOwnFailure keeps the failure legible
// without letting it hide everything else: one input crashing a version must
// not stop the rest of the corpus from being replayed, and the crash has to
// come back naming the input and the version that raised it, not a bare Run
// error that would erase every difference found elsewhere in the same run.
func TestAParallelReplayRecordsEachSubjectsOwnFailure(t *testing.T) {
	t.Parallel()
	payloads := make([]string, 40)
	for i := range payloads {
		payloads[i] = fmt.Sprintf("input-%02d", i)
	}
	store, hashes := storeWith(t, payloads...)
	m := manifestOver(hashes)

	sentinel := errors.New("the subject is broken")
	was := upper{"0.1.0", func(b []byte) []byte { return b }}
	now := failing{"0.2.0", payloads[3], sentinel}
	r, err := replay.Run(t.Context(), m, store, was, now, replay.Workers(8))
	if err != nil {
		t.Fatalf("Run: %v, want nil — one crash must not fail the whole replay", err)
	}
	if len(r.Failures) != 1 {
		t.Fatalf("Failures = %v, want exactly one", r.Failures)
	}
	f := r.Failures[0]
	if f.Input != "in-000004" || f.Version != "0.2.0" || !errors.Is(f.Err, sentinel) {
		t.Errorf("Failures[0] = %+v, want in-000004 at 0.2.0 wrapping the sentinel", f)
	}
	if r.Ran != 39 {
		t.Errorf("Ran = %d, want 39 — every other input still replayed", r.Ran)
	}
	if r.OK() {
		t.Error("OK() is true with a failure outstanding")
	}
}

// failing is a Subject that refuses one input and passes the rest through.
type failing struct {
	ver string
	on  string
	err error
}

func (f failing) Version() string { return f.ver }

func (f failing) Run(_ context.Context, in []byte) ([]byte, error) {
	if string(in) == f.on {
		return nil, f.err
	}
	return in, nil
}

func TestRunSkipsRetiredInputs(t *testing.T) {
	t.Parallel()
	store, hashes := storeWith(t, "alpha")
	m := manifestOver(hashes)
	m.Inputs[0].Status = corpus.Retired
	m.Inputs[0].Retired = &corpus.Retirement{
		AtVersion: "0.2.0", Reason: "feature removed", Decision: "GermlineThreeWayClose",
	}
	r, err := replay.Run(t.Context(), m, store, upper{"0.1.0", nil}, upper{"0.2.0", nil})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if r.Ran != 0 {
		t.Errorf("Ran = %d, want 0; a retired input must not pass or fail", r.Ran)
	}
}

// TestDirRefusesAnAlteredRecording: an input whose bytes no longer match its
// hash is not a recording any more, and replaying it would launder the edit.
func TestDirRefusesAnAlteredRecording(t *testing.T) {
	t.Parallel()
	store, hashes := storeWith(t, "alpha")
	if err := os.WriteFile(filepath.Join(string(store), hashes[0]), []byte("tampered"), 0o600); err != nil {
		t.Fatalf("tamper: %v", err)
	}
	if _, err := store.Get(hashes[0]); err == nil {
		t.Error("an altered recording was served")
	}
}

func TestRecordRefusesToRewriteAVersionsOutput(t *testing.T) {
	t.Parallel()
	store, hashes := storeWith(t, "alpha")
	m := manifestOver(hashes)
	m.Inputs[0].Outputs = map[string]string{"0.2.0": "sha256:" + replay.Hash([]byte("something else"))}
	r, err := replay.Run(t.Context(), m, store, upper{"0.1.0", func(b []byte) []byte { return b }},
		upper{"0.2.0", func(b []byte) []byte { return b }})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	err = r.Record(m)
	if err == nil || !strings.Contains(err.Error(), "difference to close") {
		t.Errorf("Record overwrote a recorded output: %v", err)
	}
}

func ledger(t *testing.T) *replay.Ledger {
	t.Helper()
	const src = "## Entries\n\n" +
		"- **Order of rendered items** — 0.1, `GermlineThreeWayClose`. Nothing depends on it.\n"
	d, err := boundary.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return &replay.Ledger{
		Boundary: d,
		LawNames: []string{"RenderIsIdempotent"},
		Store:    decide.Set{"GermlineThreeWayClose"},
	}
}

func aDiff() replay.Difference {
	return replay.Difference{Input: "in-000001", From: "0.1.0", To: "0.2.0"}
}

func TestCloseNeedsAReason(t *testing.T) {
	t.Parallel()
	_, err := ledger(t).AsImplementation(aDiff(), "GermlineThreeWayClose", "")
	if !errors.Is(err, replay.ErrNoReason) {
		t.Errorf("err = %v, want ErrNoReason", err)
	}
}

func TestCloseNeedsARecordedDecision(t *testing.T) {
	t.Parallel()
	l := ledger(t)
	for _, decision := range []string{"", "GermlineNeverWritten"} {
		_, err := l.AsImplementation(aDiff(), decision, "the parser was wrong")
		if !errors.Is(err, replay.ErrUnrecordedDecision) {
			t.Errorf("decision %q: err = %v, want ErrUnrecordedDecision", decision, err)
		}
	}
}

func TestLawCloseNeedsARegisteredLaw(t *testing.T) {
	t.Parallel()
	l := ledger(t)
	_, err := l.AsLaw(aDiff(), "GermlineThreeWayClose", "LawNobodyWrote", "it was missing a condition")
	if !errors.Is(err, replay.ErrUnknownLaw) {
		t.Errorf("err = %v, want ErrUnknownLaw", err)
	}
	if _, ok := l.AsLaw(aDiff(), "GermlineThreeWayClose", "RenderIsIdempotent", "condition added"); ok != nil {
		t.Errorf("a close naming a real law was refused: %v", ok)
	}
}

// TestBoundaryCloseNeedsAPublishedEntry is the check that keeps the boundary
// from being written to excuse the diff that prompted it.
func TestBoundaryCloseNeedsAPublishedEntry(t *testing.T) {
	t.Parallel()
	l := ledger(t)
	_, err := l.AsBoundary(aDiff(), "GermlineThreeWayClose", "Whatever is convenient", "licensed")
	if !errors.Is(err, replay.ErrUnpublishedBoundary) {
		t.Errorf("err = %v, want ErrUnpublishedBoundary", err)
	}
	ad, err := l.AsBoundary(aDiff(), "GermlineThreeWayClose", "Order of rendered items", "licensed")
	if err != nil {
		t.Fatalf("a close citing a published entry was refused: %v", err)
	}
	if ad.ClosedAs != corpus.ClosedAsBoundary {
		t.Errorf("ClosedAs = %q", ad.ClosedAs)
	}
}

// TestBoundaryCloseNeedsAnApprovedEntry is the check that keeps publishing
// from being approving: an entry the agent just wrote is published, and the
// close is refused until the approval hook says a human signed off. The
// refused close comes back filled in, as a proposal for that human.
func TestBoundaryCloseNeedsAnApprovedEntry(t *testing.T) {
	t.Parallel()
	l := ledger(t)
	approved := false
	l.Approve = func(string) (string, error) {
		if !approved {
			return "", errors.New("not committed")
		}
		return "committed abc", nil
	}
	proposal, err := l.AsBoundary(aDiff(), "GermlineThreeWayClose", "Order of rendered items", "licensed")
	if !errors.Is(err, replay.ErrAwaitingHuman) {
		t.Fatalf("err = %v, want ErrAwaitingHuman", err)
	}
	if proposal.Entry != "Order of rendered items" || proposal.Input != aDiff().Input {
		t.Errorf("the refused close came back without its proposal: %+v", proposal)
	}

	m := corpus.New("2026-09-02")
	replay.Propose(m, proposal)
	replay.Propose(m, proposal)
	open := []replay.Difference{aDiff(), {Input: "in-000002", From: "0.1.0", To: "0.2.0"}}
	awaiting, unproposed := replay.Awaiting(m, open)
	if len(m.ProposedCloses) != 1 || len(awaiting) != 1 || len(unproposed) != 1 {
		t.Fatalf("proposals %d, awaiting %v, unproposed %v", len(m.ProposedCloses), awaiting, unproposed)
	}

	approved = true
	ad, err := l.AsBoundary(aDiff(), "GermlineThreeWayClose", "Order of rendered items", "licensed")
	if err != nil {
		t.Fatalf("an approved entry was refused: %v", err)
	}
	if ad.Approval != "committed abc" {
		t.Errorf("Approval = %q, want the evidence the hook returned", ad.Approval)
	}
	if err = replay.Accept(m, ad); err != nil {
		t.Fatal(err)
	}
	if len(m.ProposedCloses) != 0 {
		t.Errorf("accepting the close left its proposal behind: %+v", m.ProposedCloses)
	}
}

func TestAcceptRefusesASecondCloseOfTheSameDifference(t *testing.T) {
	t.Parallel()
	m := corpus.New("2026-09-02")
	ad, err := ledger(t).AsImplementation(aDiff(), "GermlineThreeWayClose", "the parser was wrong")
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if first := replay.Accept(m, ad); first != nil {
		t.Fatalf("first Accept: %v", first)
	}
	if second := replay.Accept(m, ad); second == nil {
		t.Error("the same difference was closed twice")
	}
}

func TestOpenLeavesUnclosedDifferences(t *testing.T) {
	t.Parallel()
	m := corpus.New("2026-09-02")
	r := replay.Result{Differences: []replay.Difference{
		{Input: "in-000001", From: "0.1.0", To: "0.2.0"},
		{Input: "in-000002", From: "0.1.0", To: "0.2.0"},
	}}
	ad, err := ledger(t).AsImplementation(r.Differences[0], "GermlineThreeWayClose", "fixed")
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if accepted := replay.Accept(m, ad); accepted != nil {
		t.Fatalf("Accept: %v", accepted)
	}
	open := replay.Open(m, r)
	if len(open) != 1 || open[0].Input != "in-000002" {
		t.Errorf("Open = %v, want just in-000002", open)
	}
}
