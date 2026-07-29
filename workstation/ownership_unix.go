//go:build !windows

package workstation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func validateCurrentUserOwner(path string, info os.FileInfo) error {
	owner, err := fileOwnerUID(info)
	if err != nil {
		return err
	}
	if owner != uint32(os.Geteuid()) {
		return fmt.Errorf(
			"%s must be owned by runner uid %d, got uid %d",
			path,
			os.Geteuid(),
			owner,
		)
	}
	return nil
}

func validateRootOwner(path string, info os.FileInfo) error {
	owner, err := fileOwnerUID(info)
	if err != nil {
		return err
	}
	if owner != 0 {
		return fmt.Errorf(
			"%s must be root-owned, got uid %d",
			path,
			owner,
		)
	}
	return nil
}

func validatePrivateFilePermissions(path string, info os.FileInfo) error {
	if info.Mode().Perm()&0o077 != 0 {
		return errors.New(
			"device_key_path must not be readable or writable by group or others",
		)
	}
	return nil
}

func validateTrustedPathChain(path string, leafMustBelongToRunner bool) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	current := filepath.Clean(absolute)
	first := true
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s contains symlink component %s", absolute, current)
		}
		owner, err := fileOwnerUID(info)
		if err != nil {
			return err
		}
		if first && leafMustBelongToRunner && owner != uint32(os.Geteuid()) {
			return fmt.Errorf(
				"%s must be owned by runner uid %d, got uid %d",
				current,
				os.Geteuid(),
				owner,
			)
		}
		if owner != 0 && owner != uint32(os.Geteuid()) {
			return fmt.Errorf(
				"%s has untrusted owner uid %d",
				current,
				owner,
			)
		}
		if info.Mode().Perm()&0o022 != 0 &&
			!(owner == 0 && info.Mode()&os.ModeSticky != 0) {
			return fmt.Errorf(
				"%s is writable by group or others",
				current,
			)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
		first = false
	}
}

func fileOwnerUID(info os.FileInfo) (uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("file ownership metadata is unavailable")
	}
	return stat.Uid, nil
}
