package boundary_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/justinstimatze/germline/pkg/boundary"
)

// The subject under test is a formatter with two observables: the text it
// produces, which a law pins down, and the order it produces it in, which
// nothing does. The boundary declines the second and not the first.
type formatter struct {
	upper   bool
	reverse bool
}

func (f *formatter) render(items []string) string {
	out := make([]string, len(items))
	copy(out, items)
	if f.reverse {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	joined := strings.Join(out, ",")
	if f.upper {
		joined = strings.ToUpper(joined)
	}
	return joined
}

// suite is the law: rendering is case-preserving. It says nothing about order.
func (f *formatter) suite() error {
	if got := f.render([]string{"a", "b"}); strings.ToLower(got) != got {
		return errors.New("render changed case")
	}
	return nil
}

func twoEntryDoc(t *testing.T) *boundary.Document {
	t.Helper()
	const src = "## Entries\n\n" +
		"- **Order of rendered items** — 0.1, `GermlineExample`. Nothing depends on it.\n" +
		"- **Case of rendered items** — 0.1, `GermlineExample`. Also nothing.\n"
	d, err := boundary.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return d
}

// TestDistinguishCatchesAnEntryThatIsReallyAPromise is the whole point of the
// witness check: the document declines to promise case, but a law pins case
// down, so the entry is a lie and the check says so by name.
func TestDistinguishCatchesAnEntryThatIsReallyAPromise(t *testing.T) {
	t.Parallel()
	f := &formatter{}
	d := twoEntryDoc(t)
	witnesses := []boundary.Witness{
		{Entry: "Order of rendered items", Perturb: func() func() {
			f.reverse = true
			return func() { f.reverse = false }
		}},
		{Entry: "Case of rendered items", Perturb: func() func() {
			f.upper = true
			return func() { f.upper = false }
		}},
	}
	r, err := boundary.Distinguish(d, witnesses, f.suite)
	if err != nil {
		t.Fatalf("Distinguish: %v", err)
	}
	if r.OK() {
		t.Fatal("report is OK; the case entry is distinguished by a law")
	}
	if len(r.Confirmed) != 1 || r.Confirmed[0] != "Order of rendered items" {
		t.Errorf("Confirmed = %v, want just the order entry", r.Confirmed)
	}
	var found bool
	for _, fi := range r.Findings {
		if fi.Kind == boundary.Distinguished && fi.Entry == "Case of rendered items" {
			found = true
		}
	}
	if !found {
		t.Errorf("no Distinguished finding for the case entry; findings = %+v", r.Findings)
	}
}

func TestDistinguishReportsUnwitnessedAndUnclaimed(t *testing.T) {
	t.Parallel()
	f := &formatter{}
	d := twoEntryDoc(t)
	witnesses := []boundary.Witness{
		{Entry: "Order of rendered items", Perturb: func() func() {
			f.reverse = true
			return func() { f.reverse = false }
		}},
		{Entry: "Phase of the moon", Perturb: func() func() { return func() {} }},
	}
	r, err := boundary.Distinguish(d, witnesses, f.suite)
	if err != nil {
		t.Fatalf("Distinguish: %v", err)
	}
	if !r.OK() {
		t.Errorf("report is not OK; neither finding is a lie: %+v", r.Findings)
	}
	kinds := map[boundary.Kind]string{}
	for _, fi := range r.Findings {
		kinds[fi.Kind] = fi.Entry
	}
	if kinds[boundary.Unwitnessed] != "Case of rendered items" {
		t.Errorf("Unwitnessed = %q, want the case entry", kinds[boundary.Unwitnessed])
	}
	if kinds[boundary.Unclaimed] != "Phase of the moon" {
		t.Errorf("Unclaimed = %q, want the moon witness", kinds[boundary.Unclaimed])
	}
}

// TestDistinguishRefusesARedSuite: with a red baseline every perturbation
// looks caught, so every entry would read as a lie. Refuse instead.
func TestDistinguishRefusesARedSuite(t *testing.T) {
	t.Parallel()
	d := twoEntryDoc(t)
	red := func() error { return errors.New("already broken") }
	_, err := boundary.Distinguish(d, nil, red)
	if !errors.Is(err, boundary.ErrSuiteAlreadyRed) {
		t.Errorf("err = %v, want ErrSuiteAlreadyRed", err)
	}
}

// TestPerturbationIsAlwaysRestored: a witness that left its perturbation in
// place would poison every later entry's verdict silently.
func TestPerturbationIsAlwaysRestored(t *testing.T) {
	t.Parallel()
	f := &formatter{}
	d := twoEntryDoc(t)
	witnesses := []boundary.Witness{
		{Entry: "Order of rendered items", Perturb: func() func() {
			f.reverse = true
			return func() { f.reverse = false }
		}},
	}
	if _, err := boundary.Distinguish(d, witnesses, f.suite); err != nil {
		t.Fatalf("Distinguish: %v", err)
	}
	if f.reverse {
		t.Error("perturbation survived the check")
	}
}
