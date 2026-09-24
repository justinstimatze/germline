package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/justinstimatze/germline"
	"github.com/justinstimatze/germline/internal/atomicfile"
	"github.com/justinstimatze/germline/internal/metric"
	"github.com/justinstimatze/germline/internal/project"
	"github.com/justinstimatze/germline/pkg/boundary"
	"github.com/justinstimatze/germline/pkg/corpus"
	"github.com/justinstimatze/germline/pkg/law"
)

// errFailed is returned when a check found something. The message is already
// on stdout, so main adds nothing to it.
var errFailed = errors.New("check failed")

// flags builds a flag set that every subcommand shares, so -C works
// everywhere and nobody has to remember which commands accept it.
func flags(e *env, name string) *flag.FlagSet {
	fs := flag.NewFlagSet("germline "+name, flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	fs.StringVar(&e.dir, "C", e.dir, "work on this directory")
	fs.StringVar(&e.store, "store", "",
		"the decision record, overriding `git config winze.store`")
	return fs
}

func (e *env) project(fs *flag.FlagSet, args []string) (*project.Project, error) {
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	p, err := project.Open(e.dir)
	if err != nil {
		return nil, err
	}
	if e.store != "" {
		p.StoreDir = e.store
	}
	return p, nil
}

func cmdCheck(e *env, args []string) error {
	p, err := e.project(flags(e, "check"), args)
	if err != nil {
		return err
	}
	s := p.Status()

	fmt.Fprintf(e.stdout, "germline check — %s\n\n", s.Root)
	fmt.Fprintf(e.stdout, "  boundary   %d declined promises\n", s.BoundaryEntries)
	fmt.Fprintf(e.stdout, "  corpus     %d inputs, %d active, %.2f of the profile\n",
		s.CorpusInputs, s.CorpusActive, s.CorpusWeight)
	switch {
	case s.StoreDir != "":
		fmt.Fprintf(e.stdout, "  decisions  %d in %s\n", s.Decisions, s.StoreDir)
	case s.Unverified > 0:
		// The check passes here, so this line is the only place a reader
		// learns the citations went unresolved. It says how many rather than
		// that some did, because a clone with a record on it should see the
		// number go to zero and not just the sentence go away.
		fmt.Fprintf(e.stdout, "  decisions  no record configured, %s unverified\n",
			plural(s.Unverified, "citation"))
	default:
		fmt.Fprintln(e.stdout, "  decisions  no record configured")
	}
	fmt.Fprintf(e.stdout, "  laws       %d stated in prose and not yet executable\n", s.Unchecked)
	fmt.Fprintln(e.stdout)

	if s.OK() {
		fmt.Fprintln(e.stdout, "ok")
		return nil
	}
	for _, pr := range s.Problems {
		fmt.Fprintf(e.stdout, "  %s\n", pr)
	}
	fmt.Fprintf(e.stdout, "\n%s\n", plural(len(s.Problems), "problem"))
	return errFailed
}

func cmdMetrics(e *env, args []string) error {
	p, err := e.project(flags(e, "metrics"), args)
	if err != nil {
		return err
	}
	doc, _ := boundary.Load(p.Path(project.BoundaryFile)) //nolint:errcheck // a missing artifact reads as zero
	m, _ := corpus.Load(p.Path(project.ManifestFile))     //nolint:errcheck // same
	prose, err := p.Unchecked()
	if err != nil {
		return err
	}
	declared, err := law.Declared(p.Path(project.LawsDir))
	if err != nil {
		return err
	}
	set := metric.Set{Boundary: doc, Corpus: m, LawNames: declared, Prose: prose}
	fmt.Fprintf(e.stdout, "germline metrics — %s\n\n", p.Root)
	for _, r := range set.Readings() {
		fmt.Fprintf(e.stdout, "  %-26s %s\n", r.Name, r.Value)
		fmt.Fprintf(e.stdout, "  %-26s %s\n\n", "", r.Note)
	}
	return nil
}

func cmdLaws(e *env, args []string) error {
	p, err := e.project(flags(e, "laws"), args)
	if err != nil {
		return err
	}
	prose, err := p.Unchecked()
	if err != nil {
		return err
	}
	if len(prose) == 0 {
		fmt.Fprintf(e.stdout, "no %s comments in %s\n", law.Marker, p.Root)
		return nil
	}
	fmt.Fprintf(e.stdout, "%d laws stated in prose and not yet executable:\n\n", len(prose))
	for _, l := range prose {
		fmt.Fprintf(e.stdout, "  %s:%d\n    %s\n\n", l.File, l.Line, l.Statement)
	}
	return nil
}

func cmdBoundary(e *env, args []string) error {
	fset := flags(e, "boundary")
	write := fset.Bool("write", false,
		"rewrite the file from the decision record, or through the projection "+
			"when the record carries no declines")
	p, err := e.project(fset, args)
	if err != nil {
		return err
	}
	path := p.Path(project.BoundaryFile)
	doc, err := boundary.Load(path)
	if err != nil {
		return err
	}
	// The record is the source where it carries the entries. Where it does
	// not, the file is, and -write only canonicalises the wrapping; a project
	// adopts germline before it has anywhere to record a reason.
	projected, ps := p.Projected(doc)
	source := "the file"
	if projected != nil {
		doc, source = projected, "the record"
	}
	if *write {
		if !ps.OK() {
			return report(e, ps)
		}
		if wErr := atomicfile.Write(path, []byte(doc.String())); wErr != nil {
			return wErr
		}
		fmt.Fprintf(e.stdout, "rewrote %s from %s: %d entries, prose kept verbatim\n",
			project.BoundaryFile, source, len(doc.Entries))
		return nil
	}
	fmt.Fprintf(e.stdout, "%d observables this system declines to promise, from %s:\n\n",
		len(doc.Entries), source)
	for _, entry := range doc.Entries {
		fmt.Fprintf(e.stdout, "  %s\n    since %s, %s\n    %s\n\n",
			entry.Observable, entry.Since, entry.Decision, entry.Reason)
	}
	ps = append(ps, doc.Validate()...)
	if !ps.OK() {
		return report(e, ps)
	}
	return nil
}

// report prints what a check found and fails.
func report(e *env, ps germline.Problems) error {
	for _, pr := range ps.Sorted() {
		fmt.Fprintf(e.stdout, "  %s\n", pr)
	}
	return errFailed
}

func cmdCorpus(e *env, args []string) error {
	p, err := e.project(flags(e, "corpus"), args)
	if err != nil {
		return err
	}
	m, err := corpus.Load(p.Path(project.ManifestFile))
	if err != nil {
		return err
	}
	active, retired := 0, 0
	for _, in := range m.Inputs {
		if in.Status == corpus.Active {
			active++
			continue
		}
		retired++
	}
	fmt.Fprintf(e.stdout, "%d recorded inputs: %d active, %d retired\n", len(m.Inputs), active, retired)
	fmt.Fprintf(e.stdout, "active inputs carry %.4f of the operational profile\n", m.ActiveWeight())
	counts := m.Closes()
	fmt.Fprintf(e.stdout, "%d accepted diffs:", len(m.AcceptedDiffs))
	for _, c := range corpus.Closes() {
		fmt.Fprintf(e.stdout, " %s %d", c, counts[c])
	}
	fmt.Fprintln(e.stdout)
	if ps := m.Validate(); !ps.OK() {
		fmt.Fprintln(e.stdout)
		for _, pr := range ps.Sorted() {
			fmt.Fprintf(e.stdout, "  %s\n", pr)
		}
		return errFailed
	}
	return nil
}
