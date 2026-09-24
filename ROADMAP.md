# Roadmap

Ordered. Each item names what done looks like. Releases are in
`CHANGELOG.md`.

## Done

- **The framework.** The four artifacts are machine-checkable, and every
  check the method calls a program is one: `pkg/boundary`, `pkg/corpus`,
  `pkg/law`, `pkg/replay`, `pkg/decide`, and a
  `germline` command that consumes them rather than reimplementing them.
- **A sample system.** `examples/querystring/`: an algebra, a port with two
  implementations, a conformance suite, a witnessed boundary, a recorded
  corpus, and a replay whose differences close as boundary. Runs with no
  setup.
- **A brownfield pilot.** The normal-mode command grammar of a 363 KLOC C
  fork of Neovim, with the running editor as the oracle, in
  `germline/wovim-normal/` of https://github.com/wovim/wovim. Its corpus
  combines inputs enumerated from the extracted command tables with inputs
  harvested from the fork's own functional tests. Every difference is
  closed, and every boundary entry is witnessed.
- **The boundary generated from the record.** A decline in the decision
  record projects into `BOUNDARY.md`, and `germline check` refuses a file
  that has drifted from it.
- **A decision record on day one.** `germline init` scaffolds `decisions/`,
  so a new project needs no external store.

## Next

- [ ] **A worked example over stochastic content.**
  Both worked examples are deterministic parsers, so the claim that
  metamorphic laws need no oracle is asserted in `docs/the-method.md` and not
  yet demonstrated. Done means a sample system with no reference
  implementation whose laws are metamorphic relations over a stochastic
  output. An example of such a relation: an outcome built from a strictly
  worse starting draw must never score better than one built from a better
  draw.
- [ ] **Keep the text of a refused close.** `germline close` refuses a close
  it cannot back and writes nothing, so the attempted reason is lost. Done
  means an append-only log of refused closes (input, versions, attempted
  kind, attempted reason, which check refused it) and a count per kind in
  `germline metrics`. Repeated refusals for an unpublished boundary entry
  are the method's named failure: writing the boundary after the fact to
  excuse a difference.
- [ ] **An evidence tier on each decision.** Replay results, laws and
  witnessed entries are measured, and many decisions are only argued. Done
  means each decision in the record is marked measured, sourced or argued,
  and a metric counts the closes that rest only on argued decisions.
- [ ] **A scaffold mode for the phase before a boundary exists.** The loop
  assumes a settled target. Done means `germline scaffold` drafts a
  provisional boundary and candidate laws from an existing codebase, marked
  as drafts to be argued with rather than trusted.
- [ ] **State the completeness rule in the method.** Once replay shows no
  open differences, a difference found later by another corpus source (a
  fuzzer, a harvest, a user report) means the corpus was under-sampled, and
  the implementation may still be right. Done means the rule is in
  `docs/the-method.md` and `germline metrics` separates differences by the
  source that found them.
- [ ] **Regenerate the pilot in a second host language against the same
  corpus.** Done means a replay across the subject contract with every
  difference closed, which tests the claim that the four artifacts, and not
  the code, carry the system.
- [ ] **Check boundary drift in CI.** Only the pre-commit hook compares
  `BOUNDARY.md` with a record kept outside the repository. A project that
  keeps `decisions/` in the repository is already checked in CI. For an
  external store, done means a signed digest of each decline, exported from
  the store, that CI can compare against the file without seeing the store.
