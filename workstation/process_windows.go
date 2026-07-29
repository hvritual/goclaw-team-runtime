//go:build windows

package workstation

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sys/windows"
)

// Windows delegated execution does not provide a sandbox, but cancellation
// must still terminate the process tree so a timed-out verifier cannot keep
// mutating the worktree after Evidence is signed.
func prepareLocalCommand(command *exec.Cmd) {
	command.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}
	command.WaitDelay = 5 * time.Second
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		systemRoot := os.Getenv("SystemRoot")
		taskkill := filepath.Join(systemRoot, "System32", "taskkill.exe")
		if systemRoot != "" {
			killer := exec.Command(
				taskkill,
				"/PID",
				strconv.Itoa(command.Process.Pid),
				"/T",
				"/F",
			)
			if err := killer.Run(); err == nil {
				return nil
			}
		}
		return command.Process.Kill()
	}
}
