// Package boundary reads, writes and checks the enumerated list of
// observables a system declines to promise.
//
// The list is the complement of the suite's distinguishing power: two
// implementations that pass every law and every replay may differ on exactly
// what is listed here, and on nothing else. Keeping it in markdown keeps it
// readable by the human who answers for the system. Parsing it keeps it
// checkable by a program, which is the only reason the drift gauge can be
// trusted.
package boundary

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/justinstimatze/germline"
)

// Width is the column the renderer wraps entries at. It matches the prose
// around them so a rendered file and a hand-written one are the same file.
const Width = 76

// Heading is the line the entry list follows. Everything before it is
// preamble and everything after the list is epilogue; both are preserved
// verbatim so the document can carry as much prose as its author wants.
const Heading = "## Entries"

// ErrNoEntries reports a document with no entry list to parse. A boundary
// with nothing on it is written as the heading and no items, not as a file
// missing the heading, because the missing heading is usually a typo.
var ErrNoEntries = errors.New("no " + Heading + " section")

// An Entry is one declined promise: something a user could observe that the
// system does not commit to.
type Entry struct {
	// Observable is what someone could see. Stated as the observation, not
	// as the implementation that produces it.
	Observable string
	// Since is the version the entry was declared at.
	Since string
	// Decision names the memory in the decision record that carries the
	// reason, the alternatives, and what would reverse it.
	Decision string
	// Reason is one line of that memory, so a reader deciding whether a
	// replay difference is licensed does not have to open the store.
	Reason string
}

// A Document is a parsed boundary file: the prose around the list, and the
// list. Round-tripping preserves the prose, so a program may add an entry to
// a file a person wrote without flattening it.
type Document struct {
	Preamble []string
	Entries  []Entry
	Epilogue []string
}

// entryPattern matches one rendered entry after its leading "- " and after
// continuation lines have been joined. The em dash is the field separator
// because it does not occur inside an observable or a version.
var entryPattern = regexp.MustCompile(
	"^\\*\\*(.+?)\\*\\* — ([^,]+), `([^`]+)`\\. (.+)$")

// Load parses the boundary document at path.
func Load(path string) (*Document, error) {
	f, err := os.Open(path) //nolint:gosec // the path is the project's own boundary file
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only file; a close error cannot lose data
	d, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return d, nil
}

// Parse reads a boundary document. Lines before the entry heading and after
// the last entry are kept verbatim; the entries themselves are normalized, so
// that Render produces the same bytes whoever wrote them.
func Parse(r io.Reader) (*Document, error) {
	var lines []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == Heading {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil, ErrNoEntries
	}

	items, first, last := collectItems(lines, start)
	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		e, err := parseEntry(item)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if first < 0 {
		// An empty list sits under its heading, which ends where the next
		// heading starts; without one, it runs to the end of the file.
		first = len(lines)
		for i := start; i < len(lines); i++ {
			if strings.HasPrefix(lines[i], "#") {
				first = i
				break
			}
		}
		last = first
	}
	return &Document{
		Preamble: lines[:first],
		Entries:  entries,
		Epilogue: lines[last:],
	}, nil
}

// collectItems gathers the list items following the heading. It returns each
// item joined into one line, and the half-open line range the list occupied,
// so the prose on either side survives a round trip.
func collectItems(lines []string, start int) (items []string, first, last int) {
	first, last = -1, -1
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			items = append(items, cur.String())
			cur.Reset()
		}
	}
	for i := start; i < len(lines); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "- "):
			flush()
			cur.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			if first < 0 {
				first = i
			}
			last = i + 1
		case cur.Len() > 0 && strings.HasPrefix(line, "  ") && strings.TrimSpace(line) != "":
			cur.WriteString(" ")
			cur.WriteString(strings.TrimSpace(line))
			last = i + 1
		case cur.Len() > 0:
			// A blank line or unindented prose ends the list.
			flush()
			return items, first, last
		}
	}
	flush()
	return items, first, last
}

func parseEntry(item string) (Entry, error) {
	m := entryPattern.FindStringSubmatch(item)
	if m == nil {
		return Entry{}, fmt.Errorf("boundary entry is not in the form "+
			"`**observable** — version, `+\"`decision`\"+`. reason`: %q", item)
	}
	return Entry{
		Observable: m[1],
		Since:      m[2],
		Decision:   m[3],
		Reason:     m[4],
	}, nil
}

// Render writes the document back out. Render(Parse(x)) is x for any
// document this package produced, which is the law the projection rests on:
// a program may rewrite the file without a person losing what they wrote.
func (d *Document) Render(w io.Writer) error {
	bw := bufio.NewWriter(w)
	for _, line := range d.Preamble {
		fmt.Fprintln(bw, line)
	}
	for _, e := range d.Entries {
		for _, line := range wrap(e.line(), Width) {
			fmt.Fprintln(bw, line)
		}
	}
	// A list ends at a blank line. An epilogue that starts on text, which is
	// what follows a list that was empty when parsed, needs one supplied.
	if len(d.Entries) > 0 && len(d.Epilogue) > 0 && d.Epilogue[0] != "" {
		fmt.Fprintln(bw)
	}
	for _, line := range d.Epilogue {
		fmt.Fprintln(bw, line)
	}
	return bw.Flush()
}

// String renders the document to a string.
func (d *Document) String() string {
	var sb strings.Builder
	_ = d.Render(&sb) //nolint:errcheck // strings.Builder writes never fail
	return sb.String()
}

// line is the entry as one unwrapped markdown list item.
func (e Entry) line() string {
	return fmt.Sprintf("- **%s** — %s, %#q. %s",
		e.Observable, e.Since, e.Decision, e.Reason)
}

// Find returns the entry whose observable matches, and whether one did.
func (d *Document) Find(observable string) (Entry, bool) {
	for _, e := range d.Entries {
		if e.Observable == observable {
			return e, true
		}
	}
	return Entry{}, false
}

// Append adds an entry. Boundary entries are appended and never edited in
// place: withdrawing a promise is a new entry, and making one is a removal in
// a major version. Appending a duplicate observable is refused, because two
// entries for one observable make the drift gauge count it twice.
func (d *Document) Append(e Entry) error {
	if _, ok := d.Find(e.Observable); ok {
		return fmt.Errorf("boundary already declines %q; a second entry would "+
			"double-count it in the drift gauge", e.Observable)
	}
	d.Entries = append(d.Entries, e)
	return nil
}

// Project returns the document a decision record implies: the same prose,
// and the recorded entries in the order the record declares them.
//
// The record is the source and the file is what gets published. A projection
// like this can lose something: a boundary regenerated from a record that has
// forgotten an entry is a promise nobody meant to make, and the drift gauge
// reads the loss as progress.
//
// So the projection refuses. An entry the record does not carry is reported
// and the document comes back unchanged, with nothing written. The reader has
// two ways forward and the refusal names neither, because they are not the
// same decision: record the reason, or decide the observable was a promise
// after all and remove the entry in a major version.
//
// Entries the record carries and the document does not are appended, in
// record order. That is how a decline reaches the file at all.
func (d *Document) Project(recorded []Entry) (*Document, germline.Problems) {
	have := make(map[string]bool, len(recorded))
	for _, e := range recorded {
		have[e.Observable] = true
	}
	var ps germline.Problems
	for _, e := range d.Entries {
		if have[e.Observable] {
			continue
		}
		ps = append(ps, germline.Problem{
			Artifact: germline.Boundary,
			Where:    e.Observable,
			What: "is declined in the file and carries no decline in the record; " +
				"record the reason, or make it a promise in a major version",
		})
	}
	if len(ps) > 0 {
		return d, ps
	}
	return &Document{
		Preamble: d.Preamble,
		Entries:  append([]Entry(nil), recorded...),
		Epilogue: d.Epilogue,
	}, nil
}

// Decisions returns the decision each entry cites, in document order, so a
// caller can resolve them all against the record in one pass.
func (d *Document) Decisions() []string {
	out := make([]string, 0, len(d.Entries))
	for _, e := range d.Entries {
		out = append(out, e.Decision)
	}
	return out
}

// Validate checks the document against itself: every entry states an
// observable, a version, a decision and a reason, and no observable is
// declined twice. It does not check that the decision exists; that needs the
// record, and the project package does it.
func (d *Document) Validate() germline.Problems {
	var ps germline.Problems
	add := func(where, what string) {
		ps = append(ps, germline.Problem{Artifact: germline.Boundary, Where: where, What: what})
	}
	seen := make(map[string]bool, len(d.Entries))
	for _, e := range d.Entries {
		where := e.Observable
		if where == "" {
			where = "(unnamed entry)"
			add(where, "states no observable")
		}
		if seen[e.Observable] {
			add(where, "is declined twice; a duplicate double-counts the drift gauge")
		}
		seen[e.Observable] = true
		if e.Since == "" {
			add(where, "names no version; the boundary is published before dependents exist")
		}
		if e.Decision == "" {
			add(where, "cites no decision; an entry without a reason is a place "+
				"someone licensed themselves to be wrong")
		}
		if e.Reason == "" {
			add(where, "gives no reason")
		}
	}
	return ps
}

// wrap greedily fills lines to width, hanging-indenting continuations by two
// spaces so the rendered list reads as a list.
func wrap(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, len(words)/8+1)
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = "  " + w
			continue
		}
		cur += " " + w
	}
	return append(lines, cur)
}
