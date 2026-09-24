// Package laws holds the sample system's laws. Every one runs against every
// implementation of the Codec port, which is what makes this a conformance
// suite rather than a test of one codec: the port has two implementations, so
// the laws belong to the port and not to either of them.
package laws

import (
	"errors"
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/justinstimatze/germline/examples/querystring"
	"github.com/justinstimatze/germline/pkg/law"
)

// allLaws is the suite: everything that must hold of any codec. It is a
// function of the codec rather than a fixed set so that the same laws check
// both shipped implementations and any perturbation a boundary witness
// installs. witness_test.go depends on that.
func allLaws(c querystring.Codec) []law.Law {
	return []law.Law{
		{
			Name:      "QueryRoundTrips",
			Statement: "parsing what a query renders to gives that query back",
			Prop: func(g genQuery) bool {
				parsed, err := c.Parse(c.Render(g.q))
				return err == nil && parsed.Canonical() == g.q.Canonical()
			},
		},
		{
			Name:      "RenderIsIdempotentThroughParse",
			Statement: "rendering a query twice through a parse gives the same bytes",
			Prop: func(g genQuery) bool {
				once := c.Render(g.q)
				parsed, err := c.Parse(once)
				return err == nil && c.Render(parsed) == once
			},
		},
		{
			Name:      "KeyOrderSurvives",
			Statement: "keys come back in the order they were first seen",
			Prop: func(g genQuery) bool {
				parsed, err := c.Parse(c.Render(g.q))
				return err == nil && slices.Equal(parsed.Keys, g.q.Keys)
			},
		},
		{
			Name:      "RepeatedValuesKeepTheirOrder",
			Statement: "repeated keys keep their values in the order they appeared",
			Prop: func(g genQuery) bool {
				parsed, err := c.Parse(c.Render(g.q))
				if err != nil {
					return false
				}
				for _, k := range g.q.Keys {
					if !slices.Equal(parsed.Values[k], g.q.Values[k]) {
						return false
					}
				}
				return true
			},
		},
	}
}

// The roundtrip law is written out rather than declared with law.Roundtrip
// because that family needs a comparable type and a Query holds a map. The
// families cover the common shapes; the rest is an ordinary Law value, which
// is the point of making a law a value rather than a framework hook.
//
// KeyOrderSurvives is not redundant with the roundtrip law: Canonical sorts
// its keys, so a map-backed parser that lost first-seen order would still
// round-trip. That is the sort of gap a corpus replay finds late and a second
// law finds now.

// violations runs the suite against one codec and returns what broke.
func violations(c querystring.Codec) error {
	s := &law.Set{}
	s.Register(allLaws(c)...)
	if ps := s.Check(nil); !ps.OK() {
		return errors.New(ps.Sorted().String())
	}
	return nil
}

// TestEveryCodecObeysEveryLaw is the conformance suite: two implementations
// of one port, checked against the port's laws rather than against each
// other's output.
func TestEveryCodecObeysEveryLaw(t *testing.T) {
	t.Parallel()
	for _, c := range querystring.Codecs() {
		t.Run(c.Name(), func(t *testing.T) {
			t.Parallel()
			if err := violations(c); err != nil {
				t.Error(err)
			}
		})
	}
}

// genQuery wraps a Query so testing/quick can generate one. Values are drawn
// from an alphabet that includes the characters the two versions disagree
// about, because a generator that avoided them would let both implementations
// pass every law and prove nothing.
type genQuery struct{ q querystring.Query }

const alphabet = "abcXYZ019 +%&=/?#[]~-_."

// Generate builds a query with zero to four keys. Note what the bound is not:
// testing/quick calls this with a constant size of 50, so an expression like
// rnd.Intn(size%5+1) is rnd.Intn(1), which is always zero. That version of
// this generator produced nothing but empty queries and every law below
// passed vacuously; the witness check in witness_test.go is what caught it,
// by noticing that reversing key order broke no law.
func (genQuery) Generate(rnd *rand.Rand, _ int) reflect.Value {
	var q querystring.Query
	for range rnd.Intn(5) {
		q.Add(word(rnd, 1+rnd.Intn(6)), word(rnd, rnd.Intn(8)))
	}
	return reflect.ValueOf(genQuery{q})
}

func word(rnd *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rnd.Intn(len(alphabet))]
	}
	return string(b)
}
