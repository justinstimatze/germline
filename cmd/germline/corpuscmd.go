package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/justinstimatze/germline/internal/project"
	"github.com/justinstimatze/germline/pkg/boundary"
	"github.com/justinstimatze/germline/pkg/corpus"
	"github.com/justinstimatze/germline/pkg/replay"
)

// record, replay and close are the loop. Everything else in this command is
// reporting; these three are the parts that change the artifacts, and each
// one refuses the shortcut that would make it easy and wrong.

func cmdRecord(e *env, args []string) error {
	fset := flags(e, "record")
	level := fset.Int("level", int(corpus.LevelAbstraction),
		"confidentiality level 1..4; see corpus/README.md")
	weight := fset.Float64("weight", 0,
		"this input's share of the operational profile")
	p, err := e.project(fset, args)
	if err != nil {
		return err
	}
	files := fset.Args()
	if len(files) == 0 {
		return errors.New("record takes one or more files holding recorded input")
	}

	manifestPath := p.Path(project.ManifestFile)
	m, err := corpus.Load(manifestPath)
	if err != nil {
		return err
	}
	inputs := p.Path(project.InputsDir)
	//nolint:gosec // the directory is the project's own corpus, under its root
	if mkErr := os.MkdirAll(inputs, 0o750); mkErr != nil {
		return mkErr
	}

	next := nextID(m)
	for _, f := range files {
		payload, readErr := os.ReadFile(f) //nolint:gosec // a path the operator named
		if readErr != nil {
			return readErr
		}
		hash := replay.Hash(payload)
		path := filepath.Join(inputs, hash)
		if existing, ok := findByHash(m, hash); ok {
			// The same bytes under the same hash cannot rewrite a recording,
			// so a lost data file comes back by recording it again.
			//nolint:gosec // the name is a content hash under the project's corpus
			if _, statErr := os.Stat(path); errors.Is(statErr, fs.ErrNotExist) {
				//nolint:gosec // the name is a content hash under the project's corpus
				if wErr := os.WriteFile(path, payload, 0o600); wErr != nil {
					return wErr
				}
				fmt.Fprintf(e.stdout, "%s is already recorded as %s; restored its data\n", f, existing)
				continue
			}
			fmt.Fprintf(e.stdout, "%s is already recorded as %s\n", f, existing)
			continue
		}
		//nolint:gosec // the name is a content hash under the project's corpus
		if wErr := os.WriteFile(path, payload, 0o600); wErr != nil {
			return wErr
		}
		id := fmt.Sprintf("in-%06d", next)
		next++
		m.Inputs = append(m.Inputs, corpus.Input{
			ID:         id,
			RecordedAt: time.Now().UTC().Format(time.RFC3339),
			SHA256:     hash,
			Level:      corpus.Level(*level),
			Weight:     *weight,
			Status:     corpus.Active,
		})
		fmt.Fprintf(e.stdout, "recorded %s as %s (%s)\n", f, id, hash[:12])
	}
	if ps := m.Validate(); !ps.OK() {
		return fmt.Errorf("the manifest would not validate:\n%s", ps.Sorted())
	}
	return m.Save(manifestPath)
}

func nextID(m *corpus.Manifest) int {
	highest := 0
	for _, in := range m.Inputs {
		n, err := strconv.Atoi(strings.TrimPrefix(in.ID, "in-"))
		if err == nil && n > highest {
			highest = n
		}
	}
	return highest + 1
}

func findByHash(m *corpus.Manifest, hash string) (string, bool) {
	for _, in := range m.Inputs {
		if in.SHA256 == hash {
			return in.ID, true
		}
	}
	return "", false
}

func cmdReplay(e *env, args []string) error {
	fset := flags(e, "replay")
	wasSpec := fset.String("was", "", "the previous version: `version=command args...`")
	nowSpec := fset.String("now", "", "the new version: `version=command args...`")
	record := fset.Bool("record", false, "write the new version's outputs into the manifest")
	workers := fset.Int("workers", 1, "replay this many inputs at once; more than one only if the subjects are safe to run concurrently")
	p, err := e.project(fset, args)
	if err != nil {
		return err
	}
	was, err := subject(*wasSpec, p.Root)
	if err != nil {
		return fmt.Errorf("-was: %w", err)
	}
	now, err := subject(*nowSpec, p.Root)
	if err != nil {
		return fmt.Errorf("-now: %w", err)
	}

	// A replay is a gate, and a gate that cannot see the corpus must not read
	// as one that saw nothing wrong. corpus.Load treats a missing manifest as
	// an empty corpus, which is right for `check` on a new project and wrong
	// here, and a manifest that fails validation can hide every input from
	// the replay (a status of "Active" is not "active").
	manifestPath := p.Path(project.ManifestFile)
	if _, statErr := os.Stat(manifestPath); statErr != nil { //nolint:gosec // the project's own manifest path
		return fmt.Errorf("no corpus to replay: %w (run germline init)", statErr)
	}
	m, err := corpus.Load(manifestPath)
	if err != nil {
		return err
	}
	if ps := m.Validate(); !ps.OK() {
		for _, pr := range ps.Sorted() {
			fmt.Fprintf(e.stdout, "  %s\n", pr)
		}
		return fmt.Errorf("%w: the manifest does not validate, so the replay would not see every input", errFailed)
	}
	r, err := replay.Run(e.ctx(), m, replay.Dir(p.Path(project.InputsDir)), was, now,
		replay.Workers(*workers),
		replay.Progress(func(done, total int) {
			fmt.Fprintf(e.stderr, "replayed %d/%d\n", done, total)
		}))
	if err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "replayed %s from %s to %s: %d the same, %d different, %d failed\n",
		plural(r.Ran, "input"), r.From, r.To, r.Same, len(r.Differences), len(r.Failures))
	if len(r.Failures) > 0 {
		fmt.Fprintf(e.stdout, "\n%s could not be compared at all:\n\n", plural(len(r.Failures), "input"))
		for _, f := range r.Failures {
			fmt.Fprintf(e.stdout, "  %s\n", f)
		}
	}
	if active := activeInputs(m); r.Ran == 0 && active > 0 {
		return fmt.Errorf("%w: %s active in the manifest and none replayed", replay.ErrNoOutputs,
			plural(active, "input"))
	}

	if *record {
		if recErr := r.Record(m); recErr != nil {
			return recErr
		}
		if saveErr := m.Save(manifestPath); saveErr != nil {
			return saveErr
		}
		fmt.Fprintf(e.stdout, "recorded %d outputs against %s\n", len(r.Outputs), r.To)
	}

	open := replay.Open(m, r)
	awaiting, unproposed := replay.Awaiting(m, open)
	if len(open) == 0 {
		fmt.Fprintln(e.stdout, "\nno open differences")
	}
	if len(unproposed) > 0 {
		fmt.Fprintf(e.stdout, "\n%s with nothing closing them:\n\n", plural(len(unproposed), "difference"))
		for _, d := range unproposed {
			fmt.Fprintf(e.stdout, "  %s\n%s", d, outputs(m, d))
		}
		fmt.Fprintf(e.stdout, "\nClose each one exactly one of three ways:\n"+
			"  germline close -input %s -from %s -to %s -as implementation|law|boundary ...\n",
			unproposed[0].Input, unproposed[0].From, unproposed[0].To)
	}
	if len(awaiting) > 0 {
		fmt.Fprintf(e.stdout, "\n%s proposed as boundary, waiting on a human to approve the entry:\n\n",
			plural(len(awaiting), "difference"))
		for _, d := range awaiting {
			fmt.Fprintf(e.stdout, "  %s\n%s", d, outputs(m, d))
		}
		fmt.Fprintf(e.stdout, "\n%s\n", awaitingHuman)
	}
	switch {
	case len(unproposed) > 0 || len(r.Failures) > 0:
		return errFailed
	case len(awaiting) > 0:
		return errAwaiting
	}
	return nil
}

// previewBytes is how much of each output a difference shows. Enough to see
// what changed on a line-sized output, and short enough that a large one
// does not bury the rest of the list.
const previewBytes = 160

// outputs renders both versions' outputs for a difference, when the input's
// level allows printing them: a generated input (profile only) or an
// abstracted one carries none of anyone's data. A live-traffic or concrete
// input stays as hashes, because a terminal log is somewhere that data was
// never meant to go.
func outputs(m *corpus.Manifest, d replay.Difference) string {
	in, ok := m.Find(d.Input)
	if !ok || (in.Level != corpus.LevelProfileOnly && in.Level != corpus.LevelAbstraction) {
		return ""
	}
	show := func(b []byte) string {
		if len(b) > previewBytes {
			return fmt.Sprintf("%q… (%d bytes)", b[:previewBytes], len(b))
		}
		return fmt.Sprintf("%q", b)
	}
	return fmt.Sprintf("      %s: %s\n      %s: %s\n", d.From, show(d.Was), d.To, show(d.Now))
}

// activeInputs counts the inputs a replay is expected to run.
func activeInputs(m *corpus.Manifest) int {
	n := 0
	for _, in := range m.Inputs {
		if in.Status == corpus.Active {
			n++
		}
	}
	return n
}

// subject parses a `version=command args...` spec into a replay subject. The
// command reads one recorded input on stdin and writes its output on stdout,
// which is the whole contract, and the reason a C binary and a Python service
// are both replayable.
//
// The subject runs in the project directory, but a relative command path is
// the caller's, like every other path on the command line: it is resolved
// against the directory germline was run from. A bare name is left to PATH.
func subject(spec, dir string) (replay.Subject, error) {
	version, rest, ok := strings.Cut(spec, "=")
	if !ok || version == "" || strings.TrimSpace(rest) == "" {
		return nil, fmt.Errorf("want `version=command args...`, got %q", spec)
	}
	argv := strings.Fields(rest)
	if strings.ContainsRune(argv[0], filepath.Separator) && !filepath.IsAbs(argv[0]) {
		abs, err := filepath.Abs(argv[0])
		if err != nil {
			return nil, err
		}
		argv[0] = abs
	}
	return &replay.Command{
		Ver: version, Path: argv[0], Args: argv[1:], Dir: dir,
	}, nil
}

func cmdClose(e *env, args []string) error {
	fset := flags(e, "close")
	var (
		input    = fset.String("input", "", "the corpus input id the difference is on")
		from     = fset.String("from", "", "the previous version")
		to       = fset.String("to", "", "the new version")
		as       = fset.String("as", "", "implementation, law, or boundary")
		decision = fset.String("decision", "", "the decision record memory carrying the reason")
		reason   = fset.String("reason", "", "one line of that reason")
		lawName  = fset.String("law", "", "for -as law: the law that gained the condition")
		entry    = fset.String("entry", "", "for -as boundary: the published entry, verbatim")
	)
	p, err := e.project(fset, args)
	if err != nil {
		return err
	}
	if *input == "" || *from == "" || *to == "" {
		return errors.New("close needs -input, -from and -to to name the difference")
	}
	// Which of the three comes first, before anything is loaded: it is the
	// cheapest refusal and the one that teaches the shape of the loop.
	if kind := corpus.Close(*as); !kind.Valid() {
		return fmt.Errorf("-as %q; the only three closes are %v, and editing the "+
			"corpus is not one of them", *as, corpus.Closes())
	}

	doc, err := boundary.Load(p.Path(project.BoundaryFile))
	if err != nil {
		return err
	}
	store := p.Store()
	if store == nil {
		return errors.New("no decision record configured; run: git config winze.store <dir>")
	}
	ledger := &replay.Ledger{
		Boundary: doc, Store: store, LawNames: lawNames(p), Approve: p.Approver(e.ctx()),
	}
	d := replay.Difference{Input: *input, From: *from, To: *to}

	var ad corpus.AcceptedDiff
	switch corpus.Close(*as) {
	case corpus.ClosedAsImplementation:
		ad, err = ledger.AsImplementation(d, *decision, *reason)
	case corpus.ClosedAsLaw:
		ad, err = ledger.AsLaw(d, *decision, *lawName, *reason)
	case corpus.ClosedAsBoundary:
		ad, err = ledger.AsBoundary(d, *decision, *entry, *reason)
	default:
		return fmt.Errorf("unreachable: -as %q passed validation", *as)
	}
	awaiting := errors.Is(err, replay.ErrAwaitingHuman)
	if err != nil && !awaiting {
		return err
	}

	manifestPath := p.Path(project.ManifestFile)
	m, loadErr := corpus.Load(manifestPath)
	if loadErr != nil {
		return loadErr
	}
	if _, ok := m.Find(*input); !ok {
		return fmt.Errorf("%s is not in the manifest", *input)
	}
	if awaiting {
		replay.Propose(m, ad)
		if saveErr := m.Save(manifestPath); saveErr != nil {
			return saveErr
		}
		fmt.Fprintf(e.stdout, "proposed %s %s→%s as %s under %q; it waits on a human.\n\n%v\n\n%s\n",
			ad.Input, ad.From, ad.To, ad.ClosedAs, ad.Entry, err, awaitingHuman)
		return errAwaiting
	}
	if acceptErr := replay.Accept(m, ad); acceptErr != nil {
		return acceptErr
	}
	if saveErr := m.Save(manifestPath); saveErr != nil {
		return saveErr
	}
	fmt.Fprintf(e.stdout, "closed %s %s→%s as %s, citing %s\n",
		ad.Input, ad.From, ad.To, ad.ClosedAs, ad.Decision)
	if ad.Approval != "" {
		fmt.Fprintf(e.stdout, "the entry was approved: %s\n", ad.Approval)
	}
	return nil
}

// errAwaiting is the exit for work that is finished as far as it can go
// without a person. It is not a failure, and it is not a pass: a gate reads
// it as "stop and ask", which is the one honest move left.
var errAwaiting = errors.New("awaiting a human")

// awaitingHuman is what a session reads when the only thing left is a person.
// It names the move, because an agent told only "refused" goes looking for a
// way around the refusal.
const awaitingHuman = "Stop here and ask. Moving an observable into the boundary is a human " +
	"decision: a human reviews the decline, commits it, and re-runs this close. Do not commit " +
	"it yourself, and do not edit the corpus."

// lawNames is the set a law close may cite from the command line. The
// command cannot link the project's laws — they are code in the project's own
// packages, in whatever language it is written in — so this is a text check
// over the laws directory, not a link: it looks for the two shapes a law's
// name actually appears in in Go source, a call to one of pkg/law's six
// constructors or a hand-built law.Law{Name: "..."} literal (the second
// exists because a type a family can't take, like a map-backed value,
// still needs a law; see examples/querystring/laws/laws_test.go).
//
// Naming every identifier that appears anywhere under laws/ was tried first
// and was too permissive: it let a close cite allLaws, a helper function
// name, as if it were a registered law. A caller wanting the real check
// against a live set uses the Go API and passes replay.LawNamesOf(set).
func lawNames(p *project.Project) []string {
	var names []string
	seen := map[string]bool{}
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	root := p.Path(project.LawsDir)
	//nolint:gosec // walking the project's own laws directory, under its root
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // a project without a laws directory has no law names
		}
		b, err := os.ReadFile(path) //nolint:gosec // walking the project's own laws directory
		if err != nil {
			return nil //nolint:nilerr // an unreadable law file is not a reason to refuse a close
		}
		for _, m := range lawConstructorCall.FindAllStringSubmatch(string(b), -1) {
			family, name := m[1], m[2]
			if family == "Lens" {
				// pkg/law.Lens registers three laws under these exact
				// suffixes; it never registers one under the bare name.
				add(name + "GetPut")
				add(name + "PutGet")
				add(name + "PutPut")
				continue
			}
			add(name)
		}
		for _, m := range lawNameField.FindAllStringSubmatch(string(b), -1) {
			add(m[1])
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return names
}

// lawConstructorCall matches a call to one of pkg/law's six named
// constructors, capturing the family and the string literal passed as the
// law's name.
var lawConstructorCall = regexp.MustCompile(
	`\blaw\.(Roundtrip|Idempotent|Involution|Deterministic|Metamorphic|Lens)\(\s*"([^"]+)"`)

// lawNameField matches a hand-built law.Law{Name: "..."} literal's Name
// field.
var lawNameField = regexp.MustCompile(`\bName:\s*"([^"]+)"`)
