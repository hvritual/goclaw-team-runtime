package workstation

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReleaseManagerStageActivateConfirmAndRollback(t *testing.T) {
	root := t.TempDir()
	manager, err := NewReleaseManager(ReleaseManagerConfig{
		WorkRoot: root,
		OS:       "linux",
		Arch:     "amd64",
		Protocol: "goclaw.runner/test-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	stage := func(id, version, contents string) LocalReleaseRecord {
		t.Helper()
		path := filepath.Join(t.TempDir(), "goclaw-runner")
		if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
			t.Fatal(err)
		}
		record, err := manager.StageLocal(LocalReleaseInput{
			ReleaseID:  id,
			Version:    version,
			OS:         "linux",
			Arch:       "amd64",
			Protocol:   "goclaw.runner/test-v1",
			SourcePath: path,
			SHA256:     sha256Bytes([]byte(contents)),
			SizeBytes:  int64(len(contents)),
		})
		if err != nil {
			t.Fatalf("StageLocal(%s): %v", id, err)
		}
		return record
	}

	first := stage("release-one", "1.0.0", "first")
	state, err := manager.Activate(first.ReleaseID)
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentReleaseID != first.ReleaseID || !state.PendingHealth ||
		state.CurrentConfirmed {
		t.Fatalf("activation state = %#v", state)
	}
	state, err = manager.Confirm(first.ReleaseID)
	if err != nil {
		t.Fatal(err)
	}
	if state.PendingHealth || !state.CurrentConfirmed {
		t.Fatal("health confirmation remained pending")
	}
	second := stage("release-two", "1.1.0", "second")
	state, err = manager.Activate(second.ReleaseID)
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentReleaseID != second.ReleaseID ||
		state.PreviousReleaseID != first.ReleaseID ||
		!state.PendingHealth ||
		state.CurrentConfirmed ||
		!state.PreviousConfirmed {
		t.Fatalf("second activation state = %#v", state)
	}
	state, err = manager.Rollback()
	if err != nil {
		t.Fatal(err)
	}
	if state.CurrentReleaseID != first.ReleaseID ||
		state.PreviousReleaseID != second.ReleaseID ||
		state.PendingHealth ||
		!state.CurrentConfirmed ||
		state.PreviousConfirmed {
		t.Fatalf("rollback state = %#v", state)
	}
	if _, err := manager.Rollback(); !errors.Is(err, ErrConflict) {
		t.Fatalf("rollback to unconfirmed release error = %v", err)
	}
	path, err := manager.CurrentBinary()
	if err != nil {
		t.Fatal(err)
	}
	if path != first.BinaryPath {
		t.Fatalf("current binary = %q, want %q", path, first.BinaryPath)
	}
}

func TestReleaseManagerFailsClosedOnIdentityAndActiveRunner(t *testing.T) {
	root := t.TempDir()
	manager, err := NewReleaseManager(ReleaseManagerConfig{
		WorkRoot: root,
		OS:       "darwin",
		Arch:     "arm64",
		Protocol: "goclaw.runner/test-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "goclaw-runner")
	if err := os.WriteFile(source, []byte("candidate"), 0o700); err != nil {
		t.Fatal(err)
	}
	base := LocalReleaseInput{
		ReleaseID:  "candidate",
		Version:    "2.0.0",
		OS:         "darwin",
		Arch:       "arm64",
		Protocol:   "goclaw.runner/test-v1",
		SourcePath: source,
		SHA256:     sha256Bytes([]byte("candidate")),
		SizeBytes:  int64(len("candidate")),
	}
	for name, mutate := range map[string]func(*LocalReleaseInput){
		"wrong os":   func(v *LocalReleaseInput) { v.OS = "windows" },
		"wrong arch": func(v *LocalReleaseInput) { v.Arch = "amd64" },
		"wrong size": func(v *LocalReleaseInput) { v.SizeBytes++ },
		"wrong hash": func(v *LocalReleaseInput) { v.SHA256 = sha256Bytes([]byte("other")) },
		"remote uri": func(v *LocalReleaseInput) { v.SourcePath = "https://example.invalid/runner" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if _, err := manager.StageLocal(candidate); err == nil {
				t.Fatal("unsafe release unexpectedly staged")
			}
		})
	}
	record, err := manager.StageLocal(base)
	if err != nil {
		t.Fatal(err)
	}
	release, err := AcquireRunnerProcessLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Activate(record.ReleaseID); !errors.Is(err, ErrConflict) {
		t.Fatalf("active runner activation error = %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Activate(record.ReleaseID); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(record.BinaryPath, []byte("tampered"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Confirm(record.ReleaseID); err == nil {
		t.Fatal("tampered pending release was confirmed")
	}
}

func TestAcquireRunnerProcessLockIsExclusive(t *testing.T) {
	root := t.TempDir()
	release, err := AcquireRunnerProcessLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireRunnerProcessLock(root); !errors.Is(err, ErrConflict) {
		t.Fatalf("second process lock error = %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	second, err := AcquireRunnerProcessLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := second(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseManagerMutationLockIsExclusive(t *testing.T) {
	root := t.TempDir()
	manager, err := NewReleaseManager(ReleaseManagerConfig{
		WorkRoot: root,
		OS:       "linux",
		Arch:     "amd64",
		Protocol: "goclaw.runner/test-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureDir(manager.releasesRoot(), 0o700); err != nil {
		t.Fatal(err)
	}
	release, err := acquireExclusiveFileLock(
		filepath.Join(manager.releasesRoot(), runnerReleaseLockName),
		"test",
	)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "goclaw-runner")
	if err := os.WriteFile(source, []byte("candidate"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.StageLocal(LocalReleaseInput{
		ReleaseID:  "candidate",
		Version:    "1.0.0",
		OS:         "linux",
		Arch:       "amd64",
		Protocol:   "goclaw.runner/test-v1",
		SourcePath: source,
		SHA256:     sha256Bytes([]byte("candidate")),
		SizeBytes:  int64(len("candidate")),
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("concurrent stage error = %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
}
