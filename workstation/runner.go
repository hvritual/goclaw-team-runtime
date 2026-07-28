package workstation

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// RegisterRunner provisions a runner using a device key supplied or generated
// out of band by the caller. The key is persisted separately from public runner
// and task JSON.
func (s *Service) RegisterRunner(request RegisterRunnerRequest, deviceKey []byte) (Runner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateID(request.ID); err != nil {
		return Runner{}, err
	}
	keyID, err := DeviceKeyID(deviceKey)
	if err != nil {
		return Runner{}, err
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		request.Name = request.ID
	}
	request.OwnerUserID = strings.TrimSpace(request.OwnerUserID)
	if request.OwnerUserID == "" {
		return Runner{}, errors.New("runner owner_user_id is required")
	}
	request.Capabilities = normalizeCapabilities(request.Capabilities)
	request.Projects = normalizeStrings(request.Projects)
	if len(request.Projects) == 0 {
		return Runner{}, errors.New("runner requires at least one authorized project")
	}
	if existing, err := s.loadRunnerUnlocked(request.ID); err == nil {
		if existing.KeyID != keyID ||
			existing.Name != request.Name ||
			existing.OwnerUserID != request.OwnerUserID ||
			!equalStrings(existing.Capabilities, request.Capabilities) ||
			!equalStrings(existing.Projects, request.Projects) ||
			!equalMetadata(existing.Metadata, request.Metadata) {
			return Runner{}, fmt.Errorf("%w: runner %s is already registered with different attributes", ErrConflict, request.ID)
		}
		storedKey, keyErr := s.loadCredentialUnlocked(request.ID)
		if keyErr != nil {
			return Runner{}, keyErr
		}
		storedKeyID, _ := DeviceKeyID(storedKey)
		if storedKeyID != keyID {
			return Runner{}, fmt.Errorf("%w: runner credential does not match registration", ErrConflict)
		}
		return cloneJSON(existing)
	} else if !errors.Is(err, ErrNotFound) {
		return Runner{}, err
	}
	now := s.now()
	runner := Runner{
		SchemaVersion:   SchemaVersion,
		ID:              request.ID,
		Name:            request.Name,
		OwnerUserID:     request.OwnerUserID,
		Status:          RunnerOnline,
		Capabilities:    request.Capabilities,
		Projects:        request.Projects,
		KeyID:           keyID,
		Metadata:        cloneMetadata(request.Metadata),
		RegisteredAt:    now,
		LastHeartbeatAt: now,
		UpdatedAt:       now,
	}
	keyPath, err := s.credentialPath(request.ID)
	if err != nil {
		return Runner{}, err
	}
	if _, err := os.Stat(keyPath); err == nil {
		return Runner{}, fmt.Errorf("%w: orphan credential already exists for runner %s", ErrConflict, request.ID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Runner{}, err
	}
	if err := writeBytesAtomic(keyPath, append([]byte(nil), deviceKey...), 0o600); err != nil {
		return Runner{}, err
	}
	if err := s.saveRunnerUnlocked(runner); err != nil {
		// Registration is not visible until the public runner projection is
		// durable. Remove the newly-created credential on failure so a retry is
		// not blocked by an orphaned key.
		_ = os.Remove(keyPath)
		return Runner{}, err
	}
	return cloneJSON(runner)
}

func (s *Service) UpdateRunner(id string, request UpdateRunnerRequest) (Runner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	runner, err := s.loadRunnerUnlocked(id)
	if err != nil {
		return Runner{}, err
	}
	if request.Capabilities != nil ||
		request.Projects != nil ||
		request.Disabled != nil {
		tasks, err := s.listTasksUnlocked()
		if err != nil {
			return Runner{}, err
		}
		for _, task := range tasks {
			if task.Status == TaskLeased &&
				task.Lease != nil &&
				task.Lease.RunnerID == runner.ID {
				return Runner{}, fmt.Errorf(
					"%w: runner execution profile cannot change while lease %s is active",
					ErrConflict,
					task.Lease.ID,
				)
			}
		}
	}
	if strings.TrimSpace(request.Name) != "" {
		runner.Name = strings.TrimSpace(request.Name)
	}
	if request.Capabilities != nil {
		runner.Capabilities = normalizeCapabilities(request.Capabilities)
	}
	if request.Projects != nil {
		projects := normalizeStrings(request.Projects)
		if len(projects) == 0 {
			return Runner{}, errors.New("runner requires at least one authorized project")
		}
		runner.Projects = projects
	}
	if request.Metadata != nil {
		runner.Metadata = cloneMetadata(request.Metadata)
	}
	if request.Disabled != nil {
		if *request.Disabled {
			runner.Status = RunnerDisabled
		} else {
			runner.Status = RunnerOffline
		}
	}
	runner.UpdatedAt = s.now()
	if err := s.saveRunnerUnlocked(runner); err != nil {
		return Runner{}, err
	}
	return cloneJSON(runner)
}

// RotateRunnerDeviceKey replaces the evidence-signing key only while the
// runner has no active lease. The prior key is restored if the public runner
// projection cannot be persisted.
func (s *Service) RotateRunnerDeviceKey(id string, deviceKey []byte) (Runner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	runner, err := s.loadRunnerUnlocked(id)
	if err != nil {
		return Runner{}, err
	}
	keyID, err := DeviceKeyID(deviceKey)
	if err != nil {
		return Runner{}, err
	}
	tasks, err := s.listTasksUnlocked()
	if err != nil {
		return Runner{}, err
	}
	for _, task := range tasks {
		if task.Status == TaskLeased && task.Lease != nil &&
			task.Lease.RunnerID == runner.ID {
			return Runner{}, fmt.Errorf(
				"%w: cannot rotate key while runner holds lease %s",
				ErrConflict,
				task.Lease.ID,
			)
		}
	}
	currentKey, err := s.loadCredentialUnlocked(id)
	if err != nil {
		return Runner{}, err
	}
	currentID, err := DeviceKeyID(currentKey)
	if err != nil {
		return Runner{}, err
	}
	if currentID == keyID {
		return cloneJSON(runner)
	}
	archivePath, err := s.archivedCredentialPath(id, currentID)
	if err != nil {
		return Runner{}, err
	}
	if err := writeBytesAtomic(archivePath, currentKey, 0o600); err != nil {
		return Runner{}, fmt.Errorf("archive prior runner credential: %w", err)
	}
	keyPath, err := s.credentialPath(id)
	if err != nil {
		return Runner{}, err
	}
	if err := writeBytesAtomic(keyPath, append([]byte(nil), deviceKey...), 0o600); err != nil {
		return Runner{}, err
	}
	runner.KeyID = keyID
	runner.UpdatedAt = s.now()
	if err := s.saveRunnerUnlocked(runner); err != nil {
		_ = writeBytesAtomic(keyPath, currentKey, 0o600)
		return Runner{}, err
	}
	return cloneJSON(runner)
}

func (s *Service) HeartbeatRunner(id string) (Runner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	runner, err := s.loadRunnerUnlocked(id)
	if err != nil {
		return Runner{}, err
	}
	if runner.Status == RunnerDisabled {
		return Runner{}, fmt.Errorf("%w: runner %s is disabled", ErrUnauthorized, id)
	}
	now := s.now()
	runner.Status = RunnerOnline
	runner.LastHeartbeatAt = now
	runner.UpdatedAt = now
	if err := s.saveRunnerUnlocked(runner); err != nil {
		return Runner{}, err
	}
	return cloneJSON(runner)
}

func (s *Service) GetRunner(id string) (Runner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return Runner{}, err
	}
	runner, err := s.loadRunnerUnlocked(id)
	if err != nil {
		return Runner{}, err
	}
	return cloneJSON(runner)
}

func (s *Service) ListRunners() ([]Runner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.recoverExpiredUnlocked(s.now()); err != nil {
		return nil, err
	}
	runners, err := s.listRunnersUnlocked()
	if err != nil {
		return nil, err
	}
	return cloneJSON(runners)
}

func runnerAuthorized(runner Runner, projectID string, requiredCapabilities []string) bool {
	if runner.Status == RunnerDisabled || !containsString(runner.Projects, projectID, false) &&
		!containsString(runner.Projects, "*", false) {
		return false
	}
	for _, required := range normalizeCapabilities(requiredCapabilities) {
		if !containsString(runner.Capabilities, required, true) &&
			!containsString(runner.Capabilities, "*", true) {
			return false
		}
	}
	return true
}

func normalizeCapabilities(values []string) []string {
	result := normalizeStrings(values)
	for index := range result {
		result[index] = strings.ToLower(result[index])
	}
	sort.Strings(result)
	return uniqueSorted(result)
}

func normalizeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	sort.Strings(result)
	return uniqueSorted(result)
}

func uniqueSorted(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func containsString(values []string, expected string, fold bool) bool {
	for _, value := range values {
		if value == expected || fold && strings.EqualFold(value, expected) {
			return true
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalMetadata(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func cloneMetadata(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}
