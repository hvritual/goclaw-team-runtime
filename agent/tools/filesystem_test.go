package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemToolUsesPathBoundaries(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "project")
	sibling := filepath.Join(root, "project-secret")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}
	tool := NewFileSystemTool([]string{allowed}, nil, "")
	if !tool.isAllowed(filepath.Join(allowed, "file.txt")) {
		t.Fatal("file inside allowed root should be allowed")
	}
	if tool.isAllowed(filepath.Join(sibling, "secret.txt")) {
		t.Fatal("prefix sibling must not be allowed")
	}
}

func TestFileSystemToolRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	allowed := filepath.Join(root, "project")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(allowed, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	tool := NewFileSystemTool([]string{allowed}, nil, "")
	if tool.isAllowed(filepath.Join(link, "secret.txt")) {
		t.Fatal("symlink escape must not be allowed")
	}
}
