package replaygate

import (
	"strings"
	"testing"

	"github.com/justinstimatze/stull/check"
	upstream "github.com/justinstimatze/stull/examples/replaygate"
	"github.com/justinstimatze/stull/sim"
	"github.com/justinstimatze/stull/spec"
)

// TestMachineIsStandingAndSound pins the shape the wrapper produces: no
// terminal state, one live state, no hard errors, and W-HALT present. The
// warning is the point rather than an accident — stull raises it for exactly
// this shape, a fuel-bounded self-loop with nowhere to halt, so a run that
// stopped raising it would mean the machine had quietly gone back to
// converging and the gate had gone back to covering one stop.
func TestMachineIsStandingAndSound(t *testing.T) {
	t.Parallel()
	m, err := Machine()
	if err != nil {
		t.Fatalf("Machine: %v", err)
	}
	if len(m.States) != 1 || m.States[0].Terminal {
		t.Fatalf("want exactly one non-terminal state, got %d: %+v", len(m.States), m.States)
	}
	for _, s := range m.States {
		for _, tr := range s.On {
			if tr.To != m.Initial {
				t.Errorf("transition on %v targets %q, want the standing state %q",
					tr.On, tr.To, m.Initial)
			}
		}
	}
	rep := check.Inspect(m)
	if !rep.Sound() {
		t.Errorf("standing machine is unsound: %v", rep.Errors)
	}
	if !strings.Contains(strings.Join(rep.Warnings, "\n"), "W-HALT") {
		t.Errorf("want W-HALT on a machine with no terminal state, got %v", rep.Warnings)
	}
}

// TestGateKeepsGatingAfterACleanStop is the behavior the wrapper exists to
// change. Upstream's green path lands in a terminal state, so the first clean
// stop ends the machine and a later red replay in the same session is never
// run. Here the second stop must still be refused.
func TestGateKeepsGatingAfterACleanStop(t *testing.T) {
	t.Parallel()
	m, err := Machine()
	if err != nil {
		t.Fatalf("Machine: %v", err)
	}
	steps, _ := sim.Run(m, sim.Scenario{
		Name:   "clean stop, then a red replay",
		Events: sim.Stops(2),
		Script: map[string][]string{"replay": {"0\n", "1\nFAIL: TestFoo"}},
	})
	if len(steps) != 2 {
		t.Fatalf("ran %d steps, want 2: %+v", len(steps), steps)
	}
	if steps[0].Kind != "inject" {
		t.Errorf("first stop was %q, want inject on a green replay", steps[0].Kind)
	}
	if steps[1].Kind != "block" {
		t.Fatalf("second stop was %q, want block; the gate stopped gating after "+
			"the clean stop, which is upstream's shape and the thing this wrapper undoes",
			steps[1].Kind)
	}
	if !strings.Contains(steps[1].Detail, Instruction) {
		t.Errorf("second stop blocked without the close instruction: %q", steps[1].Detail)
	}
}

// TestBlockCarriesTheCloseInstruction runs upstream's own scenarios against
// the wrapped machine and requires every refusal to name the three-way close
// as well as the failing command's output — a block that says only "replay is
// red" leaves editing the corpus as the obvious next move.
func TestBlockCarriesTheCloseInstruction(t *testing.T) {
	t.Parallel()
	m, err := Machine()
	if err != nil {
		t.Fatalf("Machine: %v", err)
	}
	blocks := 0
	for _, sc := range upstream.Scenarios() {
		steps, _ := sim.Run(m, sc)
		if len(steps) == 0 {
			t.Errorf("%q produced no steps", sc.Name)
		}
		if lint := sim.Lint(steps); len(lint) != 0 {
			t.Errorf("%q is not lint-clean: %v", sc.Name, lint)
		}
		for _, s := range steps {
			if s.Kind != "block" {
				continue
			}
			blocks++
			if !strings.Contains(s.Detail, Instruction) {
				t.Errorf("%q blocked without the close instruction: %q", sc.Name, s.Detail)
			}
			if !strings.Contains(s.Detail, "FAIL: TestFoo") {
				t.Errorf("%q dropped the failing command's own output: %q", sc.Name, s.Detail)
			}
		}
	}
	if blocks == 0 {
		t.Fatal("no scenario blocked; the gate was never exercised")
	}
}

// TestFailSafeStillReleases pins the property that makes this safe to install:
// output stull cannot parse releases the stop rather than trapping the
// session. Standing redirects that transition like every other, so it is worth
// checking it still releases instead of refusing.
func TestFailSafeStillReleases(t *testing.T) {
	t.Parallel()
	m, err := Machine()
	if err != nil {
		t.Fatalf("Machine: %v", err)
	}
	steps, ctx := sim.Run(m, sim.Scenario{
		Name:   "output outside the language",
		Events: sim.Stops(1),
		Script: map[string][]string{"replay": {"garbage, no newline"}},
	})
	if len(steps) != 1 || steps[0].Kind != "inject" {
		t.Fatalf("unparseable output ran %d steps, first kind %q; want one inject",
			len(steps), stepKind(steps))
	}
	if ctx.State != m.Initial {
		t.Errorf("fail-safe left the machine in %q, want the standing state %q",
			ctx.State, m.Initial)
	}
}

// TestRefusesAMachineItCannotWrap covers the two structural assumptions. Both
// fail silently if left unchecked — a machine with no Block installs a gate
// whose refusal says nothing about the three-way close, and one with two live
// states gets collapsed into a different machine that still compiles — so the
// refusal is the whole safety here.
func TestRefusesAMachineItCannotWrap(t *testing.T) {
	t.Parallel()
	block := spec.Block{Reason: spec.S("red")}
	cases := []struct {
		name string
		m    spec.Machine
		want string
	}{
		{
			name: "no block to carry the instruction",
			m: spec.Machine{Name: "silent", Fuel: 2, Initial: "loop", States: []spec.State{
				{Name: "loop", On: []spec.Transition{{On: spec.Stop, To: "loop"}}},
			}},
			want: "0 Block effects",
		},
		{
			name: "two blocks, no unambiguous place for it",
			m: spec.Machine{Name: "twice", Fuel: 2, Initial: "loop", States: []spec.State{
				{Name: "loop", On: []spec.Transition{
					{On: spec.Stop, To: "loop", Do: []spec.Effect{block}},
					{On: spec.Stop, To: "loop", Do: []spec.Effect{block}},
				}},
			}},
			want: "2 Block effects",
		},
		{
			name: "two live states, nowhere single to stand",
			m: spec.Machine{Name: "forked", Fuel: 2, Initial: "a", States: []spec.State{
				{Name: "a", On: []spec.Transition{{On: spec.Stop, To: "b", Do: []spec.Effect{block}}}},
				{Name: "b", On: []spec.Transition{{On: spec.Stop, To: "a"}}},
			}},
			want: "2 non-terminal states",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if _, err := wrap(c.m); err == nil {
				t.Fatalf("wrap accepted %q; want a refusal naming %q", c.m.Name, c.want)
			} else if !strings.Contains(err.Error(), c.want) {
				t.Errorf("refusal was %q, want it to name %q", err, c.want)
			}
		})
	}
}

func stepKind(steps []sim.Step) string {
	if len(steps) == 0 {
		return ""
	}
	return steps[0].Kind
}
