# A sample system germline governs

A query-string codec, small enough to read in one sitting. It exists so the
whole method is visible in one directory: an algebra, a port with two
implementations, a conformance suite, property-based laws, a witnessed
boundary, a recorded corpus, a replay between two versions, and a difference
closed one of three ways.

No setup is needed to run any of this — the decision record is checked in
as `decisions/`, which a real project would keep in a private store outside
its repository, and every command below takes `-store` to point at it.

## What the system is

A query string is a good example because implementations genuinely disagree
about it and always have. HTML form submission decodes `+` as a space; RFC
3986 treats `+` as an ordinary character. Both ship in real software. Neither
is a defect. Deciding what to do about that disagreement is the exercise.

| file | what it is |
|---|---|
| `querystring.go` | the algebra — `Query`, `Parse`, `Render` — and the `Codec` port with two implementations |
| `laws/laws_test.go` | four laws, run against **both** codecs: that is what makes it a conformance suite |
| `laws/witness_test.go` | one witness per boundary entry, checking the suite really cannot distinguish it |
| `BOUNDARY.md` | three observables the codec declines to promise, one reason each |
| `corpus/` | six recorded query strings, weighted, with each version's output hash |
| `decisions/` | the decision record, as a Go package: the same shape `germline init` scaffolds |
| `cmd/form`, `cmd/rfc` | the two versions as replay subjects: stdin in, stdout out |

## Walk the loop

Build the tool and the two subjects:

```
make build
go build -o bin/form ./examples/querystring/cmd/form
go build -o bin/rfc  ./examples/querystring/cmd/rfc
```

Everything below assumes these two flags, so they are written once here and
elided after: `-C examples/querystring -store examples/querystring/decisions`.

**Run the laws.** Four properties, quantified over generated queries, against
both implementations:

```
go test ./examples/querystring/...
```

**Check the artifacts.** Boundary entries parse and each cites a decision the
record carries; the manifest validates; nothing was rewritten:

```
bin/germline check -C examples/querystring -store examples/querystring/decisions
```

**Replay one version against the other.** Six recorded inputs, four the same,
two different:

```
bin/germline replay -C … -store … -was '0.1.0=bin/form' -now '0.2.0=bin/rfc'
```

```
replayed 6 inputs from 0.1.0 to 0.2.0: 4 the same, 2 different, 0 failed

no open differences
```

It says no open differences because both were already closed, and the close is
in `corpus/manifest.json`. Delete the two `accepted_diffs` entries and run it
again to see the state a session actually meets:

```
2 differences with nothing closing them:

  in-000002: 0.1.0 produced e8090de066fd, 0.2.0 produced a67c04aded32
      0.1.0: "\"q\"=[\"hello world\"];\nq=hello+world\n"
      0.2.0: "\"q\"=[\"hello+world\"];\nq=hello%2Bworld\n"
  in-000005: 0.1.0 produced 2d1497243b28, 0.2.0 produced 7664a0a58313
      0.1.0: "\"note\"=[\"c++ rocks\"];\nnote=c%2B%2B+rocks\n"
      0.2.0: "\"note\"=[\"c+++rocks\"];\nnote=c%2B%2B%2Brocks\n"
```

The outputs print because these inputs are recorded at level 3, an
abstraction holding nobody's data. At level 1 or 4, only the hashes print.

**Close them.** Exactly one of three ways, and the command refuses the others:

```
bin/germline close -C … -store … -input in-000002 -from 0.1.0 -to 0.2.0 \
  -as boundary -entry 'How `+` is decoded' \
  -decision QueryPlusDecodingIsNotPromised \
  -reason 'the form and RFC readings of + both ship; neither is a defect'
```

Try it with `-as "edit the corpus"`, or with an entry that is not published,
or with a decision the record does not carry. Each is refused by name, and the
difference stays open, which is the correct state for a difference nobody can
justify.

**Read the gauge.**

```
bin/germline metrics -C … -store …
```

```
closes by kind        implementation 0, law 0, boundary 2
boundary reopen rate  1 of 2 boundary closes reopened an entry;
                      QueryPlusDecodingIsNotPromised took 2
```

Two closes, both in one column, both landing on one entry. On a real system
that is the signal to look: an entry many differences keep hitting is evidence
it should stop being a non-promise and become a promise, and that decision
belongs to a human.

## The part worth stealing

`laws/witness_test.go` is mutation testing pointed the other way. A boundary
entry makes a formal claim — *the suite cannot tell an implementation with
this behavior from one without it* — and until something tests that claim, an
entry is a place someone licensed themselves to be wrong. Each witness wraps
the codec so it differs on exactly one entry's observable, runs the whole
suite, and requires it not to notice.

The same file runs one control in the opposite direction. Key order is
not on the boundary; it is a promise, and `KeyOrderSurvives` checks it. So a
codec that reverses key order must be noticed by the suite, and the control
test fails if it is not. That is what catches a suite that tests nothing. A
query generator sized `rnd.Intn(size%5 + 1)` under `testing/quick`, which
passes a constant size of 50, is `rnd.Intn(1)`, always zero: every generated
query is empty, all four laws pass over nothing, and the reversed-key codec
slips through, which the control reports at once.
