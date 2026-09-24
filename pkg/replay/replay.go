// Package replay runs the recorded corpus against two versions and reports
// where they disagree.
//
// The thing anyone reviews is never a corpus entry. It is the diff between
// version N and N+1 on the same inputs, and every diff is an obligation that
// close.go will not let you discharge without evidence.
//
// Subject is the port. Command is the portable implementation: a version of
// the system is a program that reads an input on stdin and writes its output
// on stdout, which a C binary, a Python service and a Go package can all be.
package replay

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/justinstimatze/germline/pkg/corpus"
)

// A Subject is one version of the system under replay. Run must be a
// function of its input alone: a subject that reads the clock or the network
// makes every difference unattributable.
type Subject interface {
	// Version is the version string the outputs are recorded under.
	Version() string
	// Run produces this version's output for one recorded input.
	Run(ctx context.Context, input []byte) ([]byte, error)
}

// An InputStore fetches a recorded input by its content hash. Where it reads
// from is the confidentiality level's business: a directory in the repository
// for level 3, an access-controlled store elsewhere for level 4, and a
// generator from the profile for level 2.
type InputStore interface {
	Get(hash string) ([]byte, error)
}

// Dir is an InputStore reading content-addressed files from a directory, one
// file per input, named by its hash.
type Dir string

// Get reads the input with this hash.
func (d Dir) Get(hash string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(string(d), hash)) //nolint:gosec // content-addressed name
	if err != nil {
		return nil, fmt.Errorf("input %s: %w", hash[:min(12, len(hash))], err)
	}
	if got := Hash(b); got != hash {
		return nil, fmt.Errorf("input %s hashes to %s; the recording has been altered",
			hash[:min(12, len(hash))], got[:12])
	}
	return b, nil
}

// Hash is the content hash the manifest stores, as 64 lowercase hex.
func Hash(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Command is a Subject that runs an external program: the input on stdin, the
// output on stdout, a non-zero exit an error. This is the seam that lets a
// system in any language be replayed.
type Command struct {
	// Ver is the version string this program represents.
	Ver string
	// Path is the program, and Args its arguments.
	Path string
	Args []string
	// Dir is the working directory; empty means the caller's.
	Dir string
	// Timeout bounds one run. Zero means one minute.
	Timeout time.Duration
	// MaxOutput bounds the bytes kept from one run's stdout, and a run that
	// writes more fails rather than being truncated into a false match. Zero
	// means 64 MiB.
	MaxOutput int
}

// defaultMaxOutput is far above any recorded output a corpus has held, and
// low enough that a runaway subject fails one input instead of the machine.
const defaultMaxOutput = 64 << 20

// waitDelay is how long a timed-out run's pipes may stay open after the
// kill. A subject that is a wrapper (make, a shell script) leaves children
// holding stdout, and without a bound Wait would block until they exit.
const waitDelay = 5 * time.Second

// Version returns the version string.
func (c *Command) Version() string { return c.Ver }

// Run feeds one input to the program and returns its output.
func (c *Command) Run(ctx context.Context, input []byte) ([]byte, error) {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	limit := c.MaxOutput
	if limit <= 0 {
		limit = defaultMaxOutput
	}

	//nolint:gosec // the program is the project's own configured replay subject
	cmd := exec.CommandContext(ctx, c.Path, c.Args...)
	cmd.Dir = c.Dir
	cmd.Stdin = bytes.NewReader(input)
	stdout := &capped{max: limit}
	stderr := &capped{max: 64 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = waitDelay
	killGroupOnCancel(cmd)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %v: %w: %s", c.Path, c.Args, err,
			bytes.TrimSpace(stderr.buf.Bytes()))
	}
	if stdout.over {
		return nil, fmt.Errorf("%s %v: %w", c.Path, c.Args, ErrOutputTooLarge)
	}
	return stdout.buf.Bytes(), nil
}

// ErrOutputTooLarge reports a run whose stdout passed Command.MaxOutput.
var ErrOutputTooLarge = errors.New("output exceeded the replay limit")

// capped keeps the first max bytes written to it and notes whether more
// arrived. It never returns an error, so the subject is not killed by a
// broken pipe mid-write and its exit status stays its own.
type capped struct {
	buf  bytes.Buffer
	max  int
	over bool
}

func (w *capped) Write(p []byte) (int, error) {
	if room := w.max - w.buf.Len(); room < len(p) {
		w.over = true
		if room > 0 {
			w.buf.Write(p[:room])
		}
		return len(p), nil
	}
	return w.buf.Write(p)
}

// A Difference is one input on which two versions disagree. It is an
// obligation, not a report: the only way to discharge it is Ledger's three
// closes, and editing the corpus is not one of them.
type Difference struct {
	Input    string
	From, To string
	Was, Now []byte
}

// String renders a difference for a terminal, hashes rather than payloads,
// because an input may be recorded at a confidentiality level that forbids
// printing it.
func (d Difference) String() string {
	return fmt.Sprintf("%s: %s produced %s, %s produced %s",
		d.Input, d.From, Hash(d.Was)[:12], d.To, Hash(d.Now)[:12])
}

// A Failure is one input a replay could not compare at all: the input could
// not be fetched from the store, or a version crashed, timed out, or
// otherwise refused to run. There is no output pair to compare, so this is
// not a Difference and none of Ledger's three closes apply to it directly —
// but a Failure that recurs on the same input across replays is itself the
// kind of thing a boundary decline or a law fix explains.
type Failure struct {
	Input string
	// Version names which subject failed to produce output: From, To, or ""
	// when the input itself could not be fetched from the store.
	Version string
	Err     error
}

// String renders a failure for a terminal.
func (f Failure) String() string {
	if f.Version == "" {
		return fmt.Sprintf("%s: could not be fetched: %v", f.Input, f.Err)
	}
	return fmt.Sprintf("%s: %s failed: %v", f.Input, f.Version, f.Err)
}

// A Result is what one replay found.
type Result struct {
	From, To    string
	Ran         int
	Same        int
	Differences []Difference
	// Failures is every input a version could not produce output for at
	// all. It is kept separate from Differences because there is nothing to
	// diff — one or both sides never finished — and separate from a run
	// error because one input's crash must not erase what every other input
	// found.
	Failures []Failure
	// Outputs maps input id to the new version's output hash, ready to be
	// recorded against that version in the manifest.
	Outputs map[string]string
}

// OK reports whether the two versions agreed on every input replayed and
// nothing kept a comparison from happening at all.
func (r Result) OK() bool { return len(r.Differences) == 0 && len(r.Failures) == 0 }

// ErrNoOutputs reports a replay that produced nothing to compare. Green on an
// empty corpus is honest; green on a corpus that failed to load is not.
var ErrNoOutputs = errors.New("replay ran no inputs")

// A Setting tunes a replay.
type Setting func(*settings)

// settings is the tuned state; its zero value is the conservative replay.
type settings struct {
	workers  int
	progress func(done, total int)
}

// Workers replays up to n inputs at a time, and one at a time by default.
//
// Sequential is the right default because the subject contract does not
// promise otherwise. "An input on stdin, the output on stdout" says nothing
// about running two at once, and a subject holding a port, a lock file or a
// single database would be wrong to fan out behind its author's back. Command
// spawns a process per input and is safe; whether some other subject is, is
// the caller's judgement and so it is the caller's flag.
//
// It matters when the subject is expensive: an oracle that launches an
// editor per input takes half an hour over a few thousand inputs in
// sequence, and a few minutes at eight.
func Workers(n int) Setting {
	return func(s *settings) {
		if n > 1 {
			s.workers = n
		}
	}
}

// Progress calls fn as each input finishes, with the count done so far and
// the total that will run. Without it, Run reports nothing until every
// input has finished — fine at corpus sizes that clear in seconds, and
// indistinguishable from a hang at thousands of inputs with an expensive
// subject. fn is called under a lock, so it never
// needs to guard against concurrent calls itself even when Workers is set.
func Progress(fn func(done, total int)) Setting {
	return func(s *settings) { s.progress = fn }
}

// an outcome is what one input produced, kept per-input so that a parallel
// replay reports exactly what a sequential one would. Which worker finished
// first must not reach the result.
type outcome struct {
	before, after []byte
	err           error
	// failedVersion names which subject raised err: From, To, or "" when the
	// store itself could not produce the input.
	failedVersion string
	ran           bool
}

// Run replays every active input against both versions and collects the
// differences. Retired inputs are skipped: they stay in the manifest so they
// can never pass for a wrong reason, and replaying them would be exactly
// that.
//
// One input crashing a version does not stop the rest: it is recorded as a
// Failure and every other input is still replayed, so one already-understood
// crash cannot hide a real difference found somewhere else in the same run.
// Canceling ctx still aborts the whole replay and returns ctx.Err(), because
// that is the operator's own decision and not a fact about any one input.
func Run(ctx context.Context, m *corpus.Manifest, store InputStore,
	was, now Subject, set ...Setting,
) (Result, error) {
	s := settings{workers: 1}
	for _, tune := range set {
		tune(&s)
	}

	active := make([]corpus.Input, 0, len(m.Inputs))
	for _, in := range m.Inputs {
		if in.Status == corpus.Active {
			active = append(active, in)
		}
	}
	r := Result{
		From:    was.Version(),
		To:      now.Version(),
		Outputs: make(map[string]string, len(active)),
	}

	outcomes := make([]outcome, len(active))
	indices := make(chan int)
	var wg sync.WaitGroup
	var done atomic.Int64
	var progressMu sync.Mutex
	for range s.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range indices {
				outcomes[i] = replayOne(ctx, store, was, now, active[i])
				if s.progress != nil {
					n := done.Add(1)
					progressMu.Lock()
					s.progress(int(n), len(active))
					progressMu.Unlock()
				}
			}
		}()
	}
	for i := range active {
		indices <- i
	}
	close(indices)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return r, err
	}

	for i, o := range outcomes {
		in := active[i]
		if o.err != nil {
			r.Failures = append(r.Failures, Failure{Input: in.ID, Version: o.failedVersion, Err: o.err})
			continue
		}
		if !o.ran {
			continue
		}
		r.Ran++
		r.Outputs[in.ID] = "sha256:" + Hash(o.after)
		if bytes.Equal(o.before, o.after) {
			r.Same++
			continue
		}
		r.Differences = append(r.Differences, Difference{
			Input: in.ID, From: r.From, To: r.To, Was: o.before, Now: o.after,
		})
	}
	return r, nil
}

// replayOne runs both versions on one input.
func replayOne(ctx context.Context, store InputStore, was, now Subject,
	in corpus.Input,
) outcome {
	payload, err := store.Get(in.SHA256)
	if err != nil {
		return outcome{err: err}
	}
	before, err := was.Run(ctx, payload)
	if err != nil {
		return outcome{err: err, failedVersion: was.Version()}
	}
	after, err := now.Run(ctx, payload)
	if err != nil {
		return outcome{err: err, failedVersion: now.Version()}
	}
	return outcome{before: before, after: after, ran: true}
}

// Record writes this replay's outputs into the manifest against the new
// version. It refuses to overwrite an output already recorded for that
// version: what a version did is history, and a re-run that disagrees with
// the record is a difference to close, not a value to update.
func (r Result) Record(m *corpus.Manifest) error {
	for i := range m.Inputs {
		in := &m.Inputs[i]
		hash, ok := r.Outputs[in.ID]
		if !ok {
			continue
		}
		if existing, recorded := in.Outputs[r.To]; recorded && existing != hash {
			return fmt.Errorf("%s: version %s already produced %s and now produces %s; "+
				"that is a difference to close, not an output to rewrite",
				in.ID, r.To, existing, hash)
		}
		if in.Outputs == nil {
			in.Outputs = make(map[string]string, 1)
		}
		in.Outputs[r.To] = hash
	}
	return nil
}
