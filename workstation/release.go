package workstation

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	RunnerReleaseSchemaVersion = 1
	RunnerReleaseProtocol      = "1"
	maxRunnerReleaseBytes      = 256 * 1024 * 1024
	runnerProcessLockName      = ".runner-process.lock"
	runnerReleaseLockName      = ".runner-release.lock"
)

type LocalReleaseInput struct {
	ReleaseID  string `json:"release_id"`
	Version    string `json:"version"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	Protocol   string `json:"protocol"`
	SourcePath string `json:"source_path"`
	SHA256     string `json:"sha256"`
	SizeBytes  int64  `json:"size_bytes"`
}

type LocalReleaseRecord struct {
	SchemaVersion int       `json:"schema_version"`
	ReleaseID     string    `json:"release_id"`
	Version       string    `json:"version"`
	OS            string    `json:"os"`
	Arch          string    `json:"arch"`
	Protocol      string    `json:"protocol"`
	BinaryPath    string    `json:"binary_path"`
	SHA256        string    `json:"sha256"`
	SizeBytes     int64     `json:"size_bytes"`
	StagedAt      time.Time `json:"staged_at"`
}

type LocalReleaseState struct {
	SchemaVersion     int       `json:"schema_version"`
	CurrentReleaseID  string    `json:"current_release_id,omitempty"`
	PreviousReleaseID string    `json:"previous_release_id,omitempty"`
	PendingHealth     bool      `json:"pending_health"`
	CurrentConfirmed  bool      `json:"current_confirmed"`
	PreviousConfirmed bool      `json:"previous_confirmed"`
	Generation        uint64    `json:"generation"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type RunnerLifecycleProjection struct {
	CurrentVersion   string           `json:"current_version"`
	CurrentReleaseID string           `json:"current_release_id,omitempty"`
	ReleaseProtocol  string           `json:"release_protocol"`
	ExecutionProfile ExecutionProfile `json:"execution_profile"`
}

type ReleaseManagerConfig struct {
	WorkRoot   string
	OS         string
	Arch       string
	Protocol   string
	BinaryName string
}

type ReleaseManager struct {
	cfg ReleaseManagerConfig
	now func() time.Time
}

func NewReleaseManager(cfg ReleaseManagerConfig) (*ReleaseManager, error) {
	if strings.TrimSpace(cfg.WorkRoot) == "" {
		return nil, errors.New("release work_root is required")
	}
	root, err := filepath.Abs(cfg.WorkRoot)
	if err != nil {
		return nil, err
	}
	if err := ensureDir(root, 0o700); err != nil {
		return nil, err
	}
	if err := validateTrustedPathChain(root, true); err != nil {
		return nil, fmt.Errorf("release work_root ownership: %w", err)
	}
	cfg.WorkRoot = root
	cfg.OS = strings.ToLower(strings.TrimSpace(cfg.OS))
	if cfg.OS == "" {
		cfg.OS = runtime.GOOS
	}
	cfg.Arch = strings.ToLower(strings.TrimSpace(cfg.Arch))
	if cfg.Arch == "" {
		cfg.Arch = runtime.GOARCH
	}
	cfg.Protocol = strings.TrimSpace(cfg.Protocol)
	if cfg.Protocol == "" {
		cfg.Protocol = RunnerReleaseProtocol
	}
	cfg.BinaryName = strings.TrimSpace(cfg.BinaryName)
	if cfg.BinaryName == "" {
		cfg.BinaryName = "goclaw-runner"
		if cfg.OS == "windows" {
			cfg.BinaryName += ".exe"
		}
	}
	if filepath.Base(cfg.BinaryName) != cfg.BinaryName ||
		cfg.BinaryName == "." || cfg.BinaryName == ".." {
		return nil, errors.New("release binary_name must be a file name")
	}
	return &ReleaseManager{cfg: cfg, now: func() time.Time {
		return time.Now().UTC()
	}}, nil
}

func (m *ReleaseManager) StageLocal(input LocalReleaseInput) (LocalReleaseRecord, error) {
	input.ReleaseID = strings.TrimSpace(input.ReleaseID)
	if err := validateID(input.ReleaseID); err != nil {
		return LocalReleaseRecord{}, fmt.Errorf("release_id: %w", err)
	}
	input.Version = strings.TrimSpace(input.Version)
	if input.Version == "" || len(input.Version) > 100 {
		return LocalReleaseRecord{}, errors.New("release version is required and must not exceed 100 bytes")
	}
	input.OS = strings.ToLower(strings.TrimSpace(input.OS))
	input.Arch = strings.ToLower(strings.TrimSpace(input.Arch))
	input.Protocol = strings.TrimSpace(input.Protocol)
	if input.OS != m.cfg.OS || input.Arch != m.cfg.Arch {
		return LocalReleaseRecord{}, fmt.Errorf(
			"release target %s/%s does not match runner %s/%s",
			input.OS,
			input.Arch,
			m.cfg.OS,
			m.cfg.Arch,
		)
	}
	if input.Protocol != m.cfg.Protocol {
		return LocalReleaseRecord{}, fmt.Errorf(
			"release protocol %q does not match runner protocol",
			input.Protocol,
		)
	}
	checksum, err := normalizeReleaseSHA256(input.SHA256)
	if err != nil {
		return LocalReleaseRecord{}, err
	}
	input.SourcePath = strings.TrimSpace(input.SourcePath)
	if !filepath.IsAbs(input.SourcePath) {
		return LocalReleaseRecord{}, errors.New(
			"release source_path must be an absolute local path",
		)
	}
	source, err := filepath.Abs(input.SourcePath)
	if err != nil {
		return LocalReleaseRecord{}, err
	}
	info, err := os.Lstat(source)
	if err != nil {
		return LocalReleaseRecord{}, err
	}
	if !info.Mode().IsRegular() {
		return LocalReleaseRecord{}, errors.New("release source must be a regular file")
	}
	if info.Size() <= 0 || info.Size() > maxRunnerReleaseBytes {
		return LocalReleaseRecord{}, fmt.Errorf(
			"release size must be between 1 and %d bytes",
			maxRunnerReleaseBytes,
		)
	}
	if input.SizeBytes != info.Size() {
		return LocalReleaseRecord{}, fmt.Errorf(
			"release size mismatch: expected %d bytes, got %d",
			input.SizeBytes,
			info.Size(),
		)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return LocalReleaseRecord{}, err
	}
	if int64(len(data)) != input.SizeBytes ||
		sha256Bytes(data) != checksum {
		return LocalReleaseRecord{}, errors.New(
			"release identity changed while staging",
		)
	}
	if err := ensureDir(m.releasesRoot(), 0o700); err != nil {
		return LocalReleaseRecord{}, err
	}
	unlock, err := acquireExclusiveFileLock(
		filepath.Join(m.releasesRoot(), runnerReleaseLockName),
		"runner release mutation",
	)
	if err != nil {
		return LocalReleaseRecord{}, err
	}
	defer func() { _ = unlock() }()
	releaseDir, err := safeJoin(
		m.releasesRoot(),
		input.ReleaseID,
	)
	if err != nil {
		return LocalReleaseRecord{}, err
	}
	if _, err := os.Stat(releaseDir); err == nil {
		existing, loadErr := m.loadRecord(input.ReleaseID)
		if loadErr != nil {
			return LocalReleaseRecord{}, fmt.Errorf(
				"%w: existing release is invalid",
				ErrConflict,
			)
		}
		if existing.Version != input.Version ||
			existing.OS != input.OS ||
			existing.Arch != input.Arch ||
			existing.Protocol != input.Protocol ||
			existing.SHA256 != checksum ||
			existing.SizeBytes != input.SizeBytes {
			return LocalReleaseRecord{}, fmt.Errorf(
				"%w: release id identifies different immutable content",
				ErrConflict,
			)
		}
		if err := m.verifyRecord(existing); err != nil {
			return LocalReleaseRecord{}, err
		}
		return existing, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return LocalReleaseRecord{}, err
	}
	stageDir, err := os.MkdirTemp(
		m.releasesRoot(),
		".runner-release-stage-*",
	)
	if err != nil {
		return LocalReleaseRecord{}, err
	}
	cleanupStage := func() { _ = os.RemoveAll(stageDir) }
	stageBinaryPath := filepath.Join(stageDir, m.cfg.BinaryName)
	if err := writeBytesAtomic(stageBinaryPath, data, 0o700); err != nil {
		cleanupStage()
		return LocalReleaseRecord{}, err
	}
	written, err := sha256File(stageBinaryPath)
	if err != nil || written != checksum {
		cleanupStage()
		return LocalReleaseRecord{}, errors.New("staged release SHA-256 mismatch")
	}
	binaryPath := filepath.Join(releaseDir, m.cfg.BinaryName)
	record := LocalReleaseRecord{
		SchemaVersion: RunnerReleaseSchemaVersion,
		ReleaseID:     input.ReleaseID,
		Version:       input.Version,
		OS:            input.OS,
		Arch:          input.Arch,
		Protocol:      input.Protocol,
		BinaryPath:    binaryPath,
		SHA256:        checksum,
		SizeBytes:     input.SizeBytes,
		StagedAt:      m.now(),
	}
	if err := writeJSONAtomic(
		filepath.Join(stageDir, "release.json"),
		record,
		0o600,
	); err != nil {
		cleanupStage()
		return LocalReleaseRecord{}, err
	}
	if err := os.Rename(stageDir, releaseDir); err != nil {
		cleanupStage()
		return LocalReleaseRecord{}, err
	}
	if root, err := os.Open(m.releasesRoot()); err == nil {
		_ = root.Sync()
		_ = root.Close()
	}
	return record, nil
}

func (m *ReleaseManager) Activate(releaseID string) (LocalReleaseState, error) {
	unlock, err := m.acquireMutationLock()
	if err != nil {
		return LocalReleaseState{}, err
	}
	defer func() { _ = unlock() }()
	if err := m.requireRunnerIdle(); err != nil {
		return LocalReleaseState{}, err
	}
	record, err := m.loadRecord(releaseID)
	if err != nil {
		return LocalReleaseState{}, err
	}
	if err := m.verifyRecord(record); err != nil {
		return LocalReleaseState{}, err
	}
	state, err := m.Status()
	if err != nil {
		return LocalReleaseState{}, err
	}
	if state.CurrentReleaseID == record.ReleaseID {
		return state, nil
	}
	if state.PendingHealth {
		return LocalReleaseState{}, fmt.Errorf(
			"%w: current release still awaits health confirmation",
			ErrConflict,
		)
	}
	state.PreviousReleaseID = state.CurrentReleaseID
	state.PreviousConfirmed = state.CurrentConfirmed
	state.CurrentReleaseID = record.ReleaseID
	state.CurrentConfirmed = false
	state.PendingHealth = true
	state.Generation++
	state.UpdatedAt = m.now()
	if err := m.writeState(state); err != nil {
		return LocalReleaseState{}, err
	}
	return state, nil
}

func (m *ReleaseManager) Confirm(releaseID string) (LocalReleaseState, error) {
	unlock, err := m.acquireMutationLock()
	if err != nil {
		return LocalReleaseState{}, err
	}
	defer func() { _ = unlock() }()
	if err := m.requireRunnerIdle(); err != nil {
		return LocalReleaseState{}, err
	}
	state, err := m.Status()
	if err != nil {
		return LocalReleaseState{}, err
	}
	if state.CurrentReleaseID != strings.TrimSpace(releaseID) ||
		!state.PendingHealth {
		return LocalReleaseState{}, fmt.Errorf(
			"%w: release is not pending health confirmation",
			ErrConflict,
		)
	}
	record, err := m.loadRecord(state.CurrentReleaseID)
	if err != nil {
		return LocalReleaseState{}, err
	}
	if err := m.verifyRecord(record); err != nil {
		return LocalReleaseState{}, err
	}
	state.PendingHealth = false
	state.CurrentConfirmed = true
	state.Generation++
	state.UpdatedAt = m.now()
	if err := m.writeState(state); err != nil {
		return LocalReleaseState{}, err
	}
	return state, nil
}

func (m *ReleaseManager) Rollback() (LocalReleaseState, error) {
	unlock, err := m.acquireMutationLock()
	if err != nil {
		return LocalReleaseState{}, err
	}
	defer func() { _ = unlock() }()
	if err := m.requireRunnerIdle(); err != nil {
		return LocalReleaseState{}, err
	}
	state, err := m.Status()
	if err != nil {
		return LocalReleaseState{}, err
	}
	if state.PreviousReleaseID == "" {
		return LocalReleaseState{}, fmt.Errorf(
			"%w: no previous verified release",
			ErrConflict,
		)
	}
	if !state.PreviousConfirmed {
		return LocalReleaseState{}, fmt.Errorf(
			"%w: previous release has not passed health confirmation",
			ErrConflict,
		)
	}
	previous, err := m.loadRecord(state.PreviousReleaseID)
	if err != nil {
		return LocalReleaseState{}, err
	}
	if err := m.verifyRecord(previous); err != nil {
		return LocalReleaseState{}, err
	}
	state.CurrentReleaseID, state.PreviousReleaseID =
		state.PreviousReleaseID, state.CurrentReleaseID
	state.CurrentConfirmed, state.PreviousConfirmed =
		state.PreviousConfirmed, state.CurrentConfirmed
	state.PendingHealth = !state.CurrentConfirmed
	state.Generation++
	state.UpdatedAt = m.now()
	if err := m.writeState(state); err != nil {
		return LocalReleaseState{}, err
	}
	return state, nil
}

func (m *ReleaseManager) Status() (LocalReleaseState, error) {
	var state LocalReleaseState
	if err := readJSON(m.statePath(), &state); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LocalReleaseState{
				SchemaVersion: RunnerReleaseSchemaVersion,
			}, nil
		}
		return LocalReleaseState{}, err
	}
	if state.SchemaVersion != RunnerReleaseSchemaVersion {
		return LocalReleaseState{}, errors.New("unsupported local release state schema")
	}
	for _, id := range []string{
		state.CurrentReleaseID,
		state.PreviousReleaseID,
	} {
		if id != "" {
			if err := validateID(id); err != nil {
				return LocalReleaseState{}, errors.New("invalid local release state")
			}
		}
	}
	if state.CurrentReleaseID == "" {
		if state.PreviousReleaseID != "" || state.PendingHealth ||
			state.CurrentConfirmed || state.PreviousConfirmed {
			return LocalReleaseState{}, errors.New(
				"invalid empty local release state",
			)
		}
	} else if state.PendingHealth == state.CurrentConfirmed {
		return LocalReleaseState{}, errors.New(
			"local release health state is inconsistent",
		)
	}
	if state.PreviousReleaseID == "" && state.PreviousConfirmed {
		return LocalReleaseState{}, errors.New(
			"local previous release health state is inconsistent",
		)
	}
	return state, nil
}

func (m *ReleaseManager) CurrentBinary() (string, error) {
	state, err := m.Status()
	if err != nil {
		return "", err
	}
	if state.CurrentReleaseID == "" {
		return "", ErrNotFound
	}
	record, err := m.loadRecord(state.CurrentReleaseID)
	if err != nil {
		return "", err
	}
	if err := m.verifyRecord(record); err != nil {
		return "", err
	}
	return record.BinaryPath, nil
}

func (m *ReleaseManager) LifecycleProjection(
	binaryVersion string,
	profile ExecutionProfile,
) (RunnerLifecycleProjection, error) {
	projection := RunnerLifecycleProjection{
		CurrentVersion:   strings.TrimSpace(binaryVersion),
		ReleaseProtocol:  m.cfg.Protocol,
		ExecutionProfile: profile,
	}
	state, err := m.Status()
	if err != nil {
		return RunnerLifecycleProjection{}, err
	}
	if state.CurrentReleaseID == "" {
		return projection, nil
	}
	record, err := m.loadRecord(state.CurrentReleaseID)
	if err != nil {
		return RunnerLifecycleProjection{}, err
	}
	if err := m.verifyRecord(record); err != nil {
		return RunnerLifecycleProjection{}, err
	}
	projection.CurrentVersion = record.Version
	projection.CurrentReleaseID = record.ReleaseID
	return projection, nil
}

func AcquireRunnerProcessLock(workRoot string) (func() error, error) {
	root, err := filepath.Abs(strings.TrimSpace(workRoot))
	if err != nil {
		return nil, err
	}
	if err := ensureDir(root, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(root, runnerProcessLockName)
	return acquireExclusiveFileLock(path, "runner process")
}

func acquireExclusiveFileLock(
	path, purpose string,
) (func() error, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf(
				"%w: %s lock already exists",
				ErrConflict,
				purpose,
			)
		}
		return nil, err
	}
	if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return func() error { return os.Remove(path) }, nil
}

func (m *ReleaseManager) acquireMutationLock() (func() error, error) {
	if err := ensureDir(m.releasesRoot(), 0o700); err != nil {
		return nil, err
	}
	return acquireExclusiveFileLock(
		filepath.Join(m.releasesRoot(), runnerReleaseLockName),
		"runner release mutation",
	)
}

func (m *ReleaseManager) loadRecord(releaseID string) (LocalReleaseRecord, error) {
	releaseID = strings.TrimSpace(releaseID)
	if err := validateID(releaseID); err != nil {
		return LocalReleaseRecord{}, err
	}
	path, err := safeJoin(
		m.releasesRoot(),
		filepath.Join(releaseID, "release.json"),
	)
	if err != nil {
		return LocalReleaseRecord{}, err
	}
	var record LocalReleaseRecord
	if err := readJSON(path, &record); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LocalReleaseRecord{}, ErrNotFound
		}
		return LocalReleaseRecord{}, err
	}
	if record.SchemaVersion != RunnerReleaseSchemaVersion ||
		record.ReleaseID != releaseID ||
		record.OS != m.cfg.OS ||
		record.Arch != m.cfg.Arch ||
		record.Protocol != m.cfg.Protocol {
		return LocalReleaseRecord{}, errors.New("stored local release failed validation")
	}
	expectedPath, err := safeJoin(
		m.releasesRoot(),
		filepath.Join(releaseID, m.cfg.BinaryName),
	)
	if err != nil || record.BinaryPath != expectedPath {
		return LocalReleaseRecord{}, errors.New("stored local release path failed validation")
	}
	return record, nil
}

func (m *ReleaseManager) verifyRecord(record LocalReleaseRecord) error {
	info, err := os.Lstat(record.BinaryPath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != record.SizeBytes {
		return errors.New("staged release file identity mismatch")
	}
	actual, err := sha256File(record.BinaryPath)
	if err != nil {
		return err
	}
	if actual != record.SHA256 {
		return errors.New("staged release SHA-256 mismatch")
	}
	return nil
}

func (m *ReleaseManager) writeState(state LocalReleaseState) error {
	state.SchemaVersion = RunnerReleaseSchemaVersion
	return writeJSONAtomic(m.statePath(), state, 0o600)
}

func (m *ReleaseManager) requireRunnerIdle() error {
	info, err := os.Lstat(
		filepath.Join(m.cfg.WorkRoot, runnerProcessLockName),
	)
	if err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("runner process lock is not a regular file")
		}
		return fmt.Errorf(
			"%w: runner must be stopped before release activation",
			ErrConflict,
		)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *ReleaseManager) releasesRoot() string {
	return filepath.Join(m.cfg.WorkRoot, ".runner-releases")
}

func (m *ReleaseManager) statePath() string {
	return filepath.Join(m.releasesRoot(), "state.json")
}

func normalizeReleaseSHA256(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return "", errors.New("release SHA-256 must be 64 hexadecimal characters")
	}
	return value, nil
}
