# laws

A law takes an implementation and inputs and returns a violation or
nothing.

Every law here is quantified over generated inputs, never over three
chosen points. The standard library's `testing/quick` is enough to start;
`pgregory.net/rapid` adds shrinking once a law has more than one free
variable.

`example_law_test.go` checks the three lens laws — GetPut, PutGet, PutPut,
from Foster et al. 2007 and Bancilhon and Spyratos 1981 — over random
configs. [optkit](https://github.com/go-go-golems/optkit)'s `CheckLensLaws`
checks the same three laws at three fixed points; this is the
property-based form.

## Rules

A law's name describes what holds, never what the test does —
`TestLensLaws`, never `TestWorkersField`. Removing one is a decision: it
gets a memory in the decision record naming what replaces the check.

A law with no executable form yet is a comment at the top of the file it
belongs to, prefixed `LAW (unchecked):`, so it can be found and a reader
knows it is prose.

Cost laws especially — "replaying n keys costs O(n) parse steps and at
most one redraw" — often have nothing beyond a benchmark under `make
replay`. Write them anyway; treat a regression as a difference to close
like any other.
