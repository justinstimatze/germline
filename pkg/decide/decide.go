// Package decide reads the decision record, so that a boundary entry or an
// accepted diff citing a reason can be checked against a record that actually
// carries one.
//
// A citation nobody verifies is a citation to nothing, and the two artifacts
// that lean hardest on the record — the boundary and the close ledger — are
// exactly the two where being wrong is cheapest to hide. The Store interface
// is the port; Winze is the one implementation shipped, and a project keeping
// its decisions somewhere else implements Store instead of moving them.
package decide

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// A Store is where decisions live. Names are the identifiers a boundary entry
// or an accepted diff cites.
type Store interface {
	// Names returns every decision the store carries.
	Names() ([]string, error)
}

// A Boundary is a store that carries the declined promises themselves rather
// than only the names of the memories behind them. It is a second interface
// and not a second method on Store because carrying them is what a record
// grows into: a project adopts germline with a boundary file and a citation,
// and moves the entries into the record when it has somewhere to put them.
//
// Where a store implements this, the record is the source and the file is the
// projection. Where it does not, the file is the source, and the only thing
// checked is that its citations resolve.
type Boundary interface {
	Store
	// Declines returns the declined promises, in declaration order.
	Declines() ([]Declined, error)
}

// ErrNoStore reports a configured store directory that is not there. It is
// its own error because the fix is a path, not a decision.
var ErrNoStore = errors.New("decision store directory does not exist")

// Winze reads a winze store: a Go module whose top-level variable
// declarations are its memories, and whose compiler is its consistency
// checker. Reading the source directly rather than shelling out to
// winze-query means a check works on a machine that has the store but not the
// tool, which is every CI runner.
type Winze struct {
	// Dir is the store's module directory.
	Dir string

	loaded   bool
	names    []string
	declines []Declined
}

// Names returns every memory in the store, sorted. The result is cached, so a
// check that resolves a hundred citations parses the store once.
func (w *Winze) Names() ([]string, error) {
	if err := w.load(); err != nil {
		return nil, err
	}
	return w.names, nil
}

// Declines returns the declined promises the store carries, in the order they
// are declared: files by name, and within a file, top to bottom.
//
// Sorted would be wrong. A boundary is appended to and never edited, so the
// order of its entries is the order they were declared in, and a projection
// that sorted them would rewrite that history on the first regeneration.
func (w *Winze) Declines() ([]Declined, error) {
	if err := w.load(); err != nil {
		return nil, err
	}
	return w.declines, nil
}

// load parses the store once, filling both what it carries. Names and
// declines come out of the same walk because they come out of the same
// declarations: a decline is a memory too.
func (w *Winze) load() error {
	if w.loaded {
		return nil
	}
	info, err := os.Stat(w.Dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNoStore, w.Dir)
	}
	files, err := filepath.Glob(filepath.Join(w.Dir, "*.go"))
	if err != nil {
		return err
	}
	var (
		names    []string
		declines []Declined
	)
	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		found, declared, readErr := readStoreFile(fset, path)
		if readErr != nil {
			return readErr
		}
		names = append(names, found...)
		declines = append(declines, declared...)
	}
	sort.Strings(names)
	w.names, w.declines, w.loaded = names, declines, true
	return nil
}

// A Declined is one declined promise as the decision record carries it: the
// four fields a boundary entry publishes, plus the system that declines it.
//
// The system is here because one record can serve several, as when a
// repository holds a framework beside a sample system that uses it. An
// observable only means something against the system that could
// have promised it, so a record that did not say which would project every
// system's declines into every system's file.
type Declined struct {
	// System is the project that declines the observable, by the name the
	// record knows it under.
	System string
	// Observable is what someone could see and is not promised.
	Observable string
	// Since is the version the decline was declared at.
	Since string
	// Reason is the one line the boundary file publishes.
	Reason string
	// Decision is the memory the claim's subject names: the identifier the
	// boundary entry cites, which is what makes the citation resolvable.
	Decision string
	// File, Line and EndLine locate the declaration in the record, first line
	// to last. Whether a human approved the decline is read from the history
	// of exactly these lines.
	File          string
	Line, EndLine int
}

// readStoreFile returns the exported top-level variable names declared in one
// file, and the declined promises among them. A memory is a var; a helper
// type or function is not.
func readStoreFile(fset *token.FileSet, path string) ([]string, []Declined, error) {
	f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	var (
		names    []string
		declines []Declined
	)
	for _, d := range f.Decls {
		gen, ok := d.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, isValue := spec.(*ast.ValueSpec)
			if !isValue {
				continue
			}
			for i, id := range vs.Names {
				if !id.IsExported() {
					// Not a memory, and so not a decline either: an
					// unexported declaration is a draft, and a draft
					// promise nobody can cite is not withdrawn.
					continue
				}
				names = append(names, id.Name)
				if i >= len(vs.Values) {
					continue
				}
				dec, isDecline := declined(vs.Values[i])
				if !isDecline {
					continue
				}
				if bad := dec.incomplete(); bad != "" {
					return nil, nil, fmt.Errorf("%s: %s declines an observable and %s",
						path, id.Name, bad)
				}
				dec.File = path
				dec.Line = fset.Position(id.Pos()).Line
				dec.EndLine = fset.Position(vs.Values[i].End()).Line
				declines = append(declines, dec)
			}
		}
	}
	return names, declines, nil
}

// incomplete names the field a decline is missing, or the empty string. It is
// an error rather than a skip: a decline the projection cannot use is a
// promise withdrawn in the record and still published in the file, which is
// the one direction this must never fail quietly in.
func (d Declined) incomplete() string {
	switch {
	case d.System == "":
		return "names no system, so no boundary file would ever carry it"
	case d.Since == "":
		return "names no version; the boundary is published before dependents exist"
	case d.Reason == "":
		return "gives no reason"
	case d.Decision == "":
		return "has no subject, so the entry would cite nothing"
	}
	return ""
}

// declined reads a decline out of a claim, structurally rather than by type
// name. The store this was written against calls the predicate Declines and
// the payload DeclinedPromise, and the sample system's record is a different
// package with its own types; recognizing the shape rather than the names is
// what lets one reader serve both.
//
// The discriminator is an Object with an Observable. Nothing else in a
// decision record has that shape, and a claim that has it and is missing the
// rest is reported rather than skipped.
func declined(expr ast.Expr) (Declined, bool) {
	lit, ok := composite(expr)
	if !ok {
		return Declined{}, false
	}
	var (
		d      Declined
		object *ast.CompositeLit
	)
	for _, el := range lit.Elts {
		key, value, isField := field(el)
		if !isField {
			continue
		}
		switch key {
		case "Subject":
			d.Decision = leadingIdent(value)
		case "Object":
			object, _ = composite(value)
		}
	}
	if object == nil {
		return Declined{}, false
	}
	for _, el := range object.Elts {
		key, value, isField := field(el)
		if !isField {
			continue
		}
		switch key {
		case "System":
			d.System = stringValue(value)
		case "Observable":
			d.Observable = stringValue(value)
		case "Since":
			d.Since = stringValue(value)
		case "Reason":
			d.Reason = stringValue(value)
		}
	}
	if d.Observable == "" {
		return Declined{}, false
	}
	return d, true
}

// field splits a keyed composite-literal element. A positional element has no
// key and is not one of these.
func field(el ast.Expr) (key string, value ast.Expr, ok bool) {
	kv, isPair := el.(*ast.KeyValueExpr)
	if !isPair {
		return "", nil, false
	}
	id, isIdent := kv.Key.(*ast.Ident)
	if !isIdent {
		return "", nil, false
	}
	return id.Name, kv.Value, true
}

// composite unwraps an address-of and any parentheses to reach a composite
// literal, so that &T{...} and T{...} read the same.
func composite(expr ast.Expr) (*ast.CompositeLit, bool) {
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.UnaryExpr:
			if e.Op != token.AND {
				return nil, false
			}
			expr = e.X
		case *ast.CompositeLit:
			return e, true
		default:
			return nil, false
		}
	}
}

// leadingIdent returns the name at the head of a selector chain, so that both
// GermlineThreeWayClose and GermlineThreeWayClose.Entity give the memory's
// own name — which is the identifier a boundary entry cites.
func leadingIdent(expr ast.Expr) string {
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.UnaryExpr:
			expr = e.X
		case *ast.SelectorExpr:
			expr = e.X
		case *ast.Ident:
			return e.Name
		default:
			return ""
		}
	}
}

// stringValue evaluates a string expression, which in a record is either a
// literal or literals joined by +. Concatenation is not an edge case here:
// a reason is a sentence, a store is gofmt'd, and a sentence longer than a
// line arrives split.
func stringValue(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return stringValue(e.X)
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return ""
		}
		s, err := strconv.Unquote(e.Value)
		if err != nil {
			return ""
		}
		return s
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return ""
		}
		return stringValue(e.X) + stringValue(e.Y)
	default:
		return ""
	}
}

// Set is a store held in memory, for tests and for a project whose decisions
// come from somewhere this package does not know about.
type Set []string

// Names returns the set.
func (s Set) Names() ([]string, error) { return s, nil }

// Missing returns the cited names the store does not carry, sorted and
// deduplicated. An empty result is the good case.
func Missing(s Store, cited []string) ([]string, error) {
	names, err := s.Names()
	if err != nil {
		return nil, err
	}
	have := make(map[string]bool, len(names))
	for _, n := range names {
		have[n] = true
	}
	seen := make(map[string]bool, len(cited))
	var missing []string
	for _, c := range cited {
		if c == "" || have[c] || seen[c] {
			continue
		}
		seen[c] = true
		missing = append(missing, c)
	}
	sort.Strings(missing)
	return missing, nil
}
