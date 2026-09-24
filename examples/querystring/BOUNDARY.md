# Boundary

Observables this query-string codec declines to promise. Each entry names the
observable, the version it was declared at, the memory in `decisions/`
carrying the full reason, and a one-line summary of it.

Every entry here is witnessed: `laws/witness_test.go` perturbs the
implementation on exactly that observable and checks that the suite does not
notice. An entry the suite *can* distinguish is not a declined promise at all,
and the test says so by name.

The entries are generated from the `Declines` claims kept in `decisions/`,
not written here by hand.

`germline boundary -write` projects them onto this page; edit the record,
not the list, since the prose around it survives the projection either way.

## Entries

- **How `+` is decoded** — 0.1.0, `QueryPlusDecodingIsNotPromised`. Form
  submission reads it as a space and RFC 3986 reads it literally; both ship,
  and no rule satisfies both.
- **The case of hex digits in a rendered escape** — 0.1.0,
  `QueryPercentEncodingCaseIsNotPromised`. RFC 3986 makes `%2F` and `%2f`
  equivalent; compare parsed queries, not rendered bytes.
- **Whether a key with no `=` differs from one with an empty value** —
  0.1.0, `QueryEmptyValueShapeIsNotPromised`. Keeping them apart needs a
  third state in the value type that every caller would have to handle.

## How to read an entry when a replay diff lands on it

The diff is licensed. Close it as `-as boundary` with the entry named
verbatim. If the diff keeps landing on the same entry from many inputs, that
is evidence the entry should become a promise, and it goes to a human;
`germline metrics` counts it as the boundary reopen rate.
