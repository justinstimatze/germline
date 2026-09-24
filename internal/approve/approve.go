// Package approve reads whether a span of lines has been committed.
//
// Two decisions in the method stay human: moving something into the boundary
// and removing a law. A rule in prose does not keep them human. Measured on
// coding agents, an instruction not to touch the acceptance criteria leaves
// most of the tampering in place (docs/prior-art.md, "Who may relax a
// promise"). What helps is making the relaxation a separate step and giving
// the agent an honest way to stop and ask for it.
//
// The step here is a commit. A decline an agent has just written is not in
// one, so a close cannot cite it in the same breath; a human reviews the
// decline and commits it. An agent that can run git can commit too, so this
// is a visible step rather than a proof, and the refusal it produces says to
// stop and ask rather than to commit.
package approve

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrNotApproved reports a span that is not yet committed. The fix is a
// human, and never a change to the check.
var ErrNotApproved = errors.New("not approved by a human")

// timeout bounds each git call. Blame on one span takes milliseconds, so a
// call still running at the bound means a broken repository.
const timeout = 10 * time.Second

// Committed checks lines from..to of file and returns the commits that
// shaped them, or an error wrapping ErrNotApproved when any line is not yet
// in a commit, or the file's history cannot be read at all.
func Committed(ctx context.Context, file string, from, to int) ([]string, error) {
	if from < 1 || to < from {
		return nil, fmt.Errorf("approve: bad line range %d..%d in %s", from, to, file)
	}
	dir, base := filepath.Dir(file), filepath.Base(file)
	out, err := git(ctx, dir, "blame", "--porcelain", "-L",
		strconv.Itoa(from)+","+strconv.Itoa(to), "--", base)
	if err != nil {
		return nil, fmt.Errorf("%w: its history cannot be read (%v), and approval is read "+
			"from history", ErrNotApproved, err)
	}
	commits := shas(out)
	if len(commits) == 0 {
		return nil, fmt.Errorf("approve: git blame of %s:%d-%d returned no commits", file, from, to)
	}
	for _, c := range commits {
		if strings.Trim(c, "0") == "" {
			return nil, fmt.Errorf("%w: %s:%d-%d has lines no commit carries yet", ErrNotApproved,
				base, from, to)
		}
	}
	return commits, nil
}

// shas returns the distinct commits in porcelain blame output, in order of
// first appearance. A header line is a 40-hex sha followed by line numbers.
func shas(out string) []string {
	var (
		list []string
		seen = map[string]bool{}
	)
	for line := range strings.SplitSeq(out, "\n") {
		sha, _, ok := strings.Cut(line, " ")
		if !ok || len(sha) != 40 || strings.Trim(sha, "0123456789abcdef") != "" {
			continue
		}
		if !seen[sha] {
			seen[sha] = true
			list = append(list, sha)
		}
	}
	return list
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	//nolint:gosec // a fixed git subcommand; only the directory, range and path vary
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(cmd.Environ(), "LC_ALL=C")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
