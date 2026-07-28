//go:build !windows

package workstation

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// prepareLocalCommand makes context cancellation terminate the whole Unix
// process group. Codex and test commands routinely create grandchildren, so
// killing only the direct child would leak work after lease loss or shutdown.
func prepareLocalCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	command.WaitDelay = 5 * time.Second
}
