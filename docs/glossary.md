# Glossary

This glossary is alphabetical, for lookup — but reading five terms in this
order first makes the rest click faster: **artifact**, **law**,
**boundary**, **corpus**, **close**.

Where a term is borrowed, the source is named. Most of them are.

---

**accepted diff** — A difference between two versions on one recorded input
that has been closed and recorded, with which of the three closes discharged
it and which decision carries the reason. Lives in `corpus/manifest.json`.
Accepting a diff is a decision; it is not a review comment.

**algebra** — The typed operations a component offers, and the laws relating
them, written in the host language rather than a notation of its own. A lens
is one: a getter, a setter, three laws. The rule is *typed algebra in the host
language first, a notation only where the boundary is empty* — a
boundary-free notation dies the moment its escape hatch becomes the
program, and every one of them has an escape hatch.

**append-only** — The property that a corpus input, once recorded, is never
rewritten and never deleted. `germline check` enforces it against the
manifest committed at git HEAD, so it is a rule rather than a request. The one
field that may move is an input's **weight**, because the profile moves.

**artifact** — One of the four durable things: **laws**, **boundary**,
**corpus**, **decision record**. Code is not one of them. When you decide
where an hour goes, it goes into the four, and the code is regenerated from
them. See **soma**.

**boundary** — The enumerated list of observables the system declines to
promise, one reason each, published before anyone depends on the behavior.
Formally it is the complement of the suite's distinguishing power: two
implementations that pass every law and every replay may differ on exactly
what is listed, and on nothing else. From De Nicola and Hennessy's testing
equivalence. Its size over time is the **drift gauge**.

**close** — The act of discharging a replay **difference**. There are exactly
three and there is no fourth: the implementation was wrong (fix it), the law
was missing a condition (add the condition), or the behavior was never a
promise (move it to the boundary with a reason). Editing the corpus is not one
of them. From Lakatos: monster-barring, lemma-incorporation, exception-barring.
`pkg/replay`'s `Ledger` is a type rather than a paragraph because a paragraph
cannot refuse.

**component** — The unit germline governs: an algebra, its **ports**, and an
**oracle**. Chosen from the uses relation, lowest in-degree first. A thing with
in-degree 34 is not a component.

**confidentiality level** — How much of a recorded input is stored, from 1
(store nothing; shadow traffic and keep only mismatches) to 4 (store concrete
data outside the repository and replay where it lives). Fidelity falls as the
number rises. Picked per component, never per project.

**corpus** — Inputs recorded from real use, weighted by how often they happen,
plus outputs re-derived per version. The exam, not the textbook: never read it
before generating an implementation, because a finite set of points is
something you are very good at fitting.

**corpus coverage** — The share of the operational **profile** that the active
inputs carry. Gameable by weighting what was recorded rather than what
happens, which is why choosing what to record stays a human job.

**decision record** — One **memory** per decision, superseded rather than edited,
written for a reader with no context, because that is who reads it. A
project keeps it in `decisions/`, as Go source germline
parses and never builds, or in an external
[winze](https://github.com/justinstimatze/winze) store named by `git config
winze.store`. Every boundary entry and every
accepted diff cites one, and `germline check` resolves every citation.

**difference** — One recorded input on which two versions disagree. An
obligation, not a report. It stays open until a **close** discharges it.

**drift gauge** — Boundary size, trended. Growth without pruning is the escape
hatch becoming the program.

**distinguish** — To tell two implementations apart. The verb the boundary is
defined against, and the check `boundary.Distinguish` runs: perturb the
implementation on exactly one entry's observable and see whether the suite
notices. If it does, the entry is not a declined promise at all. See
**witness**.

**germline** — What a lineage conserves while the body is rebuilt every
generation. Here, the four artifacts. The body is the code.

**law** — A function that takes an implementation and inputs and returns a
violation or nothing, quantified over generated inputs and never over three
chosen points. Removing one is a decision. A law stated but not yet executable
is written as a `LAW (unchecked):` comment so it can be found.

**memory** — One entry in the **decision record**: a dated decision with its
reason, the alternatives seen, and what would reverse it. In the record's Go
source it is an exported top-level variable, and its name is what a boundary
entry or an accepted diff cites. The word is winze's, whose stores are a
model's long-term memory.

**metamorphic relation** — A law of the form "this known change to the input
makes this known change to the output", used where there is no oracle to
compare against. How greenfield work gets laws before there is a version one:
you rarely know the right answer and you nearly always know how the answer
must move. From Chen et al.

**operational profile** — Which operations happen, how often, in what
sequences, estimated over a window with recency weighting. From Musa. It is
what makes a corpus a sample of reality rather than a pile of files.

**oracle** — The thing that says whether an output is right: a recorded
corpus, a conformance suite, or a running previous version. *Own the oracle,
rent the code* — human hours go into the oracle, because search against an
oracle beats human knowledge encoded in the artifact. Sutton's bitter lesson
with the terms swapped.

**port** — A small interface in the domain package, named for what is likely
to change: time, persistence, output, identity. A dependency is a port with
someone else's boundary, so wrap it and record its behavior as the adapter's
corpus. From Cockburn by way of Parnas.

**profile** — See **operational profile**.

**reflexion loop** — Write down the five to ten interfaces you believe the
domain has, extract the real import or include graph, and compare. Where they
disagree, one is wrong, and it is usually your hypothesis. Repeat until they
agree. From Murphy, Notkin and Sullivan. The only architecture document worth
trusting.

**regen pass rate** — The share of regenerations that pass laws, suite and
replay on the first attempt. Falling means the four artifacts have drifted
from each other.

**replay** — Running every active corpus input against two versions and
collecting the differences. The thing anyone reviews is never a corpus entry;
it is the diff between version N and N+1 on the same inputs.

**retirement** — How a stale input leaves the active set: an appended event
with a version, a reason and a decision. Never a deletion. A retired input
stays in the manifest, visibly retired, so it can never pass for a wrong
reason.

**self-endorsed drift** — Of the differences a model-as-judge approved, how
many a program later caught. Above zero means the judge is not independent
enough. In practice it is well above zero, which is why *the checker is never
the author*: it is a program, and where a model must judge, it is a different
one and its verdict is not final.

**soma** — The body, as against the germline: the code, regenerated each
generation for what the four artifacts actually check, readable as a side
effect, and not a claim that a system's whole correctness reduces to those
four. If you find yourself defending a particular implementation, ask which
of the four artifacts it is standing in for, and move the knowledge there.

**subject** — One version of the system under replay. Portably, a program that
reads one recorded input on stdin and writes its output on stdout, which is
why a C binary, a Python service and a Go package are all replayable.

**three-way close** — See **close**.

**translation validation** — Checking that this particular regeneration
preserves behavior, rather than proving the generator correct in general. From
Pnueli, Siegel and Singerman. It is what makes an untrusted generator safe to
use, and it is why the corpus replay runs per regeneration and not per release.

**uses relation** — Which components depend on which, measured from the import
or include graph rather than assumed from a diagram. From Parnas. Nothing is
retired until the relation says nothing above it depends on it.

**weight** — An input's share of the operational profile. The only field of a
recording that may change after it is recorded, because the profile is
re-estimated as traffic moves.

**witness** — A function that perturbs an implementation so it differs on
exactly one boundary entry's observable and nothing else, turning the entry
from a claim into a testable one. Writing it is the real work; a witness
touching two observables at once proves nothing about either, and no
program can catch that — so it rests on the author's word the same way the
entry's reason does.
