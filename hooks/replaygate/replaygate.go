// Package replaygate wires stull's replay-gate into this repository: a session
// may not stop while `make replay` is red.
//
// The machine is stull's own examples/replaygate, imported rather than copied:
// a cell whose Model is the literal "exec", whose Instructions are a command
// line, and whose result is that command's exit code. The judge, the output
// cap, the fail-safe on unparseable output and the fuel bound are upstream's.
//
// Two things are germline's and are added on top.
//
// What the block says. Stull blocks with the failing command's own output,
// which tells a session that replay is red and not what it is allowed to do
// about it. Left at that, the obvious next move is to edit the corpus until
// the difference goes away, and that is the one edit the method forbids.
// Reason names the three-way close instead.
//
// That the gate stands. Upstream's green path lands in a terminal state, which
// is right for a convergence loop and wrong for a gate: the first clean stop
// ends the machine and every later stop in that session sails through
// ungated. Standing turns it into stull's own standing-guard shape — a
// fuel-bounded self-loop with no terminal, which check.Inspect compiles with a
// W-HALT warning rather than an error, and which stull's check_test documents
// as the intended use.
package replaygate

import (
	"fmt"

	upstream "github.com/justinstimatze/stull/examples/replaygate"
	"github.com/justinstimatze/stull/spec"
)

// Instruction is what the gate exists to deliver. A blocked session reads this
// and nothing else about the method, so it names all three closes and the edit
// that is not one of them.
const Instruction = "Close each difference as exactly one of: an implementation " +
	"fix, a law condition, or a boundary entry with a reason recorded as a " +
	"decision. The corpus is never the thing edited. Then stop again."

// Machine returns stull's replay-gate made standing, with Instruction appended
// to the reason its one Block carries.
//
// Fuel is left exactly where upstream set it. Fuel does two jobs at once here:
// it decides how many stops the gate covers, where more is better, and it
// bounds a refusal loop, where fewer is better — the runtime spends one on
// every fired transition, green or red. Upstream's number was chosen for the
// second job, so that a replay which can never go green stops gating loudly
// instead of trapping the session, and standing does not weaken that reason.
// It does change what the number buys: the same fuel now covers that many
// stops instead of one green one. Raising it would cost a session whose replay
// is legitimately red one full replay per refusal, up to about ten minutes on
// the largest corpus replayed so far.
//
// The two structural assumptions about upstream are checked rather than
// assumed, because both are silent when wrong: a wrapper that rewrote the
// wrong effect would install a gate whose refusal says nothing about the
// three-way close, and one that redirected transitions in a machine with more
// than one live state would collapse it into a different machine that still
// compiles.
func Machine() (spec.Machine, error) { return wrap(upstream.Machine()) }

// wrap is Machine's body against a machine passed in, so its two refusals can
// be reached from a test. Machine is the only caller in production and always
// passes upstream's.
func wrap(m spec.Machine) (spec.Machine, error) {
	loop, live := "", 0
	for _, s := range m.States {
		if !s.Terminal {
			live++
			loop = s.Name
		}
	}
	if live != 1 {
		return spec.Machine{}, fmt.Errorf(
			"stull's %s has %d non-terminal states, want exactly 1; there is no single "+
				"state to stand in, so this gate is not installable as written", m.Name, live)
	}

	blocks := 0
	for si := range m.States {
		for ti := range m.States[si].On {
			tr := &m.States[si].On[ti]
			tr.To = loop
			for ei := range tr.Do {
				b, ok := tr.Do[ei].(spec.Block)
				if !ok {
					continue
				}
				blocks++
				inner := b.Reason
				tr.Do[ei] = spec.Block{Reason: func(c *spec.Context) string {
					return spec.Resolve(inner, c) + "\n\n" + Instruction
				}}
			}
		}
	}
	if blocks != 1 {
		return spec.Machine{}, fmt.Errorf(
			"stull's %s has %d Block effects, want exactly 1; the close instruction "+
				"has nowhere unambiguous to go, so this gate is not installable as written",
			m.Name, blocks)
	}

	// Nothing targets the terminal states now, and an unreachable state is
	// E-ORPHAN, which check.Validate refuses — so `install` would fail rather
	// than the gate misbehaving. Drop them.
	standing := make([]spec.State, 0, len(m.States))
	for _, s := range m.States {
		if !s.Terminal {
			standing = append(standing, s)
		}
	}
	m.States = standing

	return m, nil
}
