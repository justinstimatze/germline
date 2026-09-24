//go:build unix

package replay

import (
	"os/exec"
	"syscall"
)

// killGroupOnCancel runs the subject in its own process group and kills the
// whole group on timeout. exec.CommandContext alone kills only the direct
// child, so a subject that is a wrapper leaves its children running past the
// deadline.
func killGroupOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
