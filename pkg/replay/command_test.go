package replay_test

import (
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/justinstimatze/germline/pkg/replay"
)

// TestCommandTimeoutStopsAWrapper is a subject whose child outlives it: the
// shell is killed at the deadline and its sleep keeps stdout open. Before the
// process group and WaitDelay, Run returned when the sleep did, eight seconds
// past a one-second timeout.
func TestCommandTimeoutStopsAWrapper(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	c := &replay.Command{Ver: "1", Path: "sh", Args: []string{"-c", "sleep 8; :"}, Timeout: time.Second}
	start := time.Now()
	if _, err := c.Run(t.Context(), nil); err == nil {
		t.Fatal("a run past its timeout returned no error")
	}
	if took := time.Since(start); took > 4*time.Second {
		t.Fatalf("a one-second timeout took %s to return", took)
	}
}

// TestCommandRefusesOutputPastTheLimit keeps a runaway subject from being
// compared as if its truncated output were the whole of it.
func TestCommandRefusesOutputPastTheLimit(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("head"); err != nil {
		t.Skip("no head")
	}
	c := &replay.Command{Ver: "1", Path: "head", Args: []string{"-c", "4096", "/dev/zero"}, MaxOutput: 1024}
	if _, err := c.Run(t.Context(), nil); !errors.Is(err, replay.ErrOutputTooLarge) {
		t.Fatalf("err = %v, want ErrOutputTooLarge", err)
	}
	c.MaxOutput = 8192
	out, err := c.Run(t.Context(), nil)
	if err != nil || len(out) != 4096 {
		t.Fatalf("under the limit: %d bytes, err = %v", len(out), err)
	}
}
