//go:build !unix

package replay

import "os/exec"

// killGroupOnCancel leaves the default: on timeout the direct child is
// killed, and Command's WaitDelay bounds how long its children may hold the
// pipes open after that.
func killGroupOnCancel(*exec.Cmd) {}
