package laws

import (
	"testing"

	"github.com/justinstimatze/germline/pkg/law"
)

// Config stands in for any value a lens focuses into.
type Config struct {
	Name    string
	Workers int
}

// TestLensLaws checks the three lens laws, GetPut, PutGet and PutPut, from
// Foster et al. 2007 and Bancilhon and Spyratos 1981, over random configs.
//
// The declaration is one call because the family is in pkg/law: naming the
// getter and the setter is the whole input, and three properties quantified
// over generated values come back. It is the property-based form of
// CheckLensLaws in github.com/go-go-golems/optkit, which checks the same three
// laws at three points.
func TestLensLaws(t *testing.T) {
	t.Parallel()
	s := &law.Set{}
	s.Register(law.Lens("Workers",
		func(c Config) int { return c.Workers },
		func(c Config, v int) Config { c.Workers = v; return c },
	)...)
	if ps := s.Check(nil); !ps.OK() {
		t.Error(ps.Sorted())
	}
}
