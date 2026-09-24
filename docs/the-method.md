# The method

This is for whoever has to rebuild the theory of a system from its written
record, with no one to ask, and how to leave that record better than you
found it. The ideas are old and cited in `prior-art.md`; what is new is the
economics.

## What you own and what you rent

Four artifacts are durable. The code is not. When you decide where an hour
goes, it goes into the four, and the code is regenerated from them — for the
failure modes those four actually check, not because a system's whole
correctness reduces to them. Where a faster, more direct check exists for
something outside that shape, use it instead of forcing it through laws,
corpus, or boundary.

| | artifact | what it is |
|---|---|---|
| own | laws | Executable functions that take an implementation and inputs and return a violation or nothing. Quantified over generated inputs, never three chosen points. Removing one is a decision rather than an edit. |
| own | boundary | The enumerated list of observables the system declines to promise, one reason per entry, published before anyone depends on the behavior. Its size over time is the drift gauge. |
| own | corpus | Inputs recorded from real use, weighted by how often they happen, plus outputs re-derived per version. Inputs are append-only and never authored. Outputs are never edited; the diff between two versions on the same inputs is what gets reviewed, and accepting a diff is a decision. A stale input is retired by an appended event with a version and a reason, never deleted. |
| own | decision record | One memory per decision in the record, superseded rather than edited, written for a reader with zero context, because that reader is you. |
| rent | code | Derived from the four above plus a host language. Disposable. Readable as a side effect. If you find yourself protecting a particular implementation, ask which of the four it is standing in for, and move the knowledge there. |

Only the corpus needs a running system to check outputs against. Laws and
the boundary do not: a system with no oracle at all can still state laws and
publish a boundary today, and neither waits on the other three. What it
cannot do yet is replay. See "Two starting conditions" below for the
greenfield path in full.

## On arrival at any repository

Do this in order, all of it, before the first edit — a day on a large
system, an hour on a small one.

1. **Read the decision record before the code.** The repository's
   `decisions/`, or the external store it names. If there is no record and no
   docs, the system has no theory on disk and step 4 is where you start
   writing one.
2. **Run the full-state checks before believing anything.**
   `git log --all -- <path>` before you say a file does not exist.
   `git log --oneline -- <file>` before you say who wrote it or which fork it
   belongs to. The doc in front of you is a snapshot; the log is the state.
3. **Hypothesize the ports, extract the uses relation, diff.** Write down the
   five to ten interfaces you believe the domain has. Extract the real include
   or import graph. Compare. Where they disagree, one of them is wrong, and it
   is usually your hypothesis. Repeat until they agree. This is the reflexion
   loop and it is the only architecture document worth trusting.
4. **Find the oracle.** A recorded corpus, a conformance suite, hand-written
   tests, or nothing. If nothing, record before you touch: instrument the
   running system and capture real sessions until the operational profile is
   visible. A change without an oracle is a guess, and you will endorse your
   own guess.
5. **Pick the smallest removable thing.** From the uses relation, the
   component with the lowest in-degree and a natural port. Extract that
   first. It proves the loop on something that cannot take the system down,
   and it tells you what the loop costs here.

## The loop, per change

The loop runs generate, critique, revise, with critique kept separate from
the other two and required to produce a written list, using rules that
live outside you.

1. **Regenerate or edit** the implementation against the laws, ports, and
   boundary as written. Do not read the corpus first; treat it as the exam
   you'll be given, never as the textbook you study from.
2. **Run the laws, run the suite, replay the corpus** against old and new.
   Collect every difference. Write the list down before you touch anything.
3. **Close each difference with exactly one of three edits.** The
   implementation was wrong: fix it. The law was missing a condition: add the
   condition to the law. The behavior was never a promise: move it to the
   boundary with a reason. The corpus is never the thing edited.
4. **Record the decision.** Any edit to a law or the boundary is a decision.
   Write a memory with the reason, the alternatives you saw, and what would
   have to be true for it to be reversed, naming the earlier decision it
   supersedes if there is one. A boundary move is recorded rather than written:
   the memory carries the reason, a `Declines` value beside it carries the
   observable and the line the file publishes, and the file is generated from
   that. An entry with no reason cannot be written at all, which is a stronger
   thing than a check that the citation resolves — germline's own boundary had
   two entries citing a memory about something adjacent, and a name check
   cannot see the difference.
5. **Validate the result with a program.** The author judging its own
   output is the check this replaces. Per-regeneration translation validation. If a judgment call is
   unavoidable, it goes to a model of a different family, and its verdict is
   still not the last word.

## Rules you will want to break

| rule | why it holds |
|---|---|
| You are never the checker of your own output. | In practice a model endorses a real share of its own semantic drift, even after stating the exact distinction it broke. The checker is a program; where a model must judge, it is a different one, and its verdict is not final. |
| Never edit a test to make it pass. | The three-way close exists so you never have to. If a test is wrong, that is a law change or a boundary move, each recorded. A silently edited test is a law deleted with no trace, the one failure the structure is built to prevent. |
| Never move a corpus item to the boundary without a written reason. | The boundary is where dependency stops being your problem. Every entry without a reason is a place you licensed yourself to be wrong, and unreasoned entries make the drift gauge lie. |
| Never trust a document over the log. | A design doc said a subsystem was fork-specific; eighteen upstream commits said otherwise. Docs record intention at the time of writing. The log records what happened. |
| Never read the corpus before generating. | An implementation shaped to the recorded points passes the replay and fails the next user. Property-based laws over generated inputs exist because the recorded corpus is finite and you are very good at fitting finite things. |
| Never generalize before naming the file, the number, or the command. | A claim with no file behind it was written without looking. Your abstraction rate climbs exactly when you disagree, because disagreement offers fewer nouns. That is when to slow down and find one. |

## Mistakes worth not repeating

- Read the dependency graph's anomalies before reading what you were pointed
  at. A large file inside the main cycle that no prompt names is the one to
  open first.
- Run `git log --oneline -- <file>` before calling a subsystem local or
  inherited. An architecture doc's framing is a claim; the log is the record.
- Before publishing a rule, find one case in the code where it would decide
  and check whether the code agrees.
- Say whether a thing was written or executed. They are different states.
- Check a file exists before pointing at it.
- Before believing a green number, make it go red on purpose. A metric that
  counts an empty registry reads zero forever, and zero is a plausible
  number.
- A law needs coverage the way a boundary entry needs a witness. A generator
  sized `rnd.Intn(size%5 + 1)` under `testing/quick`, which passes a constant
  size of 50, is `rnd.Intn(1)`: every value is empty and every law passes
  over nothing. Perturb the implementation in a way the law should catch and
  check that it does.
- A threshold calibrated on sequential runs breaks when the runs go parallel,
  and in a recording harness it breaks as data: a slow run is reported as a
  hang and the input is excluded as if something were wrong with it. Keep the
  reasoning beside the constant, and keep exclusions as a list with a reason
  each, never a count.
- A guard on a proxy passes while the thing it stands for is wrong. Assert on
  the content, not on a count of it.

## Two starting conditions

**Brownfield, a system past human working memory.** The oracle exists and is
running. Record it. Extract one component at a time, lowest in-degree first,
strangler-fig with the corpus attached. Write the algebra for each component
as you extract it, never for the whole system at once. Expect the first
extraction to take five times as long as the fifth. Money is here. The
barrier is accountability, cleared by publishing the boundary so the human
who answers for the system can see what was licensed.

A third thing the pilot settled, about the corpus rather than the code. Record
from real use is right for a system with users and useless for one without:
a system nobody uses has an undefined operational profile rather than a
small one, and weighting a corpus by it is a placeholder wearing a
measurement's clothes.
Where the component is table-driven, enumerate the table instead. Coverage
stops being a fraction of a guess and becomes a fraction of the path set, which
is countable and can be a gate. The numbers are the argument: 121 hand-written
inputs agreed with the oracle 92 times and looked healthy; 1,143 enumerated
ones agreed 596 times against the same parser, and the gap was three real
defects — one of them invisible to the hand-written set because every case in
it had happened to use the feature correctly.

Two things the first pilot settled, in `germline/wovim-normal/` of
https://github.com/wovim/wovim. First, look for
an observation point the system already has before building one: a documented
event carrying the component's own state made the parse of a large C program
readable from outside the process, so the oracle was the installed binary and
nothing was compiled. Second, and this is the one to expect again: **an old
system's tables carry its grammar and its control flow carries its semantics.**
Extracting the tables got a regenerated parser to 92 of 121 inputs on the
second try, and every one of the 29 that were left came down to a line inside
one of 188 separate handlers — a count invented here, a type overridden there —
readable by a person and by no extraction. That split is where the boundary
goes, and naming it is most of what the first extraction buys.

**Greenfield, no version one.** No corpus exists and the algebra is being
invented at the same time. Laws first, with metamorphic relations standing in
for expected outputs: a metamorphic law relates two runs of the same
implementation to each other instead of checking one against a reference, so
it needs no oracle — a composite built from a strictly worse starting draw
must never score better than an identical one built from a better draw, and
that holds whether or not a reference implementation exists to compare
against. A second path manufactures the oracle instead of doing without
one. Weyuker named it in 1982: pseudo-oracle, an independently produced
second version whose agreement with the first stands in for correctness.
Her theory asks for a second version rather than a prior one, so it reaches
greenfield exactly as it reaches brownfield replay. Run the laws over any
implementation that satisfies them: a compact model of desired behavior,
written small enough to trust by inspection, made to answer for every
input rather than only the ones you thought to check, and built before the
expensive one on purpose — cheap now buys the real implementation
something to check against starting on line one, rather than after a
thousand lines went unchecked. Replay the real, growing implementation
against that model
exactly as if it were a recovered brownfield oracle. Metamorphic relations
need no oracle ever; this path manufactures one, once, and the ordinary
replay loop takes over from there. The independence the theory leans on is
the same one "you are never the checker of your own output" exists to
guard: two implementations generated by the same model from the same laws
can share blind spots a genuinely independent second team would not, so
agreement between them is weaker evidence than Weyuker's framing assumed,
not proof.

A compact model is executable, where a written spec needs a reader to judge
conformance; `prior-art.md` compares the two. Ports from the question of what is likely to change,
which is nearly always time, persistence, output, and identity. Record from
the first real session onward and treat the corpus as thin until the
operational profile is visible. Speed is here, and the risk is a boundary
that never gets written because nothing depended on anything yet. Write it
anyway; it is cheapest before there are dependents.

## Where the go-go-golems shape stops and this one starts

Across Manuel Odendahl's recent
[go-go-golems](https://github.com/go-go-golems) repositories, the diary-and-design-doc discipline is
universal, ports are in every library, a conformance suite shows up
wherever a port has two implementations, and algebras live in the
extracted kits. Keep all of that.
Three things are missing there: a corpus recorded from use, property-based
laws, a published boundary. Add those and the shape is complete. Remove
nothing to make room for them.

Only the first of the three needs an oracle. The corpus is a diff against
what came before, so something has to say what that earlier version did;
the replay is where the oracle requirement lives. A law is a relation inside one
implementation — `Roundtrip` says `parse(render(x))` returns `x`,
`Deterministic` says the same input gives the same output, `Lens` relates a
`get` to its `put` — and none of the six families takes a reference to
compare against. Neither does the boundary. A system with no oracle at all
can still state laws and publish a boundary; what it cannot do yet is
replay.

## The whole lifecycle

Nine stages. Each takes one of the owned artifacts in and puts one out, has a
check that is a program, and exists to prevent one named failure.

| stage | in | out | check | failure it prevents |
|---|---|---|---|---|
| Intake | A request, from anywhere | The request classified as one of six: a new law, a boundary move, a profile shift seen in recording, a host change, a cost requirement, an incident. | Can it be stated as a diff to laws, boundary, or profile? If not, it is not yet a request. | Building what was said instead of what must hold. "Add a cache" is an implementation and gets rewritten as the cost law it stands for. |
| Design | The classified request | A design doc in question, options, criteria form; candidate laws; ports touched; boundary entries proposed; cost law if any. For a new component, the reflexion loop first. | Wide equilibrium: regenerate under two candidate algebras against the current corpus, keep the one with the smaller boundary. | A design doc written after the code with the real order hidden. Fake the rational process and let the diary keep the real one. |
| Generate | Laws, ports, boundary, host | An implementation, or several. Cheap, so several. | None yet. The generator is untrusted by construction. | Reading the corpus first and fitting it. |
| Verify | Implementation, corpus, laws, suite | A written list of differences, each closed by one of three edits, and the decisions those edits create. | Property-based laws, conformance suite, replay diff old against new, cost benchmark. All programs. | Self-endorsed drift. |
| Review | The decisions | Two human decisions at most: a boundary move, a law removal. Everything else is reported. | The reviewer reads the reason and the affected corpus items, never the diff. | Diff review creeping back because the boundary was never published and the human has nothing else to read. |
| Release | Accepted diffs, manifest | A version whose number is set by the artifacts: major withdraws a promise, minor adds a law or a promise, patch changes only the implementation. | Shadow-diff in production on a fraction of live traffic before cutover. Expand and contract for any data model a fleet shares. | A boundary withdrawal shipped as a patch. |
| Operate | Live traffic | The corpus: new inputs, profile re-estimation with recency weighting, retirement events, every incident as a recorded input plus a candidate law. | Replay of the last release against the new inputs, continuously. | An incident fixed in code with no law added, so it recurs on the next regeneration. |
| Maintain | Time | Scheduled pruning: boundary entries re-justified per quarter, laws re-weighted by profile, dependency boundaries re-read, a re-derivation under the same artifacts each time the generator improves. | Boundary size and places-per-law, trended. | Rule drift. The escape hatch becoming the program. |
| Contract | A component nobody uses, by the profile | Its inputs retired, its promises withdrawn as a major, its laws removed with reasons, its decisions kept. The decision record outlives the code. | Parnas's uses relation: nothing above it depends on it, measured, not assumed. | A system that can only grow. |

Across every stage: a dependency is a port with someone else's boundary, so
wrap it and record its behavior as the adapter's corpus. Do not decompose
where the uses relation says no; a component with in-degree 34 is not a
component. The generator is a supply-chain actor, untrusted by construction.
Budget in runs, not hours. Four things stay human: choosing what to record,
naming what the schema cannot see, the two review decisions, scheduling the
pruning.

## What to measure

| metric | what it tells you |
|---|---|
| places per law | How many files a reader must open to verify one law. Gameable by deleting laws, which is why the law set is append-mostly. |
| boundary size | Entries per quarter. Growth without pruning is the escape hatch becoming the program. |
| corpus coverage | Fraction of the operational profile's weight the recorded corpus exercises. |
| regen pass rate | Regenerations that pass laws, suite, and replay on the first attempt. Falling means the four artifacts have drifted from each other. |
| self-endorsed drift | Of the differences a model-as-judge approved, how many a program later caught. Above zero means the judge is not independent enough. |
| boundary reopen rate | Of the differences closed as "boundary," how many a later diff reopened. The one calibration metric for the fuzziest judgment in the loop. The method keeps three model judgments that need calibrating: intake classification, design options, and the three-way close. |

## When to stop and ask

Two decisions belong to the human and only two. A boundary move: something
users could observe is about to stop being a promise. A law removal:
something that held will stop being checked. Bring the corpus items it
affects and the reason. The move happens in the decision record now, not in
`BOUNDARY.md`, so that is where the gate sits.

A rule saying so is not enough. Told plainly not to modify the tests, coding
agents on realistic tasks whose tests contradicted the spec still cheated
about half the time. So the gate is read from history.
`germline close -as boundary` licenses a difference only against a decline
whose lines a human has committed, and a close against anything newer is kept as a
proposal while the replay exits 3 instead of 1. Stopping to ask is then the
move that ends the session cleanly, which is what the same measurements
found cuts the cheating most. `docs/prior-art.md` has the sources.

Everything else, do, then report. Do not ask about
implementation. Do not ask which of two equivalent algebras to keep when the
corpus does not distinguish them; pick the smaller boundary and record the
choice. Do not ask permission to record.
