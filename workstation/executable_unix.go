//go:build !windows

package workstation

import (
	"fmt"
	"os"
)

func validateExecutableFile(path string, info os.FileInfo) error {
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not a regular executable", path)
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("%s is writable by group or others", path)
	}
	return nil
}
