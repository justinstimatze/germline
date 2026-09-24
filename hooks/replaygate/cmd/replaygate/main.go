// Command replaygate is the Stop hook: it refuses to let a session stop while
// `make replay` is red.
//
// It is a stull deployment, not a second stull. The machine, the dispatch and
// the hook protocol are all upstream's — this is the "builds its own binary
// that imports this package" case stull's registry doc names, and the whole
// live path is one call to runtime.RunHook. germline needs its own binary
// rather than `stull run` for two reasons: the machine is wrapped here to carry
// the three-way close in its refusal, and the exec timeout is germline's to
// set.
//
//	replaygate run --machine replay-gate   what the Stop hook invokes
//	replaygate install [-write]            merge the hook into settings.local.json
//
// Every failure path exits 0. A hook that cannot run must never be the reason
// a session cannot stop.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/justinstimatze/stull/compile"
	"github.com/justinstimatze/stull/runtime"
	"github.com/justinstimatze/stull/spec"

	"github.com/justinstimatze/germline/hooks/replaygate"
)

// execTimeout bounds the replay. A replay that has not finished in this long is
// reported as a nonzero exit, which blocks — the wrong failure for a slow one,
// so the bound is set well above the longest replay run so far, about ten
// minutes on the largest corpus.
const execTimeout = 15 * time.Minute

// execOutputBytes is how much of the failing command's output reaches the
// refusal, matching stull's own dispatch. It is a tail, so the last thing the
// replay printed is the part that survives.
const execOutputBytes = 4000

// settingsPath is where the wiring has to land. It is gitignored, so an install
// that skips it leaves the gate dark while reporting success.
const settingsPath = ".claude/settings.local.json"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "replaygate: want `run --machine replay-gate` or `install [-write]`")
		os.Exit(0)
	}
	switch args[0] {
	case "run":
		os.Exit(run())
	case "install":
		os.Exit(install(len(args) > 1 && args[1] == "-write"))
	default:
		fmt.Fprintf(os.Stderr, "replaygate: unknown command %q\n", args[0])
		os.Exit(0)
	}
}

// run is the live hook path. The --machine argument stull's generated settings
// fragment passes is not parsed: this binary hosts one machine and reading the
// flag could only ever disagree with that.
func run() int {
	m, err := replaygate.Machine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "replaygate: %v\n", err)
		return 0
	}
	return runtime.RunHook(m, execModel)
}

// execModel routes the one cell to a subprocess. The cell's Model is the
// literal "exec" and its Instructions are the command line, so there is no
// model call anywhere in this gate: the judge is `make replay`'s exit code.
func execModel(c spec.Cell, ctx *spec.Context) string {
	if c.Model != "exec" {
		return "" // outside the cell's language: fails safe, releases the stop
	}
	return runtime.ExecModel(execTimeout, execOutputBytes)(c, ctx)
}

// install merges the Stop hook into settings.local.json. Without -write it
// prints what would change, which is the reviewable form of a file that is
// otherwise edited by a program and never read.
func install(write bool) int {
	m, err := replaygate.Machine()
	if err != nil {
		fmt.Fprintf(os.Stderr, "replaygate: %v\n", err)
		return 1
	}
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "replaygate: cannot resolve my own path: %v\n", err)
		return 1
	}

	existing := map[string]any{}
	switch data, readErr := os.ReadFile(settingsPath); {
	case readErr == nil:
		if jsonErr := json.Unmarshal(data, &existing); jsonErr != nil {
			fmt.Fprintf(os.Stderr, "replaygate: %s is not JSON: %v\n", settingsPath, jsonErr)
			return 1
		}
	case !os.IsNotExist(readErr):
		fmt.Fprintf(os.Stderr, "replaygate: %v\n", readErr)
		return 1
	}

	merged, added, err := compile.MergeHooks(existing, m, self)
	if err != nil {
		fmt.Fprintf(os.Stderr, "replaygate: %v\n", err)
		return 1
	}
	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "replaygate: %v\n", err)
		return 1
	}
	if !write {
		fmt.Printf("%s\n", out)
		fmt.Fprintf(os.Stderr, "replaygate: %d trigger(s) would be added; re-run with -write\n", added)
		return 0
	}
	if err = os.MkdirAll(filepath.Dir(settingsPath), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "replaygate: %v\n", err)
		return 1
	}
	if err = os.WriteFile(settingsPath, append(out, '\n'), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "replaygate: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "replaygate: %s carries the Stop hook (%d added)\n", settingsPath, added)
	return 0
}
