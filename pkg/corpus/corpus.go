// Package corpus reads, writes and checks the manifest: the record of what
// users did, what each version did with it, and which differences between
// versions were accepted and why.
//
// Two things with different lifetimes live here on purpose. Inputs are
// recorded from real use and are append-only; their lifetime is the
// product's. Outputs are re-derived per version and are never edited; the
// thing anyone reviews is the diff between two versions on the same input.
// The rules in appendonly.go are the executable form of that sentence.
package corpus

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/justinstimatze/germline"
	"github.com/justinstimatze/germline/internal/atomicfile"
)

// Schema is the manifest format this package reads and writes. It is stored
// in the file so that a future format can be recognized rather than guessed.
const Schema = "germline.corpus.manifest/0"

// Level is the confidentiality an input is stored at, strongest first.
// Fidelity falls as the number rises; pick per component, not per project.
type Level int

const (
	// LevelDoNotStore keeps no sessions at all: shadow traffic to old and
	// new, diff, keep only mismatches. You can replay only what is live now.
	LevelDoNotStore Level = 1
	// LevelProfileOnly keeps which operations happen, how often, and in what
	// sequences, and generates inputs from that. Loses every bug that
	// depends on concrete data.
	LevelProfileOnly Level = 2
	// LevelAbstraction keeps the same abstraction the laws use: for an
	// editor, the command stream and a synthetic buffer with the real one's
	// structure and none of its text.
	LevelAbstraction Level = 3
	// LevelConcreteExternal keeps concrete data outside the repository,
	// access-controlled, with only hashes in the manifest; replay runs where
	// the data is.
	LevelConcreteExternal Level = 4
)

// Status is where an input is in its lifecycle: recorded, active, retired.
type Status string

// The two statuses an input may hold in the manifest. Retirement is an
// appended event and never a deletion, so a retired input stays visible and
// can never pass for a wrong reason.
const (
	Active  Status = "active"
	Retired Status = "retired"
)

// Close is one of exactly three ways a replay difference may be closed. There
// is no fourth, and editing the corpus is not one of them. Every close is a
// decision and names the memory that carries its reason.
type Close string

const (
	// ClosedAsImplementation means the implementation was wrong. Fix it.
	ClosedAsImplementation Close = "implementation"
	// ClosedAsLaw means the law was missing a condition. Add the condition to
	// the law, so the next regeneration cannot lose it again.
	ClosedAsLaw Close = "law"
	// ClosedAsBoundary means the behavior was never a promise. Move it to the
	// boundary with a reason, and the entry must already exist there.
	ClosedAsBoundary Close = "boundary"
)

// Valid reports whether c is one of the three.
func (c Close) Valid() bool {
	switch c {
	case ClosedAsImplementation, ClosedAsLaw, ClosedAsBoundary:
		return true
	default:
		return false
	}
}

// Closes lists the three, in the order the method states them.
func Closes() []Close {
	return []Close{ClosedAsImplementation, ClosedAsLaw, ClosedAsBoundary}
}

// Profile is how the operational profile was estimated. Weights are
// re-estimated as traffic moves, which is the one field of an input that may
// change after recording.
type Profile struct {
	EstimatedAt         string `json:"estimated_at"`
	WindowDays          int    `json:"window_days"`
	RecencyHalfLifeDays int    `json:"recency_half_life_days"`
}

// Retirement is the appended event that retires an input: a feature removed,
// a retention policy expired, or an erasure request. All three are the same
// operation and all three name a decision.
type Retirement struct {
	AtVersion string `json:"at_version"`
	Reason    string `json:"reason"`
	Decision  string `json:"decision"`
}

// An Input is one thing a user did, recorded and never authored.
type Input struct {
	ID         string `json:"id"`
	RecordedAt string `json:"recorded_at"`
	SHA256     string `json:"sha256"`
	Level      Level  `json:"level"`
	// Weight is this input's share of the operational profile. The only
	// field that may move after recording, because the profile moves.
	Weight  float64     `json:"weight"`
	Status  Status      `json:"status"`
	Retired *Retirement `json:"retired,omitempty"`
	// Outputs maps a version to the hash of what that version produced.
	// Written once per version and never edited.
	Outputs map[string]string `json:"outputs,omitempty"`
}

// An AcceptedDiff records a difference between two versions on one input and
// which of the three closes discharged it.
type AcceptedDiff struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Input    string `json:"input"`
	ClosedAs Close  `json:"closed_as"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
	// Entry is the boundary entry a boundary close cites, by observable.
	Entry string `json:"entry,omitempty"`
	// Approval says what showed a human approved that entry: the policy and
	// the commits, as in "committed 1a2b3c4d5e6f". A reader of the manifest
	// sees which closes rest on a signature and which on a commit alone.
	Approval string `json:"approval,omitempty"`
}

// A Manifest is the tracked half of the corpus. The recorded data itself
// lives wherever the confidentiality level says; only hashes are here.
type Manifest struct {
	SchemaName    string         `json:"schema"`
	Profile       Profile        `json:"profile"`
	Inputs        []Input        `json:"inputs"`
	AcceptedDiffs []AcceptedDiff `json:"accepted_diffs,omitempty"`
	// ProposedCloses are boundary closes waiting on a human to approve the
	// entry they cite. Unlike an accepted diff, a proposal is not a record: it
	// is removed when the close it proposes is accepted, and nothing in the
	// append-only check reads it.
	ProposedCloses []AcceptedDiff `json:"proposed_closes,omitempty"`
}

// New returns an empty manifest with today's profile window.
func New(estimatedAt string) *Manifest {
	return &Manifest{
		SchemaName: Schema,
		Profile: Profile{
			EstimatedAt:         estimatedAt,
			WindowDays:          90,
			RecencyHalfLifeDays: 30,
		},
	}
}

// Load reads a manifest. A missing file is not an error to this package: a
// project with nothing recorded yet has an empty corpus, not a broken one.
// Callers that need the distinction should stat the path themselves.
func Load(path string) (*Manifest, error) {
	b, err := os.ReadFile(path) //nolint:gosec // the path is the project's own manifest
	if err != nil {
		if os.IsNotExist(err) {
			return New(time.Now().UTC().Format(time.DateOnly)), nil
		}
		return nil, err
	}
	m, err := ParseManifest(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return m, nil
}

// ParseManifest reads a manifest from bytes, for a caller holding the content
// rather than the path: a manifest as it stood at a git revision, say.
func ParseManifest(b []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Save writes the manifest with a stable layout: inputs and accepted diffs
// sorted by id so a re-save produces no diff. Readers must not depend on that
// order — the boundary declines to promise it — but a writer that shuffles
// its own file makes every review noisier for no gain.
func (m *Manifest) Save(path string) error {
	out := *m
	out.Inputs = append([]Input(nil), m.Inputs...)
	sort.SliceStable(out.Inputs, func(i, j int) bool {
		return out.Inputs[i].ID < out.Inputs[j].ID
	})
	b, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(path, append(b, '\n'))
}

// Find returns the input with this id, and whether one existed.
func (m *Manifest) Find(id string) (Input, bool) {
	for _, in := range m.Inputs {
		if in.ID == id {
			return in, true
		}
	}
	return Input{}, false
}

// ActiveWeight is the share of the operational profile the active inputs
// carry. It is corpus coverage when the weights are an honest estimate of the
// profile, and it is the number that says whether a green replay means much.
func (m *Manifest) ActiveWeight() float64 {
	var total float64
	for _, in := range m.Inputs {
		if in.Status == Active {
			total += in.Weight
		}
	}
	return total
}

// Closes counts the accepted diffs by which of the three closed them. If the
// closes keep landing in one column, that column is a dumping ground: all
// boundary means the corpus is decaying into the profile, all implementation
// means the laws are not learning anything.
func (m *Manifest) Closes() map[Close]int {
	counts := make(map[Close]int, len(Closes()))
	for _, c := range Closes() {
		counts[c] = 0
	}
	for i := range m.AcceptedDiffs {
		counts[m.AcceptedDiffs[i].ClosedAs]++
	}
	return counts
}

var (
	hash64  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	hashRef = regexp.MustCompile(`^(sha256:)?[0-9a-f]{64}$`)
)

// Validate checks the manifest against itself: ids unique, hashes well
// formed, levels and statuses in range, retirements complete, and every
// accepted diff closed one of three ways against an input that exists.
//
// It deliberately says nothing about order. Two manifests differing only in
// the order of their inputs are the same manifest, which is a promise the
// boundary declines to make and this function must not quietly make instead.
func (m *Manifest) Validate() germline.Problems {
	var ps germline.Problems
	add := func(where, what string) {
		ps = append(ps, germline.Problem{Artifact: germline.Corpus, Where: where, What: what})
	}
	if m.SchemaName != Schema {
		add("schema", fmt.Sprintf("is %q, want %q", m.SchemaName, Schema))
	}
	seen := make(map[string]bool, len(m.Inputs))
	for _, in := range m.Inputs {
		validateInput(in, seen, add)
		seen[in.ID] = true
	}
	// Weights are shares of one profile. A total past one is a profile that
	// was estimated twice, or not at all, and it makes coverage read as
	// better than complete. The tolerance is for float sums of many shares.
	if total := m.ActiveWeight(); total > 1+1e-6 {
		add("profile", fmt.Sprintf("active inputs carry %.3f of it; the shares of one "+
			"profile cannot total more than 1 (germline reweight re-estimates them)", total))
	}
	for i := range m.AcceptedDiffs {
		d := &m.AcceptedDiffs[i]
		where := fmt.Sprintf("diff %d (%s %s→%s)", i, d.Input, d.From, d.To)
		if !d.ClosedAs.Valid() {
			add(where, fmt.Sprintf("closed as %q; the only three closes are %v",
				d.ClosedAs, Closes()))
		}
		if d.Decision == "" {
			add(where, "names no decision; accepting a diff is a decision and is recorded")
		}
		if d.Reason == "" {
			add(where, "has no reason")
		}
		if !seen[d.Input] {
			add(where, "refers to an input that is not in the manifest")
		}
	}
	return ps
}

func validateInput(in Input, seen map[string]bool, add func(where, what string)) {
	where := in.ID
	if where == "" {
		where = "(unnamed input)"
		add(where, "has no id")
	}
	if seen[in.ID] {
		add(where, "id appears twice; ids identify a recording for its lifetime")
	}
	if !hash64.MatchString(in.SHA256) {
		add(where, fmt.Sprintf("sha256 %q is not 64 lowercase hex characters", in.SHA256))
	}
	if in.Level < LevelDoNotStore || in.Level > LevelConcreteExternal {
		add(where, fmt.Sprintf("confidentiality level %d is not one of 1..4", in.Level))
	}
	if in.Weight < 0 || in.Weight > 1 {
		add(where, fmt.Sprintf("weight %v is not a share in [0,1]", in.Weight))
	}
	if in.RecordedAt == "" {
		add(where, "has no recorded_at")
	}
	switch in.Status {
	case Active:
		if in.Retired != nil {
			add(where, "is active and also carries a retirement event")
		}
	case Retired:
		validateRetirement(in, where, add)
	default:
		add(where, fmt.Sprintf("status %q is neither %q nor %q", in.Status, Active, Retired))
	}
	for version, out := range in.Outputs {
		if !hashRef.MatchString(out) {
			add(where, fmt.Sprintf("output for version %s is %q, not a sha256", version, out))
		}
	}
}

func validateRetirement(in Input, where string, add func(where, what string)) {
	if in.Retired == nil {
		add(where, "is retired with no retirement event; retirement is an appended event")
		return
	}
	if in.Retired.Reason == "" {
		add(where, "is retired with no reason")
	}
	if in.Retired.Decision == "" {
		add(where, "is retired with no decision; retiring an input is a decision")
	}
	if in.Retired.AtVersion == "" {
		add(where, "is retired with no version")
	}
}
