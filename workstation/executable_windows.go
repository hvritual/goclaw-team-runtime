//go:build windows

package workstation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateExecutableFile(path string, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular executable", path)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".com", ".bat", ".cmd":
	default:
		return fmt.Errorf("%s does not have a Windows executable extension", path)
	}
	if err := validateWindowsWriteACL(path); err != nil {
		return err
	}
	return nil
}
