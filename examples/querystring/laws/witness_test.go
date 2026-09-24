package laws

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/justinstimatze/germline/examples/querystring"
	"github.com/justinstimatze/germline/pkg/boundary"
)

// A boundary entry claims the suite cannot tell an implementation with this
// behavior from one without it. That claim is testable, and this file tests
// it: each witness below perturbs the codec on exactly one entry's observable
// and nothing else, and the suite must fail to notice. If it notices, the
// observable is pinned down by some law and is a promise whether or not
// anyone meant it to be — so the entry is wrong, and the test names it.
//
// Writing the perturbation as a decorator rather than a flag inside the codec
// keeps the production code free of knobs that exist only for a test.

// subject is the implementation the suite currently runs against. A witness
// swaps it and puts it back. Not parallel, for that reason.
var subject querystring.Codec = querystring.FormCodec{}

// lowerHexCodec differs from its inner codec on exactly one observable: the
// case of the hex digits in a rendered escape. Parse is inherited, because
// the reader already accepts both cases.
type lowerHexCodec struct{ querystring.Codec }

func (c lowerHexCodec) Render(q querystring.Query) string {
	rendered := c.Codec.Render(q)
	var sb strings.Builder
	for i := 0; i < len(rendered); i++ {
		if rendered[i] == '%' && i+2 < len(rendered) {
			sb.WriteString(strings.ToLower(rendered[i : i+3]))
			i += 2
			continue
		}
		sb.WriteByte(rendered[i])
	}
	return sb.String()
}

// bareKeyCodec differs on exactly one observable: a key whose value is empty
// renders without the "=". It round-trips, because parsing "a" and "a=" gives
// the same query, which is the very thing the entry declines to promise.
type bareKeyCodec struct{ querystring.Codec }

func (c bareKeyCodec) Render(q querystring.Query) string {
	rendered := strings.ReplaceAll(c.Codec.Render(q), "=&", "&")
	return strings.TrimSuffix(rendered, "=")
}

func swap(to querystring.Codec) func() {
	was := subject
	subject = to
	return func() { subject = was }
}

// TestEveryBoundaryEntryIsWitnessedAndHolds is the check that keeps the
// boundary honest. An unwitnessed entry is an unpaid bill; a distinguished
// one is a lie in a published document.
func TestEveryBoundaryEntryIsWitnessedAndHolds(t *testing.T) {
	doc, err := boundary.Load(filepath.Join("..", "BOUNDARY.md"))
	if err != nil {
		t.Fatalf("load boundary: %v", err)
	}
	witnesses := []boundary.Witness{
		{Entry: "How `+` is decoded", Perturb: func() func() {
			return swap(querystring.RFCCodec{})
		}},
		{Entry: "The case of hex digits in a rendered escape", Perturb: func() func() {
			return swap(lowerHexCodec{subject})
		}},
		{
			Entry:   "Whether a key with no `=` differs from one with an empty value",
			Perturb: func() func() { return swap(bareKeyCodec{subject}) },
		},
	}
	report, err := boundary.Distinguish(doc, witnesses, func() error {
		return violations(subject)
	})
	if err != nil {
		t.Fatalf("distinguish: %v", err)
	}
	for _, f := range report.Findings {
		t.Errorf("%s: %s — %s", f.Kind, f.Entry, f.Problem)
	}
	if len(report.Confirmed) != len(doc.Entries) {
		t.Errorf("confirmed %d of %d entries", len(report.Confirmed), len(doc.Entries))
	}
	if _, restored := subject.(querystring.FormCodec); !restored {
		t.Errorf("a perturbation survived the check: subject is %T", subject)
	}
}

// TestAPerturbationTheSuiteCatchesIsReportedAsAPromise: the check has to be
// able to fail, or the test above proves nothing. Reversing key order is not
// on the boundary and KeyOrderSurvives pins it down, so a witness claiming
// otherwise must come back Distinguished.
func TestAPerturbationTheSuiteCatchesIsReportedAsAPromise(t *testing.T) {
	const src = "## Entries\n\n- **Order of keys** — 0.1.0, `Whatever`. Claimed, wrongly.\n"
	doc, err := boundary.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	report, err := boundary.Distinguish(doc, []boundary.Witness{{
		Entry:   "Order of keys",
		Perturb: func() func() { return swap(reversedKeyCodec{subject}) },
	}}, func() error { return violations(subject) })
	if err != nil {
		t.Fatalf("distinguish: %v", err)
	}
	if report.OK() {
		t.Fatal("the suite did not notice reversed key order, which KeyOrderSurvives pins")
	}
	if report.Findings[0].Kind != boundary.Distinguished {
		t.Errorf("kind = %s, want distinguished", report.Findings[0].Kind)
	}
}

// reversedKeyCodec parses correctly and then reverses the key order.
type reversedKeyCodec struct{ querystring.Codec }

func (c reversedKeyCodec) Parse(raw string) (querystring.Query, error) {
	q, err := c.Codec.Parse(raw)
	if err != nil {
		return q, err
	}
	for i, j := 0, len(q.Keys)-1; i < j; i, j = i+1, j-1 {
		q.Keys[i], q.Keys[j] = q.Keys[j], q.Keys[i]
	}
	return q, nil
}
