# germline

`germline` replays recorded inputs through two versions of a program and
does not let a difference slide. Each one closes exactly one of three ways:
fix the code, add a condition to a law, or declare the behavior a published
non-promise with a recorded reason. Every close is checked against the
artifact that has to back it.

```
$ germline replay -was 0.1.0=bin/form -now 0.2.0=bin/rfc
replayed 6 inputs from 0.1.0 to 0.2.0: 4 the same, 2 different, 0 failed

2 differences with nothing closing them:

  in-000002: 0.1.0 produced e8090de066fd, 0.2.0 produced a67c04aded32
      0.1.0: "\"q\"=[\"hello world\"];\nq=hello+world\n"
      0.2.0: "\"q\"=[\"hello+world\"];\nq=hello%2Bworld\n"
  in-000005: 0.1.0 produced 2d1497243b28, 0.2.0 produced 7664a0a58313
      0.1.0: "\"note\"=[\"c++ rocks\"];\nnote=c%2B%2B+rocks\n"
      0.2.0: "\"note\"=[\"c+++rocks\"];\nnote=c%2B%2B%2Brocks\n"

$ germline close -input in-000002 -from 0.1.0 -to 0.2.0 -as "edit the corpus"
germline: -as "edit the corpus"; the only three closes are [implementation law boundary], and editing the corpus is not one of them

$ germline close … -as boundary -entry "Plus signs" …
germline: the boundary does not declare this observable; publish the entry first: "Plus signs"

$ germline close … -as boundary -entry 'How `+` is decoded' -decision QueryPlusDecodingIsNotPromised …
closed in-000002 0.1.0→0.2.0 as boundary, citing QueryPlusDecodingIsNotPromised
the entry was approved: committed a090fb8cc95c
```

That session is `examples/querystring/`, with flags trimmed: a query-string
codec whose two versions disagree about whether `+` means a space. Both
readings ship in real software, so the difference closes as a published
non-promise.

The germline is what a lineage conserves while the body is rebuilt every
generation. Here it is four artifacts, kept beside the code, so that code a
model writes or rewrites is held to them:

| artifact | what it is | where |
|---|---|---|
| laws | executable checks, quantified over generated inputs, never three chosen points | `laws/` |
| boundary | the enumerated list of observables the system declines to promise, one reason each, published before anyone depends on them | `BOUNDARY.md` |
| corpus | inputs recorded from real use plus outputs re-derived per version; the diff between versions is what gets reviewed | `corpus/` |
| decision record | one entry per decision, called a *memory*, superseded rather than edited, written for a reader with no context | `decisions/`, or an external store |

The aim is code that is disposable for the failure modes these four check:
semantic drift on regeneration, disagreement with an oracle, an unpublished
promise. Regenerating a whole system from them in a second language is not
yet demonstrated, and `ROADMAP.md` tracks it. A system's whole correctness
does not reduce to four artifacts. Where a faster, more direct check exists
for something these cannot express, use it.

## The loop

Generate an implementation against laws, ports, and boundary, then run the
laws, the conformance suite, and a replay of the corpus against the previous
version. Close every difference with exactly one of three edits: the
implementation was wrong, the law was missing a condition, or the behavior
was never a promise and moves to the boundary with a reason. The corpus is
never edited, and the checker is never the author.

## Install

```
go install github.com/justinstimatze/germline/cmd/germline@latest
cd your-project && germline init && germline check
```

`init` writes a `BOUNDARY.md`, a `corpus/` with an empty manifest, a `laws/`
README, and a `decisions/` record. The record is Go source that germline
parses and never builds, so it works the same in a project written in any
language. A project that keeps its decisions elsewhere passes `-store <dir>`.

A replay compares two subject programs, the previous version and the new
one, each reading one recorded input on stdin and writing its output on
stdout. `examples/querystring/README.md` walks the whole loop, replay
included.

## What the tool does

```
germline init      write the four artifacts into a directory
germline check     check the artifacts against each other (laws run under go test)
germline metrics   the numbers the method says to watch
germline laws      laws stated in prose and not yet executable
germline boundary  list the declined promises, or rewrite the file
germline corpus    validate the manifest and show what it covers
germline record    append a recorded input to the corpus
germline reweight  re-estimate the operational profile across the corpus
germline replay    replay the corpus between two versions
germline close     close a difference one of three ways, or be refused
germline version   print the version
```

Three of those checks have no equivalent in an ordinary test suite:

- **The three-way close is checked.** `germline close` takes `-as
  implementation|law|boundary` and nothing else. A law close must name a law
  that exists. A boundary close must cite an entry already published, and one
  a human has committed. A close against an uncommitted entry is kept as a
  proposal, and `replay` exits 3, "waiting on a human", instead of 1. Every
  close names a decision the record carries.
- **Append-only is enforced against git.** `germline check` compares the
  manifest against the one committed at `HEAD` and reports any change that is
  not an append: a rewritten recording, a deleted input, an edited output, an
  un-retirement, a re-closed diff.
- **Boundary entries are testable claims.** An entry says the suite cannot
  tell an implementation with this behavior from one without it.
  `boundary.Distinguish` perturbs the implementation on exactly that
  observable and requires the suite not to notice. If it notices, the entry
  is false and the check names it. It is mutation testing pointed the other
  way: a mutant the suite is required to let live.

## Two layers

The portable layer is file formats and the command: `BOUNDARY.md` parsed and
projected, `corpus/manifest.json`, and a replay subject that is any program
reading one input on stdin and writing its output on stdout. A C binary, a
Python service and a Go package are all replayable, which matters because the
systems worth doing this to are rarely written in one language.

The typed layer is the Go packages under `pkg/`, which the command itself
consumes, so the library is never a second implementation that drifts. In Go,
a law is a value:

```go
s.Register(law.Lens("Workers", get, put)...)         // three laws
s.Register(law.Roundtrip("Format", render, parse))   // one
s.Register(law.Metamorphic("CostGrowsWithInput", …)) // where there is no oracle
```

Pre-1.0: the Go API under `pkg/` may change.

## Start here

- `examples/querystring/` — a sample system with all four artifacts, two
  implementations of one port, a witnessed boundary, and a recorded corpus.
  Runs with no setup and shows the whole loop in one directory.
- `docs/glossary.md` — every term, with its source where it has one.
- `docs/the-method.md` — the method in full, with the nine-stage lifecycle.
- `docs/prior-art.md` — which paper owns each idea, and what is left over.

## Status

The framework checks itself: `make check` runs `germline check` against this
repository.

As of September 2026 one real system has been through the whole loop: the
normal-mode command grammar of a C fork of Neovim, with the running editor
as the oracle. Its artifacts, numbers and findings live beside it, in
`germline/wovim-normal/` of https://github.com/wovim/wovim.

## Lineage

The design-doc-and-diary discipline, ports as small interfaces, and
conformance suites where a port forks are Manuel Odendahl's practice across
his [go-go-golems](https://github.com/go-go-golems) repositories, kept whole.
The decision record's format is that of a
[winze](https://github.com/justinstimatze/winze) store, and germline reads a
full winze store the same way it reads `decisions/`. The loop is one instance
of a dev-time [hybrid](https://github.com/justinstimatze/hybrid) loop. The
rest is prior art from 1969 to 2008, cited in `docs/prior-art.md`. What is new
is the economics: a second implementation is cheap, and the cost of keeping a
formal artifact in step with code falls sharply when the code is derived from
it.

## License and contact

MIT. See `LICENSE`. Contact: justin@justinstimatze.com. Security reports:
`SECURITY.md`.
