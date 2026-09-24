package laws

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/justinstimatze/germline"
	"github.com/justinstimatze/germline/internal/project"
	"github.com/justinstimatze/germline/pkg/boundary"
)

// A project's decision record can be private and outside the repository, so
// a fresh clone has none — and a CI runner is a fresh clone by definition.
// The law below is what makes passing without one safe rather than lax:
// taking the record away can only take problems away with it, so a check
// that goes green in a clone would have gone green on the machine that has
// the store.

// TestNoRecordIsNeverStricter quantifies over projects written to disk: the
// problems found with no record configured are a subset of those found with
// one. The failure it exists to catch is a check that fires only when the
// record is absent, which would fail every clone and every runner while
// passing on the one machine where somebody would notice.
//
// What it does not reach is the boundary projection. A generated record here
// carries decisions and no Declines claims, so the file stays the source and
// checkBoundaryRecord returns nothing in either run. That path is safe for a
// structural reason rather than a checked one: it reads the record first and
// gives up when there is none.
func TestNoRecordIsNeverStricter(t *testing.T) {
	t.Parallel()
	prop := func(g genProject) bool {
		root := t.TempDir()
		if err := g.write(root); err != nil {
			t.Errorf("write: %v", err)
			return false
		}
		p, err := project.Open(root)
		if err != nil {
			t.Errorf("open: %v", err)
			return false
		}
		if p.StoreDir == "" {
			t.Errorf("wrote a %s directory that Open did not find", project.DecisionsDir)
			return false
		}
		with := p.Check()
		p.StoreDir = ""
		return subset(p.Check(), with)
	}
	if err := quick.Check(prop, nil); err != nil {
		t.Errorf("NoRecordIsNeverStricter violated: %v", err)
	}
}

// subset reports whether every problem in a is also in b. Problems compare by
// value, which is what makes the statement about the problems themselves
// rather than about how many there are.
func subset(a, b germline.Problems) bool {
	in := make(map[germline.Problem]bool, len(b))
	for _, p := range b {
		in[p] = true
	}
	for _, p := range a {
		if !in[p] {
			return false
		}
	}
	return true
}

// genProject is a boundary document paired with a record carrying some of the
// decisions it cites and not the rest. Both halves are needed: a record that
// carried every citation would make the two checks agree for the uninteresting
// reason, and one that carried none would never resolve anything.
type genProject struct {
	doc     *boundary.Document
	records []string
}

// Generate builds the pair, dropping about a quarter of the memories and
// adding a few nothing cites — a record serves more than one project, so
// carrying a name no artifact mentions is its ordinary state.
func (genProject) Generate(rnd *rand.Rand, size int) reflect.Value {
	base, ok := genDoc{}.Generate(rnd, size).Interface().(genDoc)
	if !ok {
		panic("genDoc.Generate returned something other than a genDoc")
	}
	var records []string
	for _, e := range base.doc.Entries {
		if rnd.Intn(4) == 0 {
			continue // the memory behind this entry was never written
		}
		records = append(records, e.Decision)
	}
	for range rnd.Intn(3) {
		records = append(records, identifier(rnd))
	}
	return reflect.ValueOf(genProject{doc: base.doc, records: records})
}

// write lays the project out at the conventional paths, with the record as a
// decisions directory the project ships with itself. That is the sample
// system's shape, and it means Open finds a record with nothing configured.
func (g genProject) write(root string) error {
	var store strings.Builder
	store.WriteString("package decisions\n")
	seen := make(map[string]bool, len(g.records))
	for _, name := range g.records {
		if seen[name] {
			continue
		}
		seen[name] = true
		fmt.Fprintf(&store, "\nvar %s = 1\n", name)
	}
	files := map[string]string{
		project.BoundaryFile:                             g.doc.String(),
		project.ManifestFile:                             emptyManifest,
		filepath.Join(project.DecisionsDir, "memory.go"): store.String(),
	}
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// emptyManifest is a valid corpus with nothing recorded in it. A manifest
// that failed to load would put the same problem in both results and make the
// subset hold for a reason that has nothing to do with the record.
const emptyManifest = `{
  "schema": "germline.corpus.manifest/0",
  "profile": {
    "estimated_at": "2026-09-02",
    "window_days": 90,
    "recency_half_life_days": 30
  },
  "inputs": []
}
`
