# Boundary

Observables this repository declines to promise. Each entry names the
observable, the version it was declared at, the memory in the decision
record carrying the full reason, and a one-line summary of it. Entries are
appended, never edited
in place; to withdraw a promise, add an entry; to make a promise, remove an
entry in a major version and record the decision.

The entries below are generated from the decision record, unlike the prose
around them.

They come from the `Declines` claims there, and `germline boundary -write`
projects them here; `germline check` refuses this file when the two
disagree. Edit the record, not the list.

Formally this list is the complement of the test suite's distinguishing power:
two implementations that pass every law and every replay may differ on exactly
what is listed here, and on nothing else. Anything users can observe that
isn't on this list is a promise, whether or not anyone meant it to be.

## Entries

- **Ordering of entries in `corpus/manifest.json`** — 0.0,
  `GermlineCorpusInputsAppendOnlyOutputsPerVersion`. The manifest is a set;
  tooling must not depend on order.
- **The exact text of a `replay` failure message** — 0.0,
  `GermlineThreeWayClose`. Messages are for the reader at the terminal, not
  for parsing.
- **Where `germline laws` will find a `LAW (unchecked):` marker** — 0.1,
  `GermlineUncheckedMarkerAnchored`. Only where a comment opens; one inside
  a string literal or after other text on the line is not found.
- **Which law names `germline close -as law` accepts** — 0.1,
  `GermlineCLILawCheckIsTextual`. The command searches `laws/` for the name
  as text; the real check needs the Go API and a registered law set.

## How to read an entry when a replay diff lands on it

The diff is licensed. Close it as "boundary, already declared" with a pointer
to the entry. If the diff keeps landing on the same entry from many inputs,
that is evidence the entry should become a promise, and it goes to a human.
