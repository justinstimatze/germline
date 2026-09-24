package corpus_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/justinstimatze/germline/pkg/corpus"
)

const (
	hashA = "0000000000000000000000000000000000000000000000000000000000000000"
	hashB = "1111111111111111111111111111111111111111111111111111111111111111"
)

func one(t *testing.T) *corpus.Manifest {
	t.Helper()
	m := corpus.New("2026-09-02")
	m.Inputs = []corpus.Input{{
		ID:         "in-000001",
		RecordedAt: "2026-09-02T21:04:00Z",
		SHA256:     hashA,
		Level:      corpus.LevelAbstraction,
		Weight:     0.25,
		Status:     corpus.Active,
		Outputs:    map[string]string{"0.1.0": "sha256:" + hashB},
	}}
	return m
}

// TestTheShippedExampleValidates: manifest.example.json is what a project
// copies to start. If the example does not pass the checker, every project
// starts red.
func TestTheShippedExampleValidates(t *testing.T) {
	t.Parallel()
	m, err := corpus.Load(filepath.Join("..", "..", "corpus", "manifest.example.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ps := m.Validate(); !ps.OK() {
		t.Errorf("the shipped example does not validate:\n%s", ps.Sorted().String())
	}
}

// TestThisRepoManifestValidates: germline is the first system germline
// governs, and its own manifest is checked by its own checker.
func TestThisRepoManifestValidates(t *testing.T) {
	t.Parallel()
	m, err := corpus.Load(filepath.Join("..", "..", "corpus", "manifest.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ps := m.Validate(); !ps.OK() {
		t.Errorf("this repository's manifest does not validate:\n%s", ps.Sorted().String())
	}
}

func TestValidateCatchesMalformedInput(t *testing.T) {
	t.Parallel()
	m := one(t)
	m.Inputs[0].SHA256 = "not-a-hash"
	m.Inputs[0].Level = 9
	m.Inputs[0].Weight = 3
	m.Inputs[0].Status = "maybe"
	ps := m.Validate()
	for _, want := range []string{"sha256", "level", "weight", "status"} {
		if !strings.Contains(ps.String(), want) {
			t.Errorf("no problem mentioning %q in:\n%s", want, ps.String())
		}
	}
}

// TestValidateRefusesAProfileOverOne covers weights that were each fine and
// together claim more than the whole profile, which coverage would then
// report as better than complete.
func TestValidateRefusesAProfileOverOne(t *testing.T) {
	t.Parallel()
	m := one(t)
	second := m.Inputs[0]
	second.ID, second.SHA256, second.Weight = "in-000002", hashB, 0.75
	m.Inputs = append(m.Inputs, second)
	if ps := m.Validate(); !ps.OK() {
		t.Fatalf("a profile totalling exactly 1 was refused:\n%s", ps.String())
	}
	m.Inputs[1].Weight = 0.8
	if ps := m.Validate(); !strings.Contains(ps.String(), "cannot total more than 1") {
		t.Errorf("a profile totalling 1.05 passed:\n%s", ps.String())
	}
}

func TestValidateRequiresADecisionOnEveryClose(t *testing.T) {
	t.Parallel()
	m := one(t)
	m.AcceptedDiffs = []corpus.AcceptedDiff{{
		From: "0.1.0", To: "0.2.0", Input: "in-000001",
		ClosedAs: corpus.ClosedAsLaw, Reason: "the law was missing a condition",
	}}
	if ps := m.Validate(); !strings.Contains(ps.String(), "names no decision") {
		t.Errorf("a close with no decision was accepted:\n%s", ps.String())
	}
}

func TestValidateRefusesAFourthClose(t *testing.T) {
	t.Parallel()
	m := one(t)
	m.AcceptedDiffs = []corpus.AcceptedDiff{{
		From: "0.1.0", To: "0.2.0", Input: "in-000001",
		ClosedAs: "edited the corpus", Decision: "GermlineExample", Reason: "seemed fine",
	}}
	if ps := m.Validate(); !strings.Contains(ps.String(), "the only three closes") {
		t.Errorf("a fourth close was accepted:\n%s", ps.String())
	}
}

func TestValidateRefusesADiffAgainstAnAbsentInput(t *testing.T) {
	t.Parallel()
	m := one(t)
	m.AcceptedDiffs = []corpus.AcceptedDiff{{
		From: "0.1.0", To: "0.2.0", Input: "in-999999",
		ClosedAs: corpus.ClosedAsLaw, Decision: "GermlineExample", Reason: "r",
	}}
	if ps := m.Validate(); !strings.Contains(ps.String(), "not in the manifest") {
		t.Errorf("a diff against a missing input was accepted:\n%s", ps.String())
	}
}

func TestActiveWeightIgnoresRetired(t *testing.T) {
	t.Parallel()
	m := one(t)
	m.Inputs = append(m.Inputs, corpus.Input{
		ID: "in-000002", RecordedAt: "2026-09-02T21:05:00Z", SHA256: hashB,
		Level: corpus.LevelAbstraction, Weight: 0.5, Status: corpus.Retired,
		Retired: &corpus.Retirement{
			AtVersion: "0.2.0", Reason: "feature removed", Decision: "GermlineExample",
		},
	})
	if got := m.ActiveWeight(); got != 0.25 {
		t.Errorf("ActiveWeight() = %v, want 0.25", got)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Parallel()
	m := one(t)
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := m.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := corpus.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Inputs) != 1 || got.Inputs[0].SHA256 != hashA {
		t.Errorf("round trip lost the input: %+v", got.Inputs)
	}
	if ps := got.Validate(); !ps.OK() {
		t.Errorf("round-tripped manifest does not validate:\n%s", ps.String())
	}
}

// TestAReaderNeverSeesAHalfWrittenManifest is the property that makes Save
// safe to call in a loop.
//
// It used to be os.WriteFile, which truncates and then writes, so a reader
// arriving in between sees a corpus that has lost most of itself — and
// closing two thousand differences rewrites the manifest two thousand times,
// which is how a jq on it caught a half-written file on the first attempt.
// The failure a rename also prevents is the one that costs more: a crash or a
// full disk part-way through, leaving nothing to load at all.
func TestAReaderNeverSeesAHalfWrittenManifest(t *testing.T) {
	t.Parallel()
	m := one(t)
	for i := 2; i <= 200; i++ {
		in := m.Inputs[0]
		in.ID = fmt.Sprintf("in-%06d", i)
		in.SHA256 = hashB
		m.Inputs = append(m.Inputs, in)
	}
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := m.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 50 {
			if err := m.Save(path); err != nil {
				t.Errorf("Save: %v", err)
				break
			}
		}
		close(done)
	}()
	reads := 0
	for {
		select {
		case <-done:
			wg.Wait()
			if reads == 0 {
				t.Error("the reader never got a turn, so this proved nothing")
			}
			return
		default:
		}
		got, err := corpus.Load(path)
		if err != nil {
			t.Fatalf("Load after %d clean reads: %v", reads, err)
		}
		if len(got.Inputs) != len(m.Inputs) {
			t.Fatalf("read %d inputs, want %d", len(got.Inputs), len(m.Inputs))
		}
		reads++
	}
}

// TestLoadOfAMissingFileIsAnEmptyCorpus: a project with nothing recorded yet
// has an empty corpus, not a broken one, and must not be red on day one.
func TestLoadOfAMissingFileIsAnEmptyCorpus(t *testing.T) {
	t.Parallel()
	m, err := corpus.Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(m.Inputs) != 0 {
		t.Errorf("got %d inputs, want 0", len(m.Inputs))
	}
	if ps := m.Validate(); !ps.OK() {
		t.Errorf("an empty corpus does not validate:\n%s", ps.String())
	}
}
