package start

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPilotMaintenanceLockBlocksStartup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := rejectPilotMaintenanceMode(); err != nil {
		t.Fatalf("unexpected unlocked error: %v", err)
	}
	root := filepath.Join(home, ".goclaw")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, "pilot-maintenance.lock")
	if err := os.WriteFile(lock, []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := rejectPilotMaintenanceMode(); err == nil {
		t.Fatal("expected maintenance lock to block startup")
	}
}
