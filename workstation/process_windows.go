//go:build windows

package workstation

import "os/exec"

// Native Windows is a control-CLI-only target in the pilot. Keep the helper so
// cross-compilation remains possible; NewLocalExecutor rejects execution before
// this path can be reached.
func prepareLocalCommand(command *exec.Cmd) {}
