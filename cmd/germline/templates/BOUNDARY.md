# Boundary

Observables this project declines to promise, published before anyone
depends on the behavior being there.

Each entry names the observable, the version it was declared at, the memory
in the decision record that carries the reason, and one line of that
reason. Entries are appended, never edited in place; to withdraw a promise,
add an entry; to make a promise, remove an entry in a major version and
record the decision.

Formally this list is the complement of the test suite's distinguishing power:
two implementations that pass every law and every replay may differ on exactly
what is listed here, and on nothing else. Anything users can observe that
isn't on this list is a promise, whether or not anyone meant it to be.

Write the first entry before anyone depends on anything. It is cheapest then,
and a boundary that waits until there are dependents never gets written.

This file is the source until the decision record carries `Declines` claims
for this project. Once it does, the entries are generated from them by
`germline boundary -write` and `germline check` refuses a file that has
drifted; the prose around the list stays written here either way.

## Entries

## How to read an entry when a replay diff lands on it

The diff is licensed once a human has approved the entry by committing it.
Close it
as `-as boundary` with the entry named verbatim; against an entry nobody has
approved yet, the close is kept as a proposal and waits. If the diff keeps landing on the same entry from many inputs, that
is evidence the entry should become a promise, and it goes to a human.
