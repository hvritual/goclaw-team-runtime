package workstation

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Service struct {
	cfg Config
	mu  sync.Mutex
	now func() time.Time
}

func DefaultConfig() Config {
	return Config{
		Enabled:                false,
		LeaseDurationSeconds:   120,
		RunnerOfflineSeconds:   300,
		DefaultMaxAttempts:     3,
		MaxIdempotencyReceipts: 128,
	}
}

func NewService(cfg Config) (*Service, error) {
	if strings.TrimSpace(cfg.Root) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.Root = filepath.Join(home, ".goclaw", "workstation")
	}
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}
	cfg.Root = root
	defaults := DefaultConfig()
	if cfg.LeaseDurationSeconds <= 0 {
		cfg.LeaseDurationSeconds = defaults.LeaseDurationSeconds
	}
	if cfg.RunnerOfflineSeconds <= 0 {
		cfg.RunnerOfflineSeconds = defaults.RunnerOfflineSeconds
	}
	if cfg.DefaultMaxAttempts <= 0 {
		cfg.DefaultMaxAttempts = defaults.DefaultMaxAttempts
	}
	if cfg.MaxIdempotencyReceipts <= 0 {
		cfg.MaxIdempotencyReceipts = defaults.MaxIdempotencyReceipts
	}
	service := &Service{
		cfg: cfg,
		now: func() time.Time { return time.Now().UTC() },
	}
	if err := service.Ensure(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) Config() Config {
	return s.cfg
}

func (s *Service) Ensure() error {
	for _, dir := range []string{
		s.cfg.Root,
		s.tasksDir(),
		s.runnersDir(),
		s.credentialsDir(),
		s.evidenceDir(),
	} {
		if err := ensureDir(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) tasksDir() string       { return filepath.Join(s.cfg.Root, "tasks") }
func (s *Service) runnersDir() string     { return filepath.Join(s.cfg.Root, "runners") }
func (s *Service) credentialsDir() string { return filepath.Join(s.cfg.Root, "credentials") }
func (s *Service) evidenceDir() string    { return filepath.Join(s.cfg.Root, "evidence") }

func (s *Service) taskPath(id string) (string, error) {
	if err := validateID(id); err != nil {
		return "", err
	}
	return safeJoin(s.tasksDir(), id+".json")
}

func (s *Service) runnerPath(id string) (string, error) {
	if err := validateID(id); err != nil {
		return "", err
	}
	return safeJoin(s.runnersDir(), id+".json")
}

func (s *Service) credentialPath(id string) (string, error) {
	if err := validateID(id); err != nil {
		return "", err
	}
	return safeJoin(s.credentialsDir(), id+".key")
}

func (s *Service) archivedCredentialPath(runnerID, keyID string) (string, error) {
	if err := validateID(runnerID); err != nil {
		return "", err
	}
	if len(keyID) != 64 {
		return "", errors.New("invalid archived credential key id")
	}
	return safeJoin(
		s.credentialsDir(),
		filepath.Join("archive", runnerID, keyID+".key"),
	)
}

func (s *Service) evidencePath(taskID, digest string) (string, error) {
	if err := validateID(taskID); err != nil {
		return "", err
	}
	if len(digest) != 64 {
		return "", errors.New("invalid evidence digest")
	}
	return safeJoin(s.evidenceDir(), filepath.Join(taskID, digest+".json"))
}

func (s *Service) loadTaskUnlocked(id string) (Task, error) {
	path, err := s.taskPath(id)
	if err != nil {
		return Task{}, err
	}
	var task Task
	if err := readJSON(path, &task); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Task{}, fmt.Errorf("%w: task %s", ErrNotFound, id)
		}
		return Task{}, err
	}
	if task.SchemaVersion != SchemaVersion || task.ID != id {
		return Task{}, fmt.Errorf("%w: invalid task projection %s", ErrConflict, id)
	}
	return task, nil
}

func (s *Service) saveTaskUnlocked(task Task) error {
	path, err := s.taskPath(task.ID)
	if err != nil {
		return err
	}
	return writeJSONAtomic(path, task, 0o600)
}

func (s *Service) loadRunnerUnlocked(id string) (Runner, error) {
	path, err := s.runnerPath(id)
	if err != nil {
		return Runner{}, err
	}
	var runner Runner
	if err := readJSON(path, &runner); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Runner{}, fmt.Errorf("%w: runner %s", ErrNotFound, id)
		}
		return Runner{}, err
	}
	if runner.SchemaVersion != SchemaVersion || runner.ID != id {
		return Runner{}, fmt.Errorf("%w: invalid runner projection %s", ErrConflict, id)
	}
	return runner, nil
}

func (s *Service) saveRunnerUnlocked(runner Runner) error {
	path, err := s.runnerPath(runner.ID)
	if err != nil {
		return err
	}
	return writeJSONAtomic(path, runner, 0o600)
}

func (s *Service) loadCredentialUnlocked(runnerID string) ([]byte, error) {
	path, err := s.credentialPath(runnerID)
	if err != nil {
		return nil, err
	}
	key, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: credential for runner %s", ErrNotFound, runnerID)
	}
	if err != nil {
		return nil, err
	}
	if _, err := DeviceKeyID(key); err != nil {
		return nil, fmt.Errorf("%w: malformed runner credential: %v", ErrConflict, err)
	}
	return key, nil
}

func (s *Service) loadCredentialByKeyIDUnlocked(
	runnerID, keyID string,
) ([]byte, error) {
	active, activeErr := s.loadCredentialUnlocked(runnerID)
	if activeErr == nil {
		activeID, err := DeviceKeyID(active)
		if err != nil {
			return nil, err
		}
		if activeID == keyID {
			return active, nil
		}
	} else if !errors.Is(activeErr, ErrNotFound) {
		return nil, activeErr
	}
	path, err := s.archivedCredentialPath(runnerID, keyID)
	if err != nil {
		return nil, err
	}
	archived, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf(
			"%w: credential key %s for runner %s",
			ErrNotFound,
			keyID,
			runnerID,
		)
	}
	if err != nil {
		return nil, err
	}
	archivedID, err := DeviceKeyID(archived)
	if err != nil {
		return nil, err
	}
	if archivedID != keyID {
		return nil, fmt.Errorf("%w: archived credential digest mismatch", ErrConflict)
	}
	return archived, nil
}

func (s *Service) listTasksUnlocked() ([]Task, error) {
	entries, err := os.ReadDir(s.tasksDir())
	if err != nil {
		return nil, err
	}
	result := make([]Task, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		task, err := s.loadTaskUnlocked(id)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority > result[j].Priority
		}
		if !result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (s *Service) listRunnersUnlocked() ([]Runner, error) {
	entries, err := os.ReadDir(s.runnersDir())
	if err != nil {
		return nil, err
	}
	result := make([]Runner, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		runner, err := s.loadRunnerUnlocked(id)
		if err != nil {
			return nil, err
		}
		result = append(result, runner)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
