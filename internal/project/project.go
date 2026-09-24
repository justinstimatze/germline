// Package project binds the four artifacts to a directory and runs every
// check that is a program.
//
// There is almost no configuration on purpose. The paths are conventions,
// the decision store is already recorded in git config, and the only thing a
// project has to declare is what its replay subjects are — which it cannot
// have until it has two versions to compare. A project that has not got there
// yet adopts germline by making a directory and writing a boundary entry.
package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/justinstimatze/germline"
	"github.com/justinstimatze/germline/internal/approve"
	"github.com/justinstimatze/germline/pkg/boundary"
	"github.com/justinstimatze/germline/pkg/corpus"
	"github.com/justinstimatze/germline/pkg/decide"
	"github.com/justinstimatze/germline/pkg/law"
)

// The conventional layout. A project may put these elsewhere, and then it has
// to say so; the point of naming them here is that most projects will not.
const (
	BoundaryFile = "BOUNDARY.md"
	ManifestFile = "corpus/manifest.json"
	InputsDir    = "corpus/inputs"
	LawsDir      = "laws"
	// DecisionsDir is a decision record a project ships with itself, checked
	// before the store the repository configures.
	DecisionsDir = "decisions"
)

// A Project is a directory holding the four artifacts.
type Project struct {
	// Root is the project directory.
	Root string
	// StoreDir is the decision record: a decisions directory the project ships
	// with itself if there is one, otherwise `git config winze.store`. Empty
	// means the project has no record on disk, which every check that resolves
	// a citation reports as unverified rather than failing on.
	StoreDir string
}

// Open finds a project at root and reads where its decision record lives.
func Open(root string) (*Project, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	return &Project{Root: abs, StoreDir: storeDir(abs)}, nil
}

// storeDir finds the decision record. A project that ships its own — a
// decisions directory in its root — is self-contained and needs no
// configuration to be checked, which is what makes a sample system runnable
// with no setup. Everything else reads the store the repository names.
func storeDir(root string) string {
	local := filepath.Join(root, DecisionsDir)
	if info, err := os.Stat(local); err == nil && info.IsDir() {
		return local
	}
	return gitConfig(root, "winze.store")
}

// Path resolves one of the conventional files against the project root.
func (p *Project) Path(rel string) string { return filepath.Join(p.Root, rel) }

// Store returns the decision record, or nil when none is configured.
func (p *Project) Store() decide.Store {
	if p.StoreDir == "" {
		return nil
	}
	return &decide.Winze{Dir: p.StoreDir}
}

// System is the name the decision record knows this project by: its directory
// name, so that a project is identified without configuring anything.
//
// A shared record can serve several projects, so a declined promise has to
// say whose it is. Renaming the directory therefore
// disconnects a project from its own declines, and the only defense built for
// that is that it fails loudly: with no declines under the new name, every
// entry in the file is unrecorded and the check says so entry by entry.
func (p *Project) System() string { return filepath.Base(p.Root) }

// Recorded returns the boundary entries the decision record carries for this
// system, and whether the record is the source for boundaries at all.
//
// The second result is the one that decides which artifact leads. A record
// carrying no declines for any system is one nobody has moved a boundary
// into, and the file stays the source with its citations the only thing
// checked. A record carrying some, for any system, makes an empty result for
// this system mean this project's entries went missing rather than were never
// written — which is a difference worth a red check.
func (p *Project) Recorded() (entries []boundary.Entry, sourced bool, err error) {
	store, ok := p.Store().(decide.Boundary)
	if !ok || store == nil {
		return nil, false, nil
	}
	declined, err := store.Declines()
	if err != nil {
		return nil, false, err
	}
	system := p.System()
	own := p.StoreDir == p.Path(DecisionsDir)
	for _, d := range declined {
		if d.System != system {
			// A shared record serves other systems too, so a decline for one
			// of them is someone else's. A record the project ships with
			// itself serves nobody else, and a decline naming another system
			// is this project's own entry under the wrong name: skipping it
			// would publish a boundary missing it and call the check green.
			if own {
				return nil, false, fmt.Errorf(
					"%s declines %q for system %q, but this project is %q (its directory name)",
					d.Decision, d.Observable, d.System, system)
			}
			continue
		}
		entries = append(entries, boundary.Entry{
			Observable: d.Observable,
			Since:      d.Since,
			Decision:   d.Decision,
			Reason:     d.Reason,
		})
	}
	return entries, len(declined) > 0, nil
}

// Approver returns what a close calls to learn whether a human approved the
// entry for an observable, and the evidence to record if one did: every line
// of the entry's source is committed.
//
// The source is the decline in the record when the record carries this
// system's declines, and the entry in BOUNDARY.md when the file is still the
// source.
func (p *Project) Approver(ctx context.Context) func(observable string) (string, error) {
	return func(observable string) (string, error) {
		file, from, to, err := p.entrySource(observable)
		if err != nil {
			return "", err
		}
		commits, err := approve.Committed(ctx, file, from, to)
		if err != nil {
			return "", err
		}
		short := make([]string, len(commits))
		for i, c := range commits {
			short[i] = c[:12]
		}
		return "committed " + strings.Join(short, ","), nil
	}
}

// entrySource locates the lines that declare an observable's entry.
func (p *Project) entrySource(observable string) (file string, from, to int, err error) {
	if store, ok := p.Store().(decide.Boundary); ok && store != nil {
		declined, derr := store.Declines()
		if derr != nil {
			return "", 0, 0, derr
		}
		for _, d := range declined {
			if d.System == p.System() && d.Observable == observable {
				return d.File, d.Line, d.EndLine, nil
			}
		}
		if len(declined) > 0 {
			return "", 0, 0, fmt.Errorf("the record carries no decline of %q for %s",
				observable, p.System())
		}
	}
	file = p.Path(BoundaryFile)
	body, err := os.ReadFile(file) //nolint:gosec // the project's own boundary file
	if err != nil {
		return "", 0, 0, err
	}
	lines := strings.Split(string(body), "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "- **"+observable+"**") {
			continue
		}
		end := i + 1
		for end < len(lines) && strings.HasPrefix(lines[end], "  ") {
			end++
		}
		return file, i + 1, end, nil
	}
	return "", 0, 0, fmt.Errorf("%s has no entry for %q", BoundaryFile, observable)
}

// Projected returns the boundary file the decision record implies, and what
// stands in the way. A nil document means the record is not the source and
// the file on disk is what there is.
func (p *Project) Projected(doc *boundary.Document) (*boundary.Document, germline.Problems) {
	recorded, sourced, err := p.Recorded()
	switch {
	case err != nil:
		return nil, germline.Problems{{
			Artifact: germline.Decisions, Where: p.StoreDir, What: err.Error(),
		}}
	case !sourced:
		return nil, nil
	}
	projected, ps := doc.Project(recorded)
	if len(ps) > 0 {
		return nil, ps
	}
	return projected, nil
}

// checkBoundaryRecord reports the file and the record disagreeing. Being out
// of date is a problem and not a formatting nit: what a project publishes is
// the file, so a decline recorded and not projected has not been declared to
// anybody.
func (p *Project) checkBoundaryRecord(doc *boundary.Document) germline.Problems {
	projected, ps := p.Projected(doc)
	if len(ps) > 0 || projected == nil {
		return ps
	}
	if projected.String() == doc.String() {
		return nil
	}
	return germline.Problems{{
		Artifact: germline.Boundary, Where: BoundaryFile,
		What: "does not match the decision record it is generated from; " +
			"run: germline boundary -write",
	}}
}

// Check runs every check that is a program and returns what they found. An
// empty result is the good case; the caller decides what stops a build.
//
// It deliberately does not run the laws. Those are Go tests in the project's
// own package and `go test` runs them; a checker that re-ran them here would
// be a second, drifting way to do the same thing.
func (p *Project) Check() germline.Problems {
	ps, _ := p.check()
	return ps
}

// check is Check, plus the number of citations it could not resolve because
// no record is configured. Status wants both and Check wants only the first,
// and gathering the citations twice would be a second way to decide what
// counts as one.
func (p *Project) check() (ps germline.Problems, unverified int) {
	cited := make([]string, 0, 16)

	doc, err := boundary.Load(p.Path(BoundaryFile))
	if err != nil {
		ps = append(ps, germline.Problem{
			Artifact: germline.Boundary, Where: BoundaryFile, What: err.Error(),
		})
	} else {
		ps = append(ps, doc.Validate()...)
		ps = append(ps, p.checkBoundaryRecord(doc)...)
		cited = append(cited, doc.Decisions()...)
	}

	m, err := corpus.Load(p.Path(ManifestFile))
	if err != nil {
		ps = append(ps, germline.Problem{
			Artifact: germline.Corpus, Where: ManifestFile, What: err.Error(),
		})
	} else {
		ps = append(ps, m.Validate()...)
		ps = append(ps, p.checkAppendOnly(m)...)
		ps = append(ps, p.checkInputsPresent(m)...)
		for i := range m.AcceptedDiffs {
			cited = append(cited, m.AcceptedDiffs[i].Decision)
		}
		for _, in := range m.Inputs {
			if in.Retired != nil {
				cited = append(cited, in.Retired.Decision)
			}
		}
	}

	cps, unresolved := p.checkCitations(cited)
	return append(ps, cps...), unresolved
}

// checkCitations resolves every decision the artifacts cite against the
// record, and returns how many it could not resolve for want of one. A
// citation nobody verifies is a citation to nothing, and these are the two
// artifacts where being wrong is cheapest to hide.
//
// No record configured is not a problem. It used to be, and that made a fresh
// clone of a project whose record is private fail its own check — which is
// every project that keeps the record outside the repository, including this
// one, and so also every CI runner, which is a fresh clone by definition. The
// check now says how many citations went unverified and passes. A record
// configured and not there is still a problem: that is ErrNoStore, and the
// fix for it is a path rather than a decision.
//
// What this gives up is that a project checked without a record can carry a
// citation to a decision nobody wrote. It is given up on purpose: the strict
// check still runs wherever the record is, which is the machine where the
// citation is written in the first place.
func (p *Project) checkCitations(cited []string) (ps germline.Problems, unverified int) {
	store := p.Store()
	if store == nil {
		return nil, len(cited)
	}
	missing, err := decide.Missing(store, cited)
	if err != nil {
		return germline.Problems{{
			Artifact: germline.Decisions, Where: p.StoreDir, What: err.Error(),
		}}, 0
	}
	ps = make(germline.Problems, 0, len(missing))
	for _, name := range missing {
		ps = append(ps, germline.Problem{
			Artifact: germline.Decisions, Where: name,
			What: "cited by an artifact but not in the record; record the decision " +
				"with its reason, its alternatives, and what would reverse it",
		})
	}
	return ps, 0
}

// checkAppendOnly compares the working manifest against the one committed at
// HEAD. Enforcing the corpus rule against version control rather than against
// politeness is the difference between a rule and a request.
func (p *Project) checkAppendOnly(now *corpus.Manifest) germline.Problems {
	was, err := p.manifestAt("HEAD")
	if err != nil {
		if errors.Is(err, errNotTracked) || errors.Is(err, errNoGit) {
			return nil // nothing committed yet; there is nothing to append to
		}
		return germline.Problems{{
			Artifact: germline.Corpus, Where: ManifestFile, What: err.Error(),
		}}
	}
	return corpus.VerifyAppendOnly(was, now)
}

var (
	errNoGit      = errors.New("not a git repository")
	errNotTracked = errors.New("manifest is not tracked at that revision")
)

// gitTimeout bounds the two git reads. Both are local and instant; the
// deadline is here so a wedged git cannot hang a build gate forever.
const gitTimeout = 15 * time.Second

// manifestAt reads the manifest as it stood at a git revision.
//
// The path is written `./corpus/manifest.json` because git resolves
// `rev:path` from the top of the repository, not from `-C`: a project in a
// subdirectory would otherwise be compared against whatever manifest sits at
// the repository root, and pass.
//
// Only the two answers that mean "nothing committed to compare against" are
// errNotTracked. Every other git failure — a timeout, a repository git
// refuses to read, a corrupt object — is reported, because treating it as
// untracked turns a check that could not run into a check that passed.
func (p *Project) manifestAt(rev string) (*corpus.Manifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	//nolint:gosec // fixed argv; only the revision and the project root vary
	cmd := exec.CommandContext(ctx, "git", "-C", p.Root, "show", rev+":./"+ManifestFile)
	cmd.Env = append(os.Environ(), "LC_ALL=C") // the stderr below is matched as text
	out, err := cmd.Output()
	if err == nil {
		return corpus.ParseManifest(out)
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("reading the manifest at %s: %w", rev, ctx.Err())
		}
		return nil, errNoGit // git itself is not installed
	}
	stderr := string(ee.Stderr)
	switch {
	case strings.Contains(stderr, "not a git repository"):
		return nil, errNoGit
	case strings.Contains(stderr, "does not exist in"),
		strings.Contains(stderr, "exists on disk, but not in"),
		strings.Contains(stderr, "invalid object name"): // no commits yet
		return nil, errNotTracked
	}
	return nil, fmt.Errorf("reading the manifest at %s: git: %s", rev, strings.TrimSpace(stderr))
}

// checkInputsPresent reports active inputs whose recorded data is missing
// from a corpus that has data on this machine. A replay cannot run an input
// it cannot read, so a lost file is a lost recording.
//
// A corpus with no data here at all is not checked. Recorded data usually
// lives outside the repository, and a clone or a CI runner that never had
// it is not one that lost it.
func (p *Project) checkInputsPresent(m *corpus.Manifest) germline.Problems {
	dir := p.Path(InputsDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	have := make(map[string]bool, len(entries))
	for _, e := range entries {
		if !e.IsDir() && e.Name() != ".gitkeep" {
			have[e.Name()] = true
		}
	}
	if len(have) == 0 {
		return nil
	}
	var ps germline.Problems
	for _, in := range m.Inputs {
		if in.Status == corpus.Active && !have[in.SHA256] {
			ps = append(ps, germline.Problem{
				Artifact: germline.Corpus, Where: in.ID,
				What: "is active and its recorded data is not in " + InputsDir +
					"; restore it from wherever the recording is kept, or retire the input with a decision",
			})
		}
	}
	return ps
}

// A Status is one pass over the four artifacts: what each holds, and what the
// checks found. It is what a terminal prints and what a gate reads.
type Status struct {
	Root            string
	StoreDir        string
	BoundaryEntries int
	CorpusInputs    int
	CorpusActive    int
	CorpusWeight    float64
	Decisions       int
	Unchecked       int
	// Unverified counts the citations no record was configured to resolve.
	// Zero where a record is configured, whether or not it carried them.
	Unverified int
	Problems   germline.Problems
}

// OK reports whether every check passed.
func (s Status) OK() bool { return s.Problems.OK() }

// Status runs every check and counts what the artifacts hold.
func (p *Project) Status() Status {
	ps, unverified := p.check()
	s := Status{
		Root: p.Root, StoreDir: p.StoreDir,
		Unverified: unverified, Problems: ps.Sorted(),
	}
	if doc, err := boundary.Load(p.Path(BoundaryFile)); err == nil {
		s.BoundaryEntries = len(doc.Entries)
	}
	if m, err := corpus.Load(p.Path(ManifestFile)); err == nil {
		s.CorpusInputs = len(m.Inputs)
		s.CorpusWeight = m.ActiveWeight()
		for _, in := range m.Inputs {
			if in.Status == corpus.Active {
				s.CorpusActive++
			}
		}
	}
	if store := p.Store(); store != nil {
		if names, err := store.Names(); err == nil {
			s.Decisions = len(names)
		}
	}
	// A scan that could not finish is a problem, never a count of zero: zero
	// is a plausible number, and a metric that reads zero because it looked
	// at nothing passes every eye that reads it.
	if prose, err := p.Unchecked(); err == nil {
		s.Unchecked = len(prose)
	} else {
		s.Problems = append(s.Problems, germline.Problem{
			Artifact: germline.Laws, Where: p.Root,
			What: "could not scan for prose laws: " + err.Error(),
		}).Sorted()
	}
	return s
}

// Unchecked returns the laws written as prose and not yet made executable.
// Not a problem on its own: prose laws are legitimate, and losing them is the
// failure worth preventing.
func (p *Project) Unchecked() ([]law.Prose, error) { return law.Unchecked(p.Root) }

// gitConfig reads one git config value, or "" when git or the key is absent.
func gitConfig(dir, key string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	//nolint:gosec // fixed argv; only the directory and the key vary
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "config", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
