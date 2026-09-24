# Contributing

Run `make git-hooks` once per clone, so the pre-commit hook runs `make check`
before every commit. `make check` must pass: formatting, vet, lint, race
tests, the laws, `germline check` against this repository and the sample
system, the link check, govulncheck, and a tidy check. CI runs the same.

Two kinds of change need a maintainer's decision before any code: moving
something into the boundary, and removing a law. Open an issue with the
reason and the affected corpus items before sending either.

Any change to a law or to the boundary is recorded as a decision in the
decision record, with the reason, the alternatives considered and what would
reverse it. The recorded corpus is never edited; a replay difference closes
as an implementation fix, a law condition, or a boundary entry with a
reason.
