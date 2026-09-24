# laws

A law is a function that takes an implementation and inputs and returns a
violation or nothing. Every law here is quantified over generated inputs,
never over three chosen points.

In Go, `pkg/law` gives the families: one call declares a law and the property
is checked over as many generated inputs as you ask for.

```go
s := &law.Set{}
s.Register(law.Lens("Workers", get, put)...)   // three laws
s.Register(law.Roundtrip("Format", render, parse))
s.Register(law.Idempotent("Normalise", norm))
s.Register(law.Metamorphic("LongerInputCostsMore", statement, vary, run, holds))
if ps := s.Check(nil); !ps.OK() {
    t.Error(ps.Sorted().String())
}
```

## Rules

A law's name describes what holds, never what the test does —
`TestLensLaws`, never `TestWorkersField`. Removing one is a decision,
recorded with a memory naming what replaces the check.

A law with no executable form yet is a comment prefixed `LAW (unchecked):`,
so `germline laws` can find it and a reader knows it is prose. Cost laws
especially — "replaying n inputs costs O(n) parse steps and at most one
redraw" — often have nothing beyond a benchmark to run; write them anyway
and treat a regression as a difference to close like any other.
