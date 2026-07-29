package teamcontrol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

const stateFileName = "teamcontrol.json"

type fileStore struct {
	mu    sync.RWMutex
	path  string
	state state
}

func openFileStore(path string) (*fileStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("teamcontrol storage path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(filepath.Ext(absolute), ".json") {
		absolute = filepath.Join(absolute, stateFileName)
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o700); err != nil {
		return nil, fmt.Errorf("create teamcontrol directory: %w", err)
	}
	parentInfo, err := os.Lstat(filepath.Dir(absolute))
	if err != nil {
		return nil, fmt.Errorf("inspect teamcontrol directory: %w", err)
	}
	if !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("teamcontrol state directory must be a real directory")
	}
	if runtime.GOOS != "windows" && parentInfo.Mode().Perm()&0o022 != 0 {
		return nil, fmt.Errorf(
			"teamcontrol directory permissions %04o allow non-owner writes",
			parentInfo.Mode().Perm(),
		)
	}
	if info, statErr := os.Lstat(absolute); statErr == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("teamcontrol state must be a regular file")
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			return nil, fmt.Errorf(
				"teamcontrol state permissions %04o are too broad; require 0600",
				info.Mode().Perm(),
			)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect teamcontrol state: %w", statErr)
	}

	store := &fileStore{path: absolute}
	data, err := os.ReadFile(absolute)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &store.state); err != nil {
			return nil, fmt.Errorf("decode teamcontrol state: %w", err)
		}
		if store.state.SchemaVersion != SchemaVersion {
			return nil, fmt.Errorf(
				"unsupported teamcontrol schema version %d",
				store.state.SchemaVersion,
			)
		}
		beforeNormalize, err := json.Marshal(store.state)
		if err != nil {
			return nil, fmt.Errorf("encode pre-normalized teamcontrol state: %w", err)
		}
		if err := normalizeState(&store.state); err != nil {
			return nil, fmt.Errorf("normalize teamcontrol state: %w", err)
		}
		afterNormalize, err := json.Marshal(store.state)
		if err != nil {
			return nil, fmt.Errorf("encode normalized teamcontrol state: %w", err)
		}
		if !bytes.Equal(beforeNormalize, afterNormalize) {
			if err := writeStateAtomic(absolute, store.state); err != nil {
				return nil, fmt.Errorf("persist normalized teamcontrol state: %w", err)
			}
		}
	case errors.Is(err, os.ErrNotExist):
		store.state = newState()
		if err := writeStateAtomic(absolute, store.state); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("read teamcontrol state: %w", err)
	}
	return store, nil
}

func (s *fileStore) view(fn func(state) error) error {
	s.mu.RLock()
	snapshot, err := cloneState(s.state)
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	return fn(snapshot)
}

// update provides transactional in-process semantics: mutations happen on a
// deep clone and become visible only after the complete JSON snapshot has been
// fsynced and atomically renamed.
func (s *fileStore) update(fn func(*state) error) error {
	return s.updateWithChange(func(next *state) (bool, error) {
		if err := fn(next); err != nil {
			return false, err
		}
		return true, nil
	})
}

// updateWithChange lets idempotent operations avoid changing the revision,
// timestamp, or on-disk bytes when their canonical payload was already stored.
func (s *fileStore) updateWithChange(
	fn func(*state) (bool, error),
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next, err := cloneState(s.state)
	if err != nil {
		return err
	}
	changed, err := fn(&next)
	if err != nil {
		return err
	}
	if !changed || reflect.DeepEqual(s.state, next) {
		return nil
	}
	next.SchemaVersion = SchemaVersion
	next.Revision++
	next.UpdatedAt = time.Now().UTC()
	if err := writeStateAtomic(s.path, next); err != nil {
		return err
	}
	s.state = next
	return nil
}

func cloneState(input state) (state, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return state{}, err
	}
	var output state
	if err := json.Unmarshal(data, &output); err != nil {
		return state{}, err
	}
	if err := normalizeState(&output); err != nil {
		return state{}, err
	}
	return output, nil
}

func normalizeState(value *state) error {
	if value.Users == nil {
		value.Users = make(map[string]User)
	}
	if value.AccessCredentials == nil {
		value.AccessCredentials = make(map[string]AccessCredential)
	}
	if value.Teams == nil {
		value.Teams = make(map[string]Team)
	}
	if value.TeamMemberships == nil {
		value.TeamMemberships = make(map[string]TeamMembership)
	}
	if value.Projects == nil {
		value.Projects = make(map[string]Project)
	}
	if value.ProjectMemberships == nil {
		value.ProjectMemberships = make(map[string]ProjectMembership)
	}
	if value.Repositories == nil {
		value.Repositories = make(map[string]Repository)
	}
	if value.Issues == nil {
		value.Issues = make(map[string]Issue)
	}
	if value.WorkItems == nil {
		value.WorkItems = make(map[string]WorkItem)
	}
	if value.Assignments == nil {
		value.Assignments = make(map[string]Assignment)
	}
	if value.Artifacts == nil {
		value.Artifacts = make(map[string]Artifact)
	}
	if value.Links == nil {
		value.Links = make(map[string]CorrelationLink)
	}
	if value.Documents == nil {
		value.Documents = make(map[string]Document)
	}
	if value.Components == nil {
		value.Components = make(map[string]Component)
	}
	if value.Policies == nil {
		value.Policies = make(map[string]PolicyBundle)
	}
	if value.TokenBudgets == nil {
		value.TokenBudgets = make(map[string]TokenBudget)
	}
	if value.TokenUsageEvents == nil {
		value.TokenUsageEvents = make(map[string]TokenUsageEvent)
	}
	if value.KnowledgeSources == nil {
		value.KnowledgeSources = make(map[string]KnowledgeSource)
	}
	if value.SkillReleases == nil {
		value.SkillReleases = make(map[string]SkillRelease)
	}
	if value.RunnerReleases == nil {
		value.RunnerReleases = make(map[string]RunnerRelease)
	}
	if value.ContextBundles == nil {
		value.ContextBundles = make(map[string]ContextBundle)
	}
	for key, bundle := range value.ContextBundles {
		if bundle.Budget.LegacyUserID != "" {
			bundle.Budget.BudgetUserID = bundle.Budget.LegacyUserID
			if bundle.TargetUserID == "" {
				bundle.TargetUserID = bundle.Budget.LegacyUserID
			}
			bundle.Budget.LegacyUserID = ""
			value.ContextBundles[key] = bundle
		}
	}
	var err error
	value.TokenBudgets, err = normalizeProjectMap(
		value.TokenBudgets,
		func(item TokenBudget) (string, string) { return item.ProjectID, item.ID },
	)
	if err != nil {
		return fmt.Errorf("token budgets: %w", err)
	}
	if err := validateStoredBudgetTotals(value.TokenBudgets); err != nil {
		return err
	}
	value.TokenUsageEvents, err = normalizeProjectMap(
		value.TokenUsageEvents,
		func(item TokenUsageEvent) (string, string) { return item.ProjectID, item.ID },
	)
	if err != nil {
		return fmt.Errorf("token usage events: %w", err)
	}
	value.KnowledgeSources, err = normalizeProjectMap(
		value.KnowledgeSources,
		func(item KnowledgeSource) (string, string) { return item.ProjectID, item.ID },
	)
	if err != nil {
		return fmt.Errorf("knowledge sources: %w", err)
	}
	value.SkillReleases, err = normalizeProjectMap(
		value.SkillReleases,
		func(item SkillRelease) (string, string) { return item.ProjectID, item.ID },
	)
	if err != nil {
		return fmt.Errorf("skill releases: %w", err)
	}
	value.RunnerReleases, err = normalizeProjectMap(
		value.RunnerReleases,
		func(item RunnerRelease) (string, string) { return item.ProjectID, item.ID },
	)
	if err != nil {
		return fmt.Errorf("runner releases: %w", err)
	}
	value.ContextBundles, err = normalizeProjectMap(
		value.ContextBundles,
		func(item ContextBundle) (string, string) { return item.ProjectID, item.ID },
	)
	if err != nil {
		return fmt.Errorf("context bundles: %w", err)
	}
	return nil
}

func projectResourceKey(projectID, id string) string {
	return projectID + "/" + id
}

func validateStoredBudgetTotals(values map[string]TokenBudget) error {
	totals := make(map[string]int64)
	for _, budget := range values {
		if budget.LimitTokens <= 0 || budget.LimitTokens > MaxTokenBudget {
			return fmt.Errorf("token budget %q has an invalid limit", budget.ID)
		}
		if budget.UsedTokens < 0 || budget.UsedTokens > budget.LimitTokens {
			return fmt.Errorf("token budget %q has invalid usage", budget.ID)
		}
		total := totals[budget.ProjectID]
		if budget.LimitTokens > MaxProjectTokenTotal-total {
			return fmt.Errorf(
				"project %q token budget total exceeds JavaScript safe integer",
				budget.ProjectID,
			)
		}
		totals[budget.ProjectID] = total + budget.LimitTokens
	}
	return nil
}

func normalizeProjectMap[T any](
	values map[string]T,
	identity func(T) (projectID, id string),
) (map[string]T, error) {
	result := make(map[string]T, len(values))
	for storedKey, value := range values {
		projectID, id := identity(value)
		if projectID == "" || id == "" {
			return nil, fmt.Errorf("resource %q has an empty project or id", storedKey)
		}
		key := projectResourceKey(projectID, id)
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("multiple resources normalize to %q", key)
		}
		result[key] = value
	}
	return result, nil
}

func writeStateAtomic(path string, value state) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode teamcontrol state: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create teamcontrol temporary file: %w", err)
	}
	tempPath := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(tempPath)
	}
	if err := file.Chmod(0o600); err != nil {
		cleanup()
		return err
	}
	if _, err := file.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write teamcontrol temporary file: %w", err)
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync teamcontrol temporary file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close teamcontrol temporary file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace teamcontrol state: %w", err)
	}
	// Directory fsync makes the rename durable on filesystems that support it.
	if directory, openErr := os.Open(dir); openErr == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}
