// Command germline checks the four artifacts, records inputs, replays them
// between two versions, and closes the differences one of three ways.
//
// Every subcommand is a thin consumer of the packages under pkg/. Nothing is
// implemented twice: if the command can do something the library cannot, the
// command is wrong.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
)

// version is overridden at release with
// -ldflags "-X main.version=$(git describe --tags --always --dirty)". There is
// no constant to hand-edit; the git tag is the single source of truth.
var version = "dev"

// An env is what a subcommand writes to and reads the project from, so every
// one of them is testable without a process.
type env struct {
	stdout io.Writer
	stderr io.Writer
	dir    string
	// store overrides where decisions live. Empty means git config, which is
	// where a real project keeps it; an override exists because a sample
	// program has to carry its own record to be runnable.
	store string
	// base is the context long-running subcommands derive from. Nil means
	// context.Background, so a test may set a deadline and a caller need not.
	base context.Context
}

func (e *env) ctx() context.Context {
	if e.base == nil {
		return context.Background()
	}
	return e.base
}

type command struct {
	summary string
	run     func(e *env, args []string) error
}

func commands() map[string]command {
	return map[string]command{
		"version":  {"print the version", cmdVersion},
		"init":     {"write the four artifacts into a directory", cmdInit},
		"check":    {"check the artifacts against each other (laws run under go test)", cmdCheck},
		"metrics":  {"the numbers the method says to watch", cmdMetrics},
		"laws":     {"laws stated in prose and not yet executable", cmdLaws},
		"boundary": {"list the declined promises, or -write them from the record", cmdBoundary},
		"corpus":   {"validate the manifest and show what it covers", cmdCorpus},
		"record":   {"append a recorded input to the corpus", cmdRecord},
		"reweight": {"re-estimate the operational profile across the corpus", cmdReweight},
		"replay":   {"replay the corpus between two versions", cmdReplay},
		"close":    {"close a replay difference one of three ways", cmdClose},
	}
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "germline:", err)
		os.Exit(1)
	}
	e := &env{stdout: os.Stdout, stderr: os.Stderr, dir: dir}
	err = run(e, os.Args[1:])
	switch {
	case err == nil:
		return
	case errors.Is(err, errFailed):
		// The check already said what it found, on stdout, in the words the
		// reader needs. Adding "germline: check failed" underneath it says
		// nothing and buries the part that matters.
		os.Exit(1)
	case errors.Is(err, errAwaiting):
		// Also already explained. A distinct status so a gate can let a
		// session stop to ask, rather than trapping it between a red replay
		// and a close it is not allowed to make.
		os.Exit(3)
	default:
		fmt.Fprintln(os.Stderr, "germline:", err)
		os.Exit(1)
	}
}

// plural renders "1 difference" and "2 differences".
func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// errUsage is returned when there is nothing to do but print the usage.
var errUsage = errors.New("a subcommand is required")

func run(e *env, args []string) error {
	cmds := commands()
	if len(args) == 0 {
		usage(e.stderr, cmds)
		return errUsage
	}
	name := args[0]
	if name == "--version" || name == "-version" {
		name = "version"
		args = args[:1]
	}
	if name == "-h" || name == "--help" || name == "help" {
		usage(e.stdout, cmds)
		return nil
	}
	c, ok := cmds[name]
	if !ok {
		usage(e.stderr, cmds)
		return fmt.Errorf("unknown subcommand %q", name)
	}
	err := c.run(e, args[1:])
	if errors.Is(err, flag.ErrHelp) {
		// The flag package already printed the usage; asking for it succeeded.
		return nil
	}
	return err
}

func usage(w io.Writer, cmds map[string]command) {
	fmt.Fprintln(w, "usage: germline <command> [flags]")
	fmt.Fprintln(w)
	names := make([]string, 0, len(cmds))
	for n := range cmds {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(w, "  %-9s %s\n", n, cmds[n].summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags come after the command, and every command takes these two:")
	fmt.Fprintln(w, "  -C <dir>      work on this directory instead of the current one")
	fmt.Fprintln(w, "  -store <dir>  the decision record, overriding `git config winze.store`")
}

func cmdVersion(e *env, _ []string) error {
	fmt.Fprintln(e.stdout, buildVersion())
	return nil
}

// buildVersion returns the most specific version string available: the
// ldflags-baked value from a release or `make install`; else the module
// version when installed with `go install ...@vX.Y.Z`; else the VCS revision
// (plus -dirty) from a local `go build`; else "dev" for a tarball build.
func buildVersion() string {
	if version != "dev" {
		return version
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev, dirty string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
			if len(rev) > 12 {
				rev = rev[:12]
			}
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if rev != "" {
		return rev + dirty
	}
	return version
}
