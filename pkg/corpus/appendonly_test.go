package corpus_test

import (
	"strings"
	"testing"

	"github.com/justinstimatze/germline/pkg/corpus"
)

func TestAppendingIsClean(t *testing.T) {
	t.Parallel()
	was := one(t)
	now := one(t)
	now.Inputs[0].Weight = 0.9 // the profile moved; the recording did not
	now.Inputs[0].Outputs["0.2.0"] = "sha256:" + hashA
	now.Inputs = append(now.Inputs, corpus.Input{
		ID: "in-000002", RecordedAt: "2026-09-03T09:00:00Z", SHA256: hashB,
		Level: corpus.LevelAbstraction, Weight: 0.1, Status: corpus.Active,
	})
	now.AcceptedDiffs = append(now.AcceptedDiffs, corpus.AcceptedDiff{
		From: "0.1.0", To: "0.2.0", Input: "in-000001",
		ClosedAs: corpus.ClosedAsLaw, Decision: "GermlineExample", Reason: "missing condition",
	})
	if ps := corpus.VerifyAppendOnly(was, now); !ps.OK() {
		t.Errorf("appending was reported as a violation:\n%s", ps.Sorted().String())
	}
}

func TestEditsAreCaught(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		mutate func(*corpus.Manifest)
		want   string
	}{
		"rewriting an input": {
			func(m *corpus.Manifest) { m.Inputs[0].SHA256 = hashB },
			"cannot be rewritten",
		},
		"deleting an input": {
			func(m *corpus.Manifest) { m.Inputs = nil },
			"never deleted",
		},
		"rewriting an output": {
			func(m *corpus.Manifest) { m.Inputs[0].Outputs["0.1.0"] = "sha256:" + hashA },
			"is history",
		},
		"dropping an output": {
			func(m *corpus.Manifest) { delete(m.Inputs[0].Outputs, "0.1.0") },
			"is gone",
		},
		"moving the recording time": {
			func(m *corpus.Manifest) { m.Inputs[0].RecordedAt = "2020-01-01T00:00:00Z" },
			"recorded_at changed",
		},
		"changing the confidentiality level": {
			func(m *corpus.Manifest) { m.Inputs[0].Level = corpus.LevelProfileOnly },
			"is a new input",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			was, now := one(t), one(t)
			tc.mutate(now)
			ps := corpus.VerifyAppendOnly(was, now)
			if ps.OK() {
				t.Fatalf("%s was not caught", name)
			}
			if !strings.Contains(ps.String(), tc.want) {
				t.Errorf("problem does not mention %q:\n%s", tc.want, ps.String())
			}
		})
	}
}

func TestUnRetiringIsCaught(t *testing.T) {
	t.Parallel()
	was, now := one(t), one(t)
	was.Inputs[0].Status = corpus.Retired
	was.Inputs[0].Retired = &corpus.Retirement{
		AtVersion: "0.2.0", Reason: "feature removed", Decision: "GermlineExample",
	}
	ps := corpus.VerifyAppendOnly(was, now)
	if !strings.Contains(ps.String(), "active again") {
		t.Errorf("un-retiring was not caught:\n%s", ps.String())
	}
}

func TestEditingAnAcceptedDiffIsCaught(t *testing.T) {
	t.Parallel()
	d := corpus.AcceptedDiff{
		From: "0.1.0", To: "0.2.0", Input: "in-000001",
		ClosedAs: corpus.ClosedAsLaw, Decision: "GermlineExample", Reason: "missing condition",
	}
	was, now := one(t), one(t)
	was.AcceptedDiffs = []corpus.AcceptedDiff{d}
	edited := d
	edited.ClosedAs = corpus.ClosedAsBoundary
	now.AcceptedDiffs = []corpus.AcceptedDiff{edited}
	if ps := corpus.VerifyAppendOnly(was, now); !strings.Contains(ps.String(), "was edited") {
		t.Errorf("re-closing a diff a second way was not caught:\n%s", ps.String())
	}
	now.AcceptedDiffs = nil
	if ps := corpus.VerifyAppendOnly(was, now); !strings.Contains(ps.String(), "now gone") {
		t.Errorf("deleting an accepted diff was not caught:\n%s", ps.String())
	}
}
