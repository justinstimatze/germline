# hooks

Rules that have to fire at the tool call where the mistake would happen.

A rule whose false positive costs one sentence is an
[onsetter](https://github.com/justinstimatze/onsetter) ask. A rule whose false
negative costs a shipped regression is a
[stull](https://github.com/justinstimatze/stull) guard. stull compiles
guarded statecharts into Claude Code hooks.

- The four asks live in `CLAUDE.md` at the repo root, where onsetter reads
  them. They need onsetter installed as a `PreToolUse` hook on `Write` and
  `Edit`; nothing else is installed per ask.
- `replaygate/` is the stull machine: on `Stop`, run `make replay`; block with
  the reason if it is red, allow if it is green. No model is called; the
  judge is the command's exit code.

## The replay gate

`make replay-gate` builds `bin/replaygate` and merges the `Stop` hook into
`.claude/settings.local.json`. That file is gitignored because it names an
absolute path on the machine that ran it, so the Makefile target is the
tracked form of the wiring, and re-running it is a no-op.

The machine is stull's own `examples/replaygate`, imported rather than
copied: a cell whose `Model` is the literal `"exec"` and whose
`Instructions` are a command line. The output cap, the fuel bound and the
fail-safe on unparseable output are stull's.

`replaygate.Machine` makes the refusal name the three-way close, because a
block that says only "replay is red" leaves editing the corpus as the
obvious next move. It also keeps the gate standing: every transition points back at
the one live state, so a clean stop does not end the machine and let every
later stop in the session through. stull's `check.Inspect` compiles that
shape with a `W-HALT` warning, which is expected. `replaygate.Machine`
returns an error rather than patching upstream by position, so an upstream
reshape fails at install instead of silently dropping the refusal text.

`germline replay` exits 3 when the only open differences are boundary closes
proposed and waiting on a human to approve the entry. The gate judges exit
codes as zero or not, so a project whose `make replay` runs `germline replay`
maps 3 to a pass in the recipe:

```make
replay:
	germline replay -was '1=bin/old' -now '2=bin/new' || [ $$? -eq 3 ]
```

Left red, the gate would trap a session between a replay it may not stop on
and a close it may not make. An exit that lets an agent stop and ask is what
cut cheating most in the measurements `docs/prior-art.md` cites.

The exec timeout is 15 minutes, above the longest replay run so far (about
ten minutes on the largest corpus), because a timeout reports as a nonzero
exit and blocks, which is the wrong failure for a slow replay.

`Fuel`, stull's per-session budget of transitions, is left where upstream
set it. It decides how many stops the gate covers, and it bounds a refusal
loop, since every fired transition spends one unit whether green or red.
Raising it would cost a session whose replay is legitimately red one full
replay per refusal.
