// Package germline is the shared vocabulary of the four artifacts: laws, a
// boundary, a corpus, and a decision record. It holds the types every check
// speaks in and nothing else, so that a package checking one artifact never
// has to import a package checking another.
//
// The artifacts themselves live in subpackages: pkg/boundary, pkg/corpus,
// pkg/law, pkg/replay and pkg/decide. The command in cmd/germline is a thin
// consumer of those, never a second implementation of them.
package germline

import (
	"fmt"
	"sort"
	"strings"
)

// Artifact names which of the four a problem was found in. A check reports
// against exactly one, so a reader knows which file to open.
type Artifact string

// The four durable artifacts, plus the config that binds them to a project.
const (
	Laws      Artifact = "laws"
	Boundary  Artifact = "boundary"
	Corpus    Artifact = "corpus"
	Decisions Artifact = "decisions"
	Config    Artifact = "config"
)

// A Problem is one thing a check found wrong, addressed to the person who has
// to fix it rather than to the program that found it. Where names the entry,
// the input id or the law; What says what to do about it.
type Problem struct {
	Artifact Artifact
	Where    string
	What     string
}

// String renders a problem as one terminal line.
func (p Problem) String() string {
	if p.Where == "" {
		return fmt.Sprintf("%s: %s", p.Artifact, p.What)
	}
	return fmt.Sprintf("%s %s: %s", p.Artifact, p.Where, p.What)
}

// Problems is what every check returns. The empty slice is the good case, so
// a check that finds nothing allocates nothing.
type Problems []Problem

// OK reports whether the check found nothing.
func (ps Problems) OK() bool { return len(ps) == 0 }

// String renders every problem, one per line, for a terminal or an error
// message. Problems is deliberately not an error: the empty case is the good
// case, and a type that is always non-nil and sometimes means nothing is the
// wrong shape for `if err != nil`. Test OK, then print this.
func (ps Problems) String() string {
	lines := make([]string, 0, len(ps))
	for _, p := range ps {
		lines = append(lines, p.String())
	}
	return strings.Join(lines, "\n")
}

// Sorted returns the problems in a stable order: by artifact, then by where,
// then by what. Checks may run concurrently and maps may iterate in any
// order; the output a person reads should not move between runs.
func (ps Problems) Sorted() Problems {
	out := make(Problems, len(ps))
	copy(out, ps)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Artifact != out[j].Artifact {
			return out[i].Artifact < out[j].Artifact
		}
		if out[i].Where != out[j].Where {
			return out[i].Where < out[j].Where
		}
		return out[i].What < out[j].What
	})
	return out
}

// In returns the problems belonging to one artifact.
func (ps Problems) In(a Artifact) Problems {
	var out Problems
	for _, p := range ps {
		if p.Artifact == a {
			out = append(out, p)
		}
	}
	return out
}
