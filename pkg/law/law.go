// Package law makes laws first-class: values with a name, a statement a
// person can read, and an executable form quantified over generated inputs.
//
// A law is never three chosen points. The families below turn one line of
// declaration into a property checked over as many inputs as the config asks
// for, which is the whole reason the second implementation is cheap: the
// generator is untrusted, and the laws are what make that safe.
package law

import (
	"fmt"
	"reflect"
	"testing/quick"

	"github.com/justinstimatze/germline"
)

// A Law is a statement that holds. Name says what holds, not what the test
// does: "ParseRendersBack", never "TestParserField".
type Law struct {
	// Name is the identifier a decision or a boundary entry cites.
	Name string
	// Statement is the law in one sentence, for the reader who has to decide
	// whether a replay difference violates it.
	Statement string
	// Prop is a function returning bool whose arguments testing/quick
	// generates. Every argument type must be generatable: a basic type, or
	// one implementing quick.Generator.
	Prop any
}

// Verify quantifies the law over generated inputs. A nil config uses
// testing/quick's default of 100 inputs.
func (l Law) Verify(cfg *quick.Config) error {
	if err := l.valid(); err != nil {
		return err
	}
	return quick.Check(l.Prop, cfg)
}

func (l Law) valid() error {
	switch {
	case l.Name == "":
		return fmt.Errorf("law has no name: %q", l.Statement)
	case l.Statement == "":
		return fmt.Errorf("law %s has no statement; a law nobody can read is a "+
			"test with ambitions", l.Name)
	case l.Prop == nil:
		return fmt.Errorf("law %s has no executable form; write it as a "+
			"LAW (unchecked): comment instead, so it can be found", l.Name)
	}
	t := reflect.TypeOf(l.Prop)
	if t.Kind() != reflect.Func || t.NumOut() != 1 || t.Out(0).Kind() != reflect.Bool {
		return fmt.Errorf("law %s: Prop is %v, want a func returning bool", l.Name, t)
	}
	return nil
}

// A Set is a collection of laws. A component registers its own into the
// package Default in an init function, and the whole set is then listable,
// countable and runnable without anyone maintaining an index.
type Set struct{ laws []Law }

// Default is the set every package-level Register writes to.
var Default = &Set{}

// Register adds laws to the default set.
func Register(laws ...Law) { Default.Register(laws...) }

// Register adds laws to this set.
func (s *Set) Register(laws ...Law) { s.laws = append(s.laws, laws...) }

// Laws returns the registered laws in registration order.
func (s *Set) Laws() []Law {
	out := make([]Law, len(s.laws))
	copy(out, s.laws)
	return out
}

// Check runs every law and reports each violation against the law's name, so
// a failure names what stopped holding rather than which test failed.
func (s *Set) Check(cfg *quick.Config) germline.Problems {
	var ps germline.Problems
	seen := make(map[string]bool, len(s.laws))
	for _, l := range s.laws {
		if seen[l.Name] {
			ps = append(ps, germline.Problem{
				Artifact: germline.Laws, Where: l.Name,
				What: "two laws share this name; a citation cannot resolve",
			})
		}
		seen[l.Name] = true
		if err := l.Verify(cfg); err != nil {
			ps = append(ps, germline.Problem{
				Artifact: germline.Laws, Where: l.Name,
				What: fmt.Sprintf("%s — violated: %v", l.Statement, err),
			})
		}
	}
	return ps
}

// Roundtrip is the law that rendering and parsing are inverse: whatever a
// value renders to parses back to that value. The one law worth reaching for
// first on any format, because it is the one a regenerated implementation
// breaks first.
func Roundtrip[A comparable](name string, render func(A) string, parse func(string) (A, error)) Law {
	return Law{
		Name:      name,
		Statement: "parsing what a value renders to gives that value back",
		Prop: func(a A) bool {
			b, err := parse(render(a))
			return err == nil && b == a
		},
	}
}

// Idempotent is the law that applying twice is applying once. True of every
// normalizer, formatter and projection, and the cheapest way to catch one
// that is quietly accumulating.
func Idempotent[A comparable](name string, f func(A) A) Law {
	return Law{
		Name:      name,
		Statement: "applying twice is the same as applying once",
		Prop:      func(a A) bool { return f(f(a)) == f(a) },
	}
}

// Involution is the law that applying twice is the identity: undo and redo,
// reverse, negate, toggle.
func Involution[A comparable](name string, f func(A) A) Law {
	return Law{
		Name:      name,
		Statement: "applying twice returns the original",
		Prop:      func(a A) bool { return f(f(a)) == a },
	}
}

// Deterministic is the law that the same input twice gives the same output.
// Worth stating explicitly for anything that touches a map, a clock or a
// goroutine, which is where a regeneration silently stops being a function.
func Deterministic[A any, B comparable](name string, f func(A) B) Law {
	return Law{
		Name:      name,
		Statement: "the same input always gives the same output",
		Prop:      func(a A) bool { return f(a) == f(a) },
	}
}

// Metamorphic is the law that a known change to the input makes a known
// change to the output. It is how greenfield work gets laws before there is
// anything to compare against: you rarely know the right answer, and you
// nearly always know how the answer must move.
//
// vary returns a related input and whether it produced one; when it does not,
// the case is skipped rather than failed.
func Metamorphic[A, B any](name, statement string, vary func(A) (A, bool),
	f func(A) B, holds func(before, after B) bool,
) Law {
	return Law{
		Name:      name,
		Statement: statement,
		Prop: func(a A) bool {
			varied, ok := vary(a)
			if !ok {
				return true
			}
			return holds(f(a), f(varied))
		},
	}
}

// Lens returns the three lens laws for a getter and setter over a
// configuration C and a focus V. GetPut: putting back what you got changes
// nothing. PutGet: you get what you put. PutPut: the last put wins.
//
// The smallest algebra worth owning, and the shape every options struct,
// settings record and nested update turns out to have.
func Lens[C, V comparable](name string, get func(C) V, put func(C, V) C) []Law {
	return []Law{
		{
			Name:      name + "GetPut",
			Statement: "putting back what you got changes nothing",
			Prop:      func(c C) bool { return put(c, get(c)) == c },
		},
		{
			Name:      name + "PutGet",
			Statement: "you get what you put",
			Prop:      func(c C, v V) bool { return get(put(c, v)) == v },
		},
		{
			Name:      name + "PutPut",
			Statement: "the last put wins",
			Prop:      func(c C, a, b V) bool { return put(put(c, a), b) == put(c, b) },
		},
	}
}
