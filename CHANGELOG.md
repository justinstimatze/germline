# Changelog

## 0.1.0 — 2026-09-24

Initial release.

- The four artifacts: executable laws (`laws/`), an enumerated boundary
  (`BOUNDARY.md`), a recorded corpus (`corpus/`), and a decision record
  (`decisions/`, or an external store).
- The `germline` command: `init`, `check`, `metrics`, `laws`, `boundary`,
  `corpus`, `record`, `reweight`, `replay`, `close`, `version`.
- The Go library under `pkg/`: `boundary`, `corpus`, `law`, `replay`,
  `decide`.
- A boundary close licenses a difference only against an entry a human
  approved, meaning every line of it is committed. A close against an
  uncommitted entry
  is kept in the manifest as a proposal, and `close` and `replay` exit 3,
  "waiting on a human".
- `examples/querystring/`, a sample system carrying all four artifacts.
- `hooks/replaygate`, a Claude Code `Stop` hook that refuses to stop while
  `make replay` is red.
- `skills/germline`, the arrival sequence and per-change loop as an agent
  skill.
