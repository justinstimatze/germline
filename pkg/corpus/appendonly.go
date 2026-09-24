package corpus

import (
	"fmt"

	"github.com/justinstimatze/germline"
)

// "Inputs are append-only and outputs are never edited" is the sentence the
// whole method rests on, and until it is a program it is a request. This file
// is the program. It compares a manifest against the last committed version
// of itself and reports every departure that is not an append.
//
// One field may move: Weight. The profile is re-estimated with recency
// weighting as traffic shifts, so a weight that never changed would be the
// bug. Everything else about a recording is fixed the moment it is recorded.

// VerifyAppendOnly reports every way now departs from was other than by
// appending. An empty result is the good case.
func VerifyAppendOnly(was, now *Manifest) germline.Problems {
	var ps germline.Problems
	add := func(where, what string) {
		ps = append(ps, germline.Problem{Artifact: germline.Corpus, Where: where, What: what})
	}
	for _, old := range was.Inputs {
		cur, ok := now.Find(old.ID)
		if !ok {
			add(old.ID, "was recorded and is now gone; a stale input is retired by an "+
				"appended event, never deleted")
			continue
		}
		compareInput(old, cur, add)
	}
	compareDiffs(was, now, add)
	return ps
}

func compareInput(old, cur Input, add func(where, what string)) {
	where := old.ID
	if cur.SHA256 != old.SHA256 {
		add(where, fmt.Sprintf("content hash changed from %s to %s; an input is what a "+
			"user did and cannot be rewritten", short(old.SHA256), short(cur.SHA256)))
	}
	if cur.RecordedAt != old.RecordedAt {
		add(where, fmt.Sprintf("recorded_at changed from %s to %s", old.RecordedAt, cur.RecordedAt))
	}
	if cur.Level != old.Level {
		add(where, fmt.Sprintf("confidentiality level changed from %d to %d; re-recording at "+
			"another level is a new input", old.Level, cur.Level))
	}
	for version, out := range old.Outputs {
		got, ok := cur.Outputs[version]
		switch {
		case !ok:
			add(where, fmt.Sprintf("the output recorded for version %s is gone; outputs are "+
				"re-derived per version and kept", version))
		case got != out:
			add(where, fmt.Sprintf("the output for version %s changed from %s to %s; what a "+
				"version did is history, and the thing to review is the diff to the next one",
				version, short(out), short(got)))
		}
	}
	compareStatus(old, cur, add)
}

func compareStatus(old, cur Input, add func(where, what string)) {
	where := old.ID
	if old.Status == Retired && cur.Status != Retired {
		add(where, "was retired and is active again; retirement is an appended event and "+
			"un-retiring is a new recording")
	}
	if old.Retired == nil || cur.Retired == nil {
		if old.Retired != nil && cur.Retired == nil {
			add(where, "lost its retirement event")
		}
		return
	}
	if *old.Retired != *cur.Retired {
		add(where, "the retirement event was edited; it records what happened and when")
	}
}

func compareDiffs(was, now *Manifest, add func(where, what string)) {
	index := make(map[[3]string]*AcceptedDiff, len(now.AcceptedDiffs))
	for i := range now.AcceptedDiffs {
		d := &now.AcceptedDiffs[i]
		index[diffKey(d)] = d
	}
	for i := range was.AcceptedDiffs {
		old := &was.AcceptedDiffs[i]
		where := fmt.Sprintf("diff %s %s→%s", old.Input, old.From, old.To)
		cur, ok := index[diffKey(old)]
		if !ok {
			add(where, "was accepted and is now gone; an accepted diff is a decision and "+
				"is superseded, not deleted")
			continue
		}
		if *cur != *old {
			add(where, fmt.Sprintf("was edited: closed as %q for %q, now %q for %q",
				old.ClosedAs, old.Reason, cur.ClosedAs, cur.Reason))
		}
	}
}

func diffKey(d *AcceptedDiff) [3]string { return [3]string{d.Input, d.From, d.To} }

func short(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}
