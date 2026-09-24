# germline

The germline is what a lineage conserves while the body is rebuilt every
generation. Here it is four artifacts: executable laws, an enumerated
boundary, a recorded corpus, and a decision record. Code is the soma,
regenerated from the four for the failure modes they actually check —
semantic drift on regeneration, disagreement with an oracle, an unpublished
promise. A system's whole correctness does not reduce to four artifacts. A
type system, a linter, a human glancing at a diff: reach for whichever
check is fastest for the failure mode in front of you, rather than routing
it through laws, corpus, or boundary because the apparatus is there.

This file is for working on germline itself, and its hooks assume the
author's own tools (onsetter, stull, a winze store). Adopting germline in
another project needs none of them: `skills/germline/SKILL.md` and
`germline init` are the whole of it.

This repo is the template and the method, and only that. A system's artifacts
belong beside the system: the one real pilot, the normal-mode command grammar
of a C fork of Neovim, keeps its four in `germline/wovim-normal/` of
https://github.com/wovim/wovim.

## On arrival, read in this order

1. `README.md`, then `docs/the-method.md` — the method, with the nine-stage
   lifecycle.
2. The decision record, current decisions first.
3. `docs/glossary.md` for any term, and `docs/prior-art.md` for where each
   idea comes from.
4. `skills/germline/SKILL.md` — the arrival sequence and per-change loop as a
   skill.
5. `ROADMAP.md`, then `git log --oneline`.

Do not start from the code; there is almost none.

## The decision record

A project's decisions live in its own `decisions/` directory, which
`germline init` scaffolds, or in an external store named by `git config
winze.store` when the project has no `decisions/`. Both are the same shape:
Go source in which a memory is an exported top-level variable, read by
parsing and never built. This repository's own record is an external
[winze](https://github.com/justinstimatze/winze) store, which is why a fresh
clone's `germline check` reports citations it could not resolve and passes.

A decision is never edited; a newer one supersedes it. Write each memory for
a reader with zero context: the reason, the alternatives seen, and what would
reverse it, with a date.

A declined promise is a `Declines` value whose subject is the memory holding
the reason. Once the record carries one, the record is the source of the
boundary and `BOUNDARY.md` is its projection.

## Layout

- `germline.go` — the shared vocabulary: `Artifact`, `Problem`, `Problems`.
  Every check speaks in these so no artifact package imports another.
- `pkg/boundary` — parse, project and witness `BOUNDARY.md`.
  `Distinguish` is the one that matters: it perturbs an implementation on
  exactly one entry's observable and requires the suite not to notice.
- `pkg/corpus` — the manifest schema, validation, and `VerifyAppendOnly`.
- `pkg/law` — laws as values, six families, the `LAW (unchecked):` scanner.
- `pkg/replay` — the subject contract, the replay, and `Ledger`: the
  three-way close as a type that refuses a close it cannot back.
- `pkg/decide` — the decision record as a port; `Winze` is the one
  implementation, reading the store's Go source rather than shelling out.
- `internal/metric`, `internal/project` — the six metrics, and the conventions that
  bind the four artifacts to a directory.
- `cmd/germline` — the subcommands, each a thin consumer of the above. If the
  command can do something the library cannot, the command is wrong.
- `laws/` — this repository's own laws, property-based. `make laws` runs them.
- `BOUNDARY.md` — enumerated non-promises, one reason each. The decision
  record is the source and this is the projection: `make boundaries` writes
  both files from the store's `Declines` values, and `germline check`
  refuses one that has drifted. A project whose record carries no declines
  keeps the file as its source, which is how a new one starts.
- `corpus/` — the manifest and its README on the input/output lifecycle.
  Recorded data lives outside the repo; inputs are never authored or edited.
- `docs/` — the method, the glossary, and the prior art.
- `examples/querystring/` — the sample system, carrying all four artifacts.
  The place to look before changing any of them.
  The sample system is the only other system this repository carries. The one
  real pilot lives in the fork it is about, at `germline/wovim-normal/` in
  https://github.com/wovim/wovim, with its own Makefile and its own module.
  It reads the same decision record this repository does, so a decline
  recorded here projects into its `BOUNDARY.md` there.
- `hooks/replaygate/` — the stull `Stop` gate: no stopping while
  `make replay` is red. Wired by `make replay-gate`; the refusal names the
  three-way close, because a block that says only "replay is red" leaves
  editing the corpus as the obvious next move.
- `skills/germline/` — the skill.

## Rules carried by hooks

Write files in this repo through `Write` and `Edit`, never through shell
heredocs or `sd`/`sed`. The asks below live in onsetter, a `PreToolUse` hook
on `Write` and `Edit` only, and a file written by shell walks past every one
of them.

Every glob is `**/`-prefixed so the asks reach the sample system and any
project nested in this tree, and `not-in:` keeps them off the templates
under `cmd/germline/templates/`. `onsetter replay '**/*'` reports how often
an ask would fire; it cannot measure `added:` or `removed:` gates, so drive
`onsetter hook` with an old/new pair for those.

Every anchored pattern in an `added:` or `removed:` gate carries `(?m)`.
Those gates match against the inserted or deleted lines joined into one
string, and without `(?m)` Go's `^` anchors to the start of that string
rather than the start of each line.

Where a system's declines live in an external record, a boundary move happens
there, and the gate for it belongs in that record's own `CLAUDE.md`.

```ask
in: **/corpus/**
not-in: {pkg/**,cmd/germline/templates/**,**/corpus/README.md}
on: edit

The corpus is recorded, never edited. A replay difference closes as an implementation fix, a law condition, or a boundary entry with a reason. Which of the three is this? If this is a new recording being appended, continue.
```

```ask
in: **/*_test.go
removed: (?i)\b(assert|require|Equal|Fatal|Errorf|want)\b

This edit removes an assertion. Is it a law change or a boundary move? Either is a decision: record it in the decision record before the test changes. If the assertion moved rather than vanished, continue.
```

```ask
in: **/BOUNDARY.md
not-in: cmd/germline/templates/**
added: (?m)^\s*[-*] 

A boundary entry is being added by hand. Where the decision record carries this system's declines, the file is generated from it and a hand-added entry is overwritten by the next `germline boundary -write` — record it instead, and germline's own check will refuse the file until you do. If this system has no declines in the record yet, the file is the source: has the decision been recorded with its reason, and does the entry name that memory?
```

```ask
in: **/laws/**
not-in: cmd/germline/templates/**
removed: (?m)^func (Check|Law|Prop|Test)

A law is being removed. Removal is a decision: what replaces the check, and is that recorded in the decision record? If the function moved files, continue.
```

## Two decisions stay human

A boundary move and a law removal. Bring the reason and the affected corpus
items, and stop. Everything else: do it, then report. Do not ask permission
to record, to regenerate, or to pick between two algebras the corpus cannot
distinguish; pick the smaller boundary and record the choice.

## Conventions

Go, `main` branch, MIT. Contact: justin@justinstimatze.com.

`make check` before every commit, and `make git-hooks` once per clone so the
pre-commit hook runs it for you: gofumpt and goimports via `golangci-lint
fmt`, `go vet` with every analyzer except fieldalignment and with strict
shadow, golangci-lint with staticcheck's ST family on, race tests, the laws,
`germline check` against this repository and the sample system, the hooks
build, the markdown link check, govulncheck, and a tidy check. A `//nolint`
needs a specific linter and an explanation or nolintlint rejects it. The
version string comes from the git tag through `-ldflags`; there is no constant
to edit.

CI runs all of that, `germline check` included. What it cannot do is resolve
a citation: the record is a private store outside this repository and a runner
has no `winze.store` to point at, so the check counts the citations it could
not verify, says so, and passes. `laws/project_law_test.go` is what makes that
safe — taking the record away can only take problems away with it, so a green
check in a clone would have been green with the store too.

What no runner can check is whether `BOUNDARY.md` still matches the record it
is generated from, because the record is not there to compare against. That is
structural rather than a gap waiting on work: any in-repo stand-in for the
record would carry the same fields the file already publishes, and checking
one against the other checks nothing. The pre-commit hook catches boundary
drift and it is the only thing that does, so a commit made with `--no-verify`
can still publish a drifted file.
