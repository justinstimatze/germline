package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/justinstimatze/germline/internal/project"
	"github.com/justinstimatze/germline/pkg/corpus"
)

// cmdReweight re-estimates the operational profile across the recorded corpus.
//
// This is the one field of a recorded input that is allowed to change, and it
// has to be, because a weight is a claim about how often something happens and
// that claim goes stale while the recording does not. VerifyAppendOnly permits
// it for exactly this reason. Everything else about an input — its bytes, its
// hash, when it was recorded, what version produced what output — stays where
// it was put.
//
// Only the uniform estimate is built in, and it is not really an estimate: it
// is the honest placeholder for a profile nobody has measured, and it says so
// by spreading the weight evenly rather than inventing a shape. A project that
// has measured a profile passes it in a file.
func cmdReweight(e *env, args []string) error {
	fset := flags(e, "reweight")
	uniform := fset.Bool("uniform", false,
		"spread the profile evenly across active inputs; the placeholder for an unmeasured profile")
	at := fset.String("at", "", "the date this estimate was made; empty means today")
	p, err := e.project(fset, args)
	if err != nil {
		return err
	}
	if !*uniform {
		return errors.New("reweight needs an estimate to apply; only -uniform is built in")
	}

	manifestPath := p.Path(project.ManifestFile)
	m, err := corpus.Load(manifestPath)
	if err != nil {
		return err
	}
	active := 0
	for _, in := range m.Inputs {
		if in.Status == corpus.Active {
			active++
		}
	}
	if active == 0 {
		return errors.New("no active inputs to weight")
	}

	// A retired input keeps whatever weight it had. Its weight is part of the
	// record of what the profile looked like when it was retired, and rounding
	// it to zero would quietly rewrite that.
	share := 1.0 / float64(active)
	for i := range m.Inputs {
		if m.Inputs[i].Status == corpus.Active {
			m.Inputs[i].Weight = share
		}
	}
	m.Profile.EstimatedAt = *at
	if m.Profile.EstimatedAt == "" {
		m.Profile.EstimatedAt = time.Now().UTC().Format(time.DateOnly)
	}

	if ps := m.Validate(); !ps.OK() {
		return fmt.Errorf("the manifest would not validate:\n%s", ps.Sorted())
	}
	if saveErr := m.Save(manifestPath); saveErr != nil {
		return saveErr
	}
	fmt.Fprintf(e.stdout, "weighted %s at %.9f each, estimated %s\n",
		plural(active, "active input"), share, m.Profile.EstimatedAt)
	return nil
}
