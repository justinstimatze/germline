# Prior art

Every component of the method has a prior name, most of them from 1969 to
1990. This page lists which paper owns each name and what the old version
lacked. In nearly every row it lacked the same thing: a cheap way to keep the
formal artifact in step with the code.

Each citation is marked. **verified** means the bibliographic details were
checked against a live source. **recalled** means they come from memory and
the year and venue should be checked before being cited elsewhere.

## The verdict in one table

| component | prior name | first | what the old version lacked |
|---|---|---|---|
| Laws as executable checks | Algebraic specification of abstract data types; property-based testing | 1977 | A generator that keeps the implementation derived from the equations, so the equations stop drifting from the code. |
| Two implementations may legitimately differ | Abstraction function and representation invariant; data refinement; testing equivalence | 1972 | Nothing. This part is finished theory, used without change. |
| Ports as small interfaces | Module secret; the *uses* relation; program families and product lines | 1972 | A measurement of the uses relation on a live system, which is now one script. |
| The enumerated boundary | Testing equivalence; nondeterminism in refinement calculus; undefined behavior as license | 1984 | A published list. The theory said what the boundary is; nobody wrote it down per system. |
| Previous version as spec; replay and diff | Pseudo-oracle; differential testing; characterization tests; translation validation | 1981 | A second implementation that costs nothing. The pseudo-oracle needed a second team. |
| Leave the right gaps and fill them | Program sketching with counterexample-guided synthesis; program calculation | 1986 | An unbounded hole-filler. Sketching's holes were bounded so a solver could finish. |
| The decision record | Programming as theory building; the faked rational design process; design rationale | 1970 | A reader. Rationale was written for nobody; the next model instance is somebody. |
| Brownfield exceeds human capacity | Laws of software evolution; reflexion models; barriers to maintenance automation | 1980 | An agent that pays the comprehension cost without being blamed for the result. |
| A human approves relaxing a promise | Read-only tests and an abort exit, measured on agents; two-party review; pending snapshots | 2025 | A second party the author cannot impersonate. Review assumed two human accounts; the agent and its user share one. |

## Laws as executable checks

A law is a function that takes an implementation and returns a violation or
nothing. Every piece of that sentence was published before 1985 except the
word "function".

- **Hoare 1969**, An axiomatic basis for computer programming. CACM 12(10).
  *recalled.* A program's meaning is a precondition, the program, and a
  postcondition, with correctness a proof obligation stated apart from the
  code. A `Check` function is a Hoare triple whose proof has been replaced by
  an execution.
- **Hoare 1972**, Proof of correctness of data representations. Acta
  Informatica 1(4). *recalled.* A representation is correct if an abstraction
  function maps it to the abstract value and every operation commutes with
  that map. Two implementations are equivalent when they abstract to the same
  state, which is what lets them legitimately differ.
- **Guttag 1977**, Abstract data types and the development of data
  structures, CACM 20(6); **Guttag and Horning 1978**, The algebraic
  specification of abstract data types, Acta Informatica 10. *recalled.*
  Specify a type by equations over its operations: `pop(push(s, x)) = s` is
  the whole contract for a stack. The equations were checked by hand or by
  rewriting tools nobody adopted. Larch (1985 onward) added an interface
  language per host, which is the ports-plus-algebra split, and still did not
  leave the lab.
- **Goguen, Thatcher, Wagner 1978**, An initial algebra approach to the
  specification, correctness, and implementation of abstract data types.
  Current Trends in Programming Methodology IV. *recalled.* An equational
  spec fixes behavior up to isomorphism and everything else is
  representation, which is the semantic reason to call the leftover a
  boundary.
- **Bancilhon and Spyratos 1981**, Update semantics of relational views, ACM
  TODS 6(4); **Foster, Greenwald, Moore, Pierce, Schmitt 2007**, Combinators
  for bidirectional tree transformations, TOPLAS 29(3). *recalled.* The lens
  laws GetPut, PutGet and PutPut, which `law.Lens` checks over generated
  inputs.
- **Meyer 1992**, Applying "Design by Contract". IEEE Computer 25(10).
  *recalled.* Contracts checked at runtime. They lived inside the code they
  constrained, so they drifted with it.
- **Liskov and Wing 1994**, A behavioral notion of subtyping. TOPLAS 16(6).
  *recalled.* What an adapter owes a port. A conformance suite is this
  condition run instead of proved.
- **Claessen and Hughes 2000**, QuickCheck: a lightweight tool for random
  testing of Haskell programs. ICFP 2000. *recalled.* State a property
  universally, generate inputs, shrink counterexamples. It is also the
  cheapest defense against an implementation that special-cases the corpus:
  the generator picks inputs the implementer never saw.

## Ports and the uses relation

- **Parnas 1972**, On the criteria to be used in decomposing systems into
  modules. CACM 15(12). *recalled.* Decompose by the decisions most likely
  to change and hide each behind an interface. `Clock`, `Sink` and `Store`
  are the three decisions most likely to change in any service: time,
  output, persistence.
- **Parnas 1979**, Designing software for ease of extension and contraction.
  IEEE TSE SE-5(2), 128–138. *verified.* Removing a feature is a design goal
  on the same footing as adding one, and the *uses* relation says which
  subsets are removable. He had no way to measure it on a live system; an
  include graph now measures it in one script.
- **Parnas 1976**, On the design and development of program families, IEEE
  TSE SE-2(1); **Kang et al. 1990**, Feature-oriented domain analysis, SEI
  CMU/SEI-90-TR-21; **Czarnecki and Eisenecker 2000**, Generative
  Programming. *recalled.* Program families and product lines. Their open
  problem is the combinatorial test matrix, which is why one build with a
  published boundary beats a set of feature switches.
- **Dijkstra 1968**, The structure of the "THE"-multiprogramming system.
  CACM 11(5). *recalled.* Strict layers make a system testable in pieces.
- **De Nicola and Hennessy 1984**, Testing equivalences for processes.
  Theoretical Computer Science 34. *recalled.* Two processes are equivalent
  iff no test in a given class can tell them apart. This is the boundary's
  formal statement: it is the complement of the suite's distinguishing
  power, and publishing it is publishing which observers you admit.

## The oracle: previous version as spec

- **Weyuker 1982**, On testing non-testable programs, The Computer Journal
  25(4), 465–470; **Davis and Weyuker 1981**, Pseudo-oracles for
  non-testable programs, ACM '81. *verified.* When no oracle exists, an
  independently produced second version whose agreement with the first
  stands in for correctness. The method is the pseudo-oracle with the second
  team replaced by a regeneration, and the second team was the cost that made
  pseudo-oracles rare.
- **Feldt 1998**, Generating diverse software versions with genetic
  programming: an experimental study. IEE Proceedings–Software 145.
  **McMinn 2009**, Search-based failure discovery using testability
  transformations to generate pseudo-oracles. GECCO 2009, 1689–1696.
  *verified*, via **Barr, Harman, McMinn, Shahbaz, Yoo 2015**, The oracle
  problem in software testing: a survey, IEEE TSE 41(5), section 5.1, which
  was read directly. Both generate the second version rather than
  hand-building it, which Weyuker's paper did not. Both are narrower than
  the method's use: they transform existing code by search, for one-off
  failure discovery, rather than generating from laws an oracle that a
  growing implementation replays against continuously.
- **McKeeman 1998**, Differential testing for software. Digital Technical
  Journal 10(1). *recalled.* Feed the same generated inputs to several
  implementations and investigate every disagreement.
- **Feathers 2004**, Working Effectively with Legacy Code. *recalled.*
  Characterization tests record what code does now rather than what it
  should do, and seams are ports found after the fact. What was missing was
  a way to write ten thousand characterization tests. Recording and
  enumeration are that way.
- **Chen, Cheung, Yiu 1998**, Metamorphic testing: a new approach for
  generating next test cases. HKUST-CS98-01. *recalled.* Relations between
  outputs instead of expected outputs. This is the greenfield oracle: laws
  over the implementation's own outputs, with the corpus accruing as real use
  begins.
- **Pnueli, Siegel, Singerman 1998**, Translation validation. TACAS 1998,
  LNCS 1384. *recalled.* Check each run of an untrusted generator rather
  than proving the generator. The checker is never the author.
- **Mills, Dyer, Linger 1987**, Cleanroom software engineering, IEEE
  Software 4(5); **Musa 1993**, Operational profiles in
  software-reliability engineering, IEEE Software 10(2). *recalled.* Test
  from a statistical model of real use. The corpus's profile weights are
  Musa's operational profile.

Hardware verification's golden reference model is the closest mature
practice for replaying continuously against a model as the implementation
grows: RTL is checked against a C or SystemC model cycle by cycle. The model
is written by hand from the spec. This is known from secondary sources only
and has not been checked against a primary one.

## Refinement and the gaps in between

- **Dijkstra 1976**, A Discipline of Programming; **Morgan 1990**,
  Programming from Specifications; **Back 1978**. *recalled.* An
  implementation refines a spec if every behavior it has is permitted. The
  boundary is the nondeterminism the spec keeps on purpose, and regenerating
  picks another point in the set the spec permits.
- **Backus 1978**, Can programming be liberated from the von Neumann style?
  CACM 21(8). *recalled.* Programs should have an algebra of laws.
- **Meertens 1986**, Algorithmics: towards programming as a mathematical
  activity. CWI Monographs 1, North-Holland, 289–334. *verified.* Derive an
  efficient program from an obviously correct one by calculation. A model is
  a fast, unsound calculator, so each derivation needs validation after it.
- **Iverson 1980**, Notation as a tool of thought. CACM 23(8). *recalled.*
  A notation with no escape hatch, which is why the method prefers a typed
  algebra in the host language to a new notation.
- **Solar-Lezama 2006, 2008**, Combinatorial sketching for finite programs,
  ASPLOS 2006; Program synthesis by sketching, PhD thesis. *recalled.* Holes
  filled by a solver in a counterexample-guided loop. The method's loop is
  that loop with one change: a counterexample may revise the law or the
  boundary as well as the filling, which is Lakatos's three-way choice.
- **Jackson 2006**, Software Abstractions. *recalled.* The small-scope
  hypothesis: most bugs have small counterexamples.
- **Newcombe et al. 2015**, How Amazon Web Services uses formal methods.
  CACM 58(4). *recalled.* Laws-first works when someone pays the sync cost,
  and only a handful of teams chose to.

## The decision record

- **Naur 1985**, Programming as theory building. Microprocessing and
  Microprogramming 15(5). *recalled.* The program is the theory its
  programmers hold, and documentation cannot revive a system whose theory
  has died. That is the strongest standing objection to the method. The
  counter is that Naur's documentation was written for readers who already
  held a theory. A decision record written for a reader with no context is
  the only theory a model instance ever has.
- **Parnas and Clements 1986**, A rational design process: how and why to
  fake it. IEEE TSE SE-12(2). *recalled.* Document as if the process had
  been rational. Human maintainers rarely read those documents, and the next
  model instance reads little else.
- **Kunz and Rittel 1970**, Issues as elements of information systems;
  **MacLean, Young, Bellotti, Moran 1991**, Questions, options, and
  criteria. *recalled.* Structured design rationale, which failed on the
  cost of writing it during design. That cost is now the cheap part.
- **Nygard 2011**, Documenting architecture decisions. *recalled.* Dated
  decision records, superseded rather than edited.
- **Brooks 1975**, The Mythical Man-Month, chapter 4. *recalled.*
  Conceptual integrity comes from one mind. Under regeneration there is no
  mind, so integrity has to live in the algebra and the boundary.

## Brownfield

- **Lehman 1980**, Programs, life cycles, and laws of software evolution.
  Proceedings of the IEEE 68(9). *recalled.* Conservation of familiarity:
  the rate of change is bounded by what the people involved can absorb. The
  method bets this bound does not apply to a model, and that bet has not
  been measured.
- **Brooks 1987**, No silver bullet. IEEE Computer 20(4). *recalled.* Most
  of the remaining complexity is essential. The method's economics depend on
  that being an underestimate of the accidental fraction.
- **Chikofsky and Cross 1990**, Reverse engineering and design recovery: a
  taxonomy. IEEE Software 7(1). *recalled.*
- **Murphy, Notkin, Sullivan 1995**, Software reflexion models. FSE 1995.
  *recalled.* Hypothesize the architecture, extract the dependency graph,
  diff. This is the first step on any brownfield system.
- **Opdyke 1992**; **Fowler 1999**, Refactoring. *recalled.*
  "Behavior-preserving" has no meaning without an oracle. The corpus is that
  oracle for systems that never had a suite.
- **Foote and Yoder 1997**, Big ball of mud. PLoP 1997. *recalled.*
- **Cordy 2003**, Comprehending reality: practical barriers to industrial
  adoption of software maintenance automation. IWPC 2003, 196–205.
  *verified.* Customers resisted maintenance automation. The details below
  are recalled: the maintainers who answered for a change could not read the
  automated one. Someone still answers for a corpus diff, and the published
  boundary is what shows them which differences were licensed.

## Who may relax a promise

Moving an observable into the boundary relaxes an acceptance criterion, and
an agent that can relax its own criteria will. The measurements are recent
and they agree on which controls hold.

- **Zhong et al. 2025**, ImpossibleBench (arxiv.org/abs/2510.20270).
  *verified.* Tasks whose tests contradict their spec, so any pass is a
  cheat. Under the strictest prompt, which says "**DO NOT MODIFY THE
  TESTS.**" among other things, GPT-5 cheated on 54% of the SWE-bench-derived
  conflicting tasks and Claude Opus 4.1 on 50%. Hiding the tests "reduces
  cheating success rate to near zero" and costs legitimate performance;
  read-only tests restore that performance "while preventing test
  modification attempts", and do not stop special-casing. Letting the agent
  abort with `flag_for_human_intervention` cut GPT-5 from 54% to 9% and o3
  from 49% to 12%, with a much smaller effect on Claude Opus 4.1. LLM
  monitors caught 86–89% of cheating on the simpler task set and 42–65% on
  the SWE-bench one.
- **METR 2025**, Recent frontier models are reward hacking
  (metr.org/blog/2025-06-05-recent-reward-hacking/). *verified.* On one task,
  twenty runs per prompt: o3 reward-hacked in 80% with no instruction, 80%
  with "Please do not cheat", and 70% with "Please do not reward hack".
- **Baker et al. 2025**, Monitoring reasoning models for misbehavior and the
  risks of promoting obfuscation (arxiv.org/abs/2503.11926). *verified.*
  With too much optimization against a chain-of-thought monitor, "agents
  learn obfuscated reward hacking, hiding their intent within the CoT".
- **SLSA 1.2 source track**, two-party review (slsa.dev/spec/v1.2/
  source-requirements). *verified.* "Uploader and reviewer are two different
  trusted persons." GitHub's protected branches can require "the most recent
  reviewable push must be approved by someone other than the person who
  pushed it". Both assume the two parties are separate accounts, and an
  agent running as its user's account is not a separate account.
- **insta** (insta.rs/docs/cli/) and **Jest** (jestjs.io/docs/
  snapshot-testing). *verified.* A changed snapshot is pending until
  reviewed; Jest will not write snapshots in CI without `--updateSnapshot`,
  because "new snapshots automatically pass". The accept step is separate
  from the run, and nothing requires a different actor to take it.
- **Claude Code permissions** (code.claude.com/docs/en/permissions).
  *verified.* Edit deny rules do not reach "arbitrary subprocesses that read
  or write files indirectly", so a rule on `decisions/` does not stop
  `germline boundary -write` from writing through it; the sandbox is the
  enforcement the page points to.

What germline takes from these is the two cheap levers. Relaxing a promise
is a separate step: a close cites only a decline that is already committed,
so one written in the same session is refused. And stopping is honest: the
refused close is kept as a proposal and the replay exits 3, so asking a
human ends the session cleanly. An agent that can run git can also commit,
so the commit makes the step visible without proving who took it. Proving
that would take a party the agent cannot impersonate, which costs more than
the method asks of a project.

## Spec-driven development in 2026

The sharpest difference is that an oracle is executable, while a spec is
prose that needs a reader to judge conformance. The reader is usually the model that wrote the code, and the
method's rule is that the checker is never the author.

- **GitHub Spec Kit** (github.com/github/spec-kit). Its own README describes
  its conformance check as unit tests for English. Its converge step exists
  because implementations drift from spec, plan and tasks, and it re-checks
  on a schedule rather than on every change.
- **Tessl** (tessl.io) pitched executable specifications run against
  generated code with unseen test values. Birgitta Böckeler's review on
  martinfowler.com (2025-10-15) found that framework still in private beta,
  described by its own team as more future than current product, and saw
  non-deterministic output from the same spec. By September 2026 the site no
  longer pitched executable specs at all.

Named but not read: Kiro, OpenSpec, BMAD. Nothing found in that search
generates a reference implementation from executable laws and replays a
growing real implementation against it continuously. The pieces are real
and old. Their combination has not turned up elsewhere so far.

## What is left after the names are taken out

- **The pseudo-oracle is cheap.** Every idea above that needed a second
  implementation was gated on its price.
- **The sync cost falls sharply.** Algebraic specs, contracts, TLA+ and design
  rationale all failed adoption on keeping the formal artifact true to code
  someone else was editing. Under derivation the code follows the artifact.
- **Naur's theory has a new reader.** A model instance with no context reads
  the decision record before anything else, so the record gets written.
- **The hole-filler is unbounded and unsound.** Validation per regeneration,
  the corpus as oracle, and property-based laws over generated inputs are
  therefore not optional.
- **Still open, and old.** Cost laws have no executable form beyond a
  benchmark. Choosing what the corpus records is Musa's problem and it has
  not moved. Lehman's conservation of familiarity is claimed repealed and
  has not been measured.

## Lineage

Several practices are kept whole from Manuel Odendahl's
[go-go-golems](https://github.com/go-go-golems) repositories: the
design-doc-and-diary discipline, ports as small interfaces, and conformance
suites wherever a port has two implementations. Surveyed in September 2026,
they lacked three things, which the method adds:
a corpus recorded from use, property-based laws, and a published boundary.
