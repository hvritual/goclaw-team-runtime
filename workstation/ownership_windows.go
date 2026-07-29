//go:build windows

package workstation

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

func validateCurrentUserOwner(path string, info os.FileInfo) error {
	reparse, err := pathHasReparsePoint(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || reparse {
		return fmt.Errorf("%s must not be a symlink or reparse link", path)
	}
	return validateWindowsWriteACL(path)
}

func validateRootOwner(path string, info os.FileInfo) error {
	return fmt.Errorf(
		"%s cannot satisfy the Unix root-owned strict sandbox contract on Windows",
		path,
	)
}

func validateTrustedPathChain(path string, leafMustBelongToRunner bool) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	current := filepath.Clean(absolute)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		reparse, attributeErr := pathHasReparsePoint(current)
		if attributeErr != nil {
			return attributeErr
		}
		if info.Mode()&os.ModeSymlink != 0 || reparse {
			return fmt.Errorf(
				"%s contains symlink or reparse component %s",
				absolute,
				current,
			)
		}
		if err := validateWindowsWriteACL(current); err != nil {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func validatePrivateFilePermissions(path string, info os.FileInfo) error {
	reparse, err := pathHasReparsePoint(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || reparse {
		return fmt.Errorf("%s must not be a symlink or reparse link", path)
	}
	return validateWindowsWriteACL(path)
}

func pathHasReparsePoint(path string) (bool, error) {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	attributes, err := windows.GetFileAttributes(pointer)
	if err != nil {
		return false, err
	}
	return attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}

func validateWindowsWriteACL(path string) error {
	sd, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return fmt.Errorf("read Windows ACL for %s: %w", path, err)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		return fmt.Errorf("Windows ACL for %s is unavailable", path)
	}
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return err
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return err
	}
	admins, err := windows.CreateWellKnownSid(
		windows.WinBuiltinAdministratorsSid,
	)
	if err != nil {
		return err
	}
	creator, err := windows.CreateWellKnownSid(windows.WinCreatorOwnerSid)
	if err != nil {
		return err
	}
	writeMask := windows.ACCESS_MASK(
		windows.FILE_GENERIC_WRITE |
			windows.GENERIC_WRITE |
			windows.WRITE_DAC |
			windows.WRITE_OWNER |
			windows.DELETE,
	)
	for index := uint32(0); index < uint32(acl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, index, &ace); err != nil {
			return err
		}
		if ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return fmt.Errorf(
				"Windows ACL for %s contains unsupported allow ACE type %d",
				path,
				ace.Header.AceType,
			)
		}
		if ace.Mask&writeMask == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if sid.Equals(user.User.Sid) ||
			sid.Equals(system) ||
			sid.Equals(admins) ||
			sid.Equals(creator) {
			continue
		}
		account, domain, _, lookupErr := sid.LookupAccount("")
		if lookupErr != nil {
			account = "unresolved"
			domain = ""
		}
		principal := account
		if domain != "" {
			principal = domain + `\` + account
		}
		return fmt.Errorf(
			"Windows ACL for %s grants write access to untrusted principal %s",
			path,
			principal,
		)
	}
	return nil
}
