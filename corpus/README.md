# corpus

Two things with different lifetimes, kept apart on purpose.

**Inputs** are things users did, recorded from real use and weighted by how
often they happen. They are append-only and never authored. Their lifetime is
the product's.

**Outputs** are what version N did with an input. They are re-derived per
version and never edited. The thing that gets reviewed is never an entry; it
is the diff between version N and N+1 on the same inputs, and accepting a diff
is a decision that gets recorded in the decision record.

## Lifecycle of one input

recorded → active → retired

Retirement is an appended event with a version and a reason, never a
deletion.

A retired input stays in the manifest, visibly retired, so it can never
pass for the wrong reason — whether the cause is a feature removed (the
close is a boundary entry), a retention policy expiring it, or an erasure
request arriving. All three retire the same way.

## What lives where

- `manifest.json` — tracked in git. Entry ids, content hashes, the version each
  output was derived at, retirement events, accepted diffs each naming the
  memory in the decision record that carries its reason.
  `manifest.example.json` shows the shape.
- `inputs/` — gitignored. Recorded data, content-addressed by the hash in the
  manifest. Where this directory actually lives depends on the confidentiality
  level below.

## Confidentiality, strongest first

Pick per component. Fidelity falls as you go down the list.

1. **Do not store.** Shadow traffic to old and new, diff, keep only mismatches.
   The corpus is the profile plus the diff log. You can only replay what is
   live right now.
2. **Store the profile.** Which operations, how often, in what sequences,
   without the sessions themselves. Generate inputs from it. Loses every bug that depends on
   concrete data.
3. **Store the abstraction of the entry.** The same abstraction function the
   laws use. For an editor: the command stream and a synthetic buffer with the
   real one's structure and none of its text.
4. **Store concrete data outside the repo**, access-controlled, hashes in the
   manifest, replay runs where the data is.

## The drift gauge

If replay diffs keep closing in the same direction, every one a retirement or
a boundary entry, the boundary is a dumping ground and the corpus is decaying
into the profile. `germline metrics` counts closes by kind.

## Recording and replaying

```
germline record -level 3 -weight 0.01 session.json
germline replay -was '0.1.0=./old-version' -now '0.2.0=./new-version'
```

A replay subject reads one recorded input on stdin and writes its output on
stdout. That is the whole contract, and it is why a C binary, a Python
service and a Go package are all replayable.
