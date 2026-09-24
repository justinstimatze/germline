---
name: germline
description: Arrival sequence and per-change loop for a repository that keeps executable laws, an enumerated boundary, a recorded corpus, and a decision record, and regenerates code from them. Triggers when a repo has a laws/ directory, a BOUNDARY.md, a corpus/ manifest, or a decisions/ record; when asked to extract a component from a large system into laws, ports, and a corpus; when asked to regenerate an implementation against a conformance suite; or when a replay diff needs closing. Does not trigger for ordinary refactors in repos without those artifacts.
---

# germline

You must rebuild the theory of this system from its written record, with
nobody to ask. Four artifacts are durable; the code is not.

The `germline` command is all this needs; `germline init` scaffolds the four.
The project's name in the record is its directory's name, so a decline's
`System` field must match it.

## On arrival, in order, before the first edit

1. **Read the decision record before the code.** The repo's `decisions/`,
   or the store `git config winze.store` names. If there is neither and no
   docs, the system has no theory on disk and step 4 starts one.
2. **Run the full-state checks.** `git log --all -- <path>` before saying a
   file does not exist. `git log --oneline -- <file>` before saying who wrote
   it or which fork it belongs to. A document is a snapshot; the log is state.
3. **Hypothesize the ports, extract the uses relation, diff.** Write the five
   to ten interfaces you believe the domain has. Extract the import or include
   graph. Compare. Where they disagree, your hypothesis is usually the wrong
   one. Repeat until they agree. A system small enough to read in one sitting
   has one port, and this step is that sentence.
4. **Find the oracle.** A recorded corpus, a conformance suite, hand-written
   tests, or nothing. If nothing, record before touching anything.
5. **Pick the smallest removable thing.** Lowest in-degree with a natural
   port. Extract it first. It proves the loop on something that cannot take
   the system down and tells you what the loop costs here.

## Per change

1. Regenerate or edit against laws, ports, and boundary. Do not read the corpus
   first; it is the exam.
2. Run the laws, the suite, and the corpus replay against the previous version.
   Write the difference list down before touching anything.
3. Close each difference with exactly one of three edits: the implementation
   was wrong, fix it; the law was missing a condition, add it; the behavior was
   never a promise, declare it with a reason. The corpus is never the thing
   edited.
4. Any edit to a law or the boundary is a decision: a new memory in the
   record with the reason, the alternatives seen, and what would reverse it,
   dated, naming the earlier decision it supersedes if there is one. Where the
   record carries a system's declines, a boundary move is a `Declines` value
   beside that memory and `BOUNDARY.md` is generated from it with `germline
   boundary -write`; where it does not, the entry goes in the file and cites
   the memory by name.
5. Validate with a program. You are never the checker of your own output.
   Where a model must judge, it is a different family and still not final.

## Rules you will want to break

- Never edit a test to make it pass. That is a law change or a boundary move,
  each recorded as a decision.
- Never move a corpus item to the boundary without a written reason.
- Never trust a document over `git log`.
- Never read the corpus before generating.
- Never generalize before naming the file, the number, or the command.

## Stop and ask for exactly two things

A boundary move and a law removal, and bring the reason plus the affected
corpus items when you raise either one. Everything else: do it, then
report it.

The tool holds you to the first. A boundary close against an entry no human
has approved is refused, kept as a proposal, and exits 3; `replay` exits 3
while such proposals are all that is left. Exit 3 means stop and ask. Do not
commit the decline yourself to get past it — the approval is the
human's commit, and one you make is the bypass the check exists to catch.

## Where the rest is

The full method with the nine-stage lifecycle and the measurement table is
`docs/the-method.md` in the germline repo, and the prior art behind each
idea is `docs/prior-art.md` beside it.
