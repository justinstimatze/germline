package law

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// A law with no executable form yet is written as a comment, prefixed so it
// can be found. The prefix is the whole mechanism: prose laws are legitimate
// and common, and the failure to prevent is not writing them but losing them.

// Marker prefixes a law that has been stated but not yet made executable.
const Marker = "LAW (unchecked):"

// markerLine matches the marker only where a comment opens: nothing but
// whitespace and comment punctuation may precede it. Without that anchor the
// scanner matches every sentence that mentions the marker, this file's own
// declaration of it included, and a scanner that finds four laws in its own
// documentation is worse than no scanner.
var markerLine = regexp.MustCompile(`^[\s/*#;%!<>-]*` + regexp.QuoteMeta(Marker))

// Prose is one law stated in a comment and not yet checked by a program.
type Prose struct {
	File      string
	Line      int
	Statement string
}

// skipDir names directories no scan should descend into. Recorded corpus data
// is excluded because a recording may contain the marker as data.
var skipDir = map[string]bool{
	".git": true, "node_modules": true, "vendor": true,
	"bin": true, "dist": true, "build": true, "inputs": true,
}

// Unchecked walks root and returns every prose law, sorted by file and line.
// It reads text files only; a file with a NUL byte early on is treated as
// binary and skipped.
func Unchecked(root string) ([]Prose, error) {
	var found []Prose
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skipDir[d.Name()] || (strings.HasPrefix(d.Name(), ".") && path != root) {
				return fs.SkipDir
			}
			return nil
		}
		in, err := scanFile(path)
		if err != nil {
			return err
		}
		found = append(found, in...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(found, func(i, j int) bool {
		if found[i].File != found[j].File {
			return found[i].File < found[j].File
		}
		return found[i].Line < found[j].Line
	})
	return found, nil
}

func scanFile(path string) ([]Prose, error) {
	f, err := os.Open(path) //nolint:gosec // walking a project directory the caller named
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only file; a close error cannot lose data

	head := make([]byte, 8192)
	n, err := f.Read(head)
	if err != nil && n == 0 {
		return nil, nil // empty or unreadable; nothing to find
	}
	if bytes.IndexByte(head[:n], 0) >= 0 {
		return nil, nil // binary
	}
	if _, seekErr := f.Seek(0, 0); seekErr != nil {
		return nil, seekErr
	}

	// ReadLine rather than a Scanner: a Scanner stops the whole file at the
	// first line longer than its buffer, and a minified or generated file in
	// the tree then hid every law after it and, through the walk, every law in
	// the project. A marker sits at the start of a line, so the first
	// fragment of an overlong line is all that needs reading.
	var found []Prose
	r := bufio.NewReaderSize(f, 64*1024)
	for line := 1; ; line++ {
		text, isPrefix, readErr := r.ReadLine()
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return nil, readErr
		}
		if prefix := markerLine.FindString(string(text)); prefix != "" {
			statement := strings.TrimSpace(string(text[len(prefix):]))
			found = append(found, Prose{File: path, Line: line, Statement: statement})
		}
		for isPrefix {
			if _, isPrefix, readErr = r.ReadLine(); readErr != nil {
				if errors.Is(readErr, io.EOF) {
					return found, nil
				}
				return nil, readErr
			}
		}
	}
	return found, nil
}
