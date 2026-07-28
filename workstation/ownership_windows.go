//go:build windows

package workstation

import "os"

// Native Windows runner execution is disabled during the pilot. These no-op
// helpers keep the cross-platform control CLI buildable; ACL validation belongs
// to a future native Windows execution contract.
func validateCurrentUserOwner(path string, info os.FileInfo) error {
	return nil
}

func validateRootOwner(path string, info os.FileInfo) error {
	return nil
}

func validateTrustedPathChain(path string, leafMustBelongToRunner bool) error {
	return nil
}
