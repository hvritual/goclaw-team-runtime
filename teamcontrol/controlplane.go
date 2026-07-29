package teamcontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"
)

const ContextCompilerVersion = "goclaw-context/v1"
const MaxTokenBudget int64 = 1_000_000_000_000_000
const MaxProjectTokenTotal int64 = 9_007_199_254_740_991

type contextCanonicalMaterial struct {
	CompilerVersion string                `json:"compiler_version"`
	ProjectID       string                `json:"project_id"`
	RepositoryID    string                `json:"repository_id,omitempty"`
	TargetUserID    string                `json:"target_user_id,omitempty"`
	Policy          ResolvedPolicy        `json:"policy"`
	Budget          ContextBudgetSnapshot `json:"budget"`
	Knowledge       []ContextResourceRef  `json:"knowledge"`
	Skills          []ContextResourceRef  `json:"skills"`
}

func (s *Service) PutTokenBudget(
	actorID string,
	input PutTokenBudgetInput,
) (TokenBudget, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return TokenBudget{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return TokenBudget{}, err
	}
	id, err := normalizeID(input.ID, "budget")
	if err != nil {
		return TokenBudget{}, err
	}
	userID := strings.TrimSpace(input.UserID)
	if userID != "" {
		if userID, err = requireID(userID, "user_id"); err != nil {
			return TokenBudget{}, err
		}
	}
	if input.LimitTokens <= 0 {
		return TokenBudget{}, fmt.Errorf("limit_tokens must be positive; zero means unconfigured")
	}
	if input.LimitTokens > MaxTokenBudget {
		return TokenBudget{}, fmt.Errorf("limit_tokens exceeds %d", MaxTokenBudget)
	}
	var result TokenBudget
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionBudgetWrite); err != nil {
			return err
		}
		if userID != "" {
			membership := findProjectMembership(st, projectID, userID)
			if membership == nil || membership.Status != MembershipActive {
				return fmt.Errorf("%w: budget user is not an active project member", ErrForbidden)
			}
		}
		now := time.Now().UTC()
		key := projectResourceKey(projectID, id)
		existing, exists := st.TokenBudgets[key]
		total := input.LimitTokens
		for storedKey, budget := range st.TokenBudgets {
			if storedKey == key || budget.ProjectID != projectID {
				continue
			}
			if budget.LimitTokens > MaxProjectTokenTotal-total {
				return conflict("project token budget total exceeds JavaScript safe integer")
			}
			total += budget.LimitTokens
		}
		if total > MaxProjectTokenTotal {
			return conflict("project token budget total exceeds JavaScript safe integer")
		}
		if exists {
			if existing.UserID != userID {
				return conflict("budget id %q belongs to another scope", id)
			}
			if input.LimitTokens < existing.UsedTokens {
				return conflict("limit_tokens cannot be lower than current usage")
			}
			existing.LimitTokens = input.LimitTokens
			existing.UpdatedBy = actorID
			existing.UpdatedAt = now
			st.TokenBudgets[key] = existing
			result = existing
			return nil
		}
		result = TokenBudget{
			ID:          id,
			ProjectID:   projectID,
			UserID:      userID,
			LimitTokens: input.LimitTokens,
			CreatedBy:   actorID,
			UpdatedBy:   actorID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		st.TokenBudgets[key] = result
		return nil
	})
	return result, err
}

func (s *Service) RecordTokenUsage(
	actorID string,
	input RecordTokenUsageInput,
) (TokenUsageEvent, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return TokenUsageEvent{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return TokenUsageEvent{}, err
	}
	id, err := requireID(input.ID, "event_id")
	if err != nil {
		return TokenUsageEvent{}, err
	}
	budgetID, err := requireID(input.BudgetID, "budget_id")
	if err != nil {
		return TokenUsageEvent{}, err
	}
	if input.Tokens <= 0 {
		return TokenUsageEvent{}, fmt.Errorf("tokens must be positive")
	}
	taskID := strings.TrimSpace(input.TaskID)
	if taskID != "" {
		if taskID, err = requireID(taskID, "task_id"); err != nil {
			return TokenUsageEvent{}, err
		}
	}
	metadata, err := cleanUsageMetadata(input.Metadata)
	if err != nil {
		return TokenUsageEvent{}, err
	}
	var result TokenUsageEvent
	err = s.store.updateWithChange(func(st *state) (bool, error) {
		if err := authorizeProject(st, actorID, projectID, ActionBudgetWrite); err != nil {
			return false, err
		}
		budgetKey := projectResourceKey(projectID, budgetID)
		budget, ok := st.TokenBudgets[budgetKey]
		if !ok {
			return false, entityNotFound("budget", budgetID)
		}
		eventKey := projectResourceKey(projectID, id)
		if existing, exists := st.TokenUsageEvents[eventKey]; exists {
			if existing.BudgetID == budgetID &&
				existing.Tokens == input.Tokens &&
				existing.TaskID == taskID &&
				maps.Equal(existing.Metadata, metadata) {
				result = existing
				return false, nil
			}
			return false, conflict("usage event id %q identifies a different payload", id)
		}
		if budget.UsedTokens > budget.LimitTokens ||
			input.Tokens > budget.LimitTokens-budget.UsedTokens {
			return false, conflict("token budget %q would be exceeded", budgetID)
		}
		now := time.Now().UTC()
		result = TokenUsageEvent{
			ID:         id,
			ProjectID:  projectID,
			BudgetID:   budgetID,
			Tokens:     input.Tokens,
			TaskID:     taskID,
			Metadata:   metadata,
			RecordedBy: actorID,
			RecordedAt: now,
		}
		budget.UsedTokens += input.Tokens
		budget.UpdatedBy = actorID
		budget.UpdatedAt = now
		st.TokenBudgets[budgetKey] = budget
		st.TokenUsageEvents[eventKey] = result
		return true, nil
	})
	return result, err
}

func (s *Service) ListTokenBudgets(userID, projectID string) ([]TokenBudget, error) {
	var result []TokenBudget
	err := s.readProject(userID, projectID, ActionBudgetRead, func(st state, _ Project) error {
		for _, value := range st.TokenBudgets {
			if value.ProjectID == projectID {
				result = append(result, value)
			}
		}
		slices.SortFunc(result, func(a, b TokenBudget) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) ListTokenUsage(
	userID, projectID, budgetID string,
) ([]TokenUsageEvent, error) {
	budgetID = strings.TrimSpace(budgetID)
	var result []TokenUsageEvent
	err := s.readProject(userID, projectID, ActionBudgetRead, func(st state, _ Project) error {
		if budgetID != "" {
			var err error
			if budgetID, err = requireID(budgetID, "budget_id"); err != nil {
				return err
			}
			_, ok := st.TokenBudgets[projectResourceKey(projectID, budgetID)]
			if !ok {
				return entityNotFound("budget", budgetID)
			}
		}
		for _, value := range st.TokenUsageEvents {
			if value.ProjectID == projectID &&
				(budgetID == "" || value.BudgetID == budgetID) {
				if err := validateStoredTokenUsageEvent(st, value); err != nil {
					return err
				}
				result = append(result, value)
			}
		}
		slices.SortFunc(result, func(a, b TokenUsageEvent) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) PutKnowledgeSource(
	actorID string,
	input PutKnowledgeSourceInput,
) (KnowledgeSource, error) {
	projectID, id, name, uri, revision, checksum, status, metadata, err :=
		validateRegistryInput(
			input.ProjectID,
			input.ID,
			"knowledge",
			input.Name,
			input.URI,
			input.Revision,
			input.SHA256,
			input.Status,
			input.Metadata,
		)
	if err != nil {
		return KnowledgeSource{}, err
	}
	actorID, err = requireID(actorID, "actor_id")
	if err != nil {
		return KnowledgeSource{}, err
	}
	var result KnowledgeSource
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionKnowledgeWrite); err != nil {
			return err
		}
		now := time.Now().UTC()
		key := projectResourceKey(projectID, id)
		if existing, ok := st.KnowledgeSources[key]; ok {
			if existing.Name != name ||
				existing.URI != uri || existing.Revision != revision ||
				existing.SHA256 != checksum {
				return conflict("knowledge source id %q identifies different immutable content", id)
			}
			if err := validateRegistryTransition(existing.Status, status); err != nil {
				return err
			}
			existing.Status = status
			existing.Metadata = metadata
			existing.UpdatedBy = actorID
			existing.UpdatedAt = now
			st.KnowledgeSources[key] = existing
			result = existing
			return nil
		}
		result = KnowledgeSource{
			ID: id, ProjectID: projectID, Name: name, URI: uri,
			Revision: revision, SHA256: checksum, Status: status, Metadata: metadata,
			CreatedBy: actorID, UpdatedBy: actorID, CreatedAt: now, UpdatedAt: now,
		}
		st.KnowledgeSources[key] = result
		return nil
	})
	return result, err
}

func (s *Service) ListKnowledgeSources(
	userID, projectID string,
) ([]KnowledgeSource, error) {
	var result []KnowledgeSource
	err := s.readProject(userID, projectID, ActionKnowledgeRead, func(st state, _ Project) error {
		for _, value := range st.KnowledgeSources {
			if value.ProjectID == projectID {
				if err := validateStoredKnowledgeSource(value); err != nil {
					return err
				}
				result = append(result, value)
			}
		}
		slices.SortFunc(result, func(a, b KnowledgeSource) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) GetKnowledgeSource(
	userID, projectID, id string,
) (KnowledgeSource, error) {
	id, err := requireID(id, "knowledge_id")
	if err != nil {
		return KnowledgeSource{}, err
	}
	var result KnowledgeSource
	err = s.readProject(userID, projectID, ActionKnowledgeRead, func(st state, _ Project) error {
		value, ok := st.KnowledgeSources[projectResourceKey(projectID, id)]
		if !ok {
			return entityNotFound("knowledge source", id)
		}
		if err := validateStoredKnowledgeSource(value); err != nil {
			return err
		}
		result = value
		return nil
	})
	return result, err
}

func (s *Service) DeleteKnowledgeSource(actorID, projectID, id string) error {
	return s.deleteRegistry(
		actorID, projectID, id, ActionKnowledgeWrite, "knowledge source",
		func(st *state, key string) (RegistryStatus, bool) {
			value, ok := st.KnowledgeSources[key]
			return value.Status, ok
		},
		func(st *state, key string) { delete(st.KnowledgeSources, key) },
	)
}

func (s *Service) PutSkillRelease(
	actorID string,
	input PutSkillReleaseInput,
) (SkillRelease, error) {
	projectID, id, name, uri, version, checksum, status, metadata, err :=
		validateRegistryInput(
			input.ProjectID,
			input.ID,
			"skill",
			input.Name,
			input.URI,
			input.Version,
			input.SHA256,
			input.Status,
			input.Metadata,
		)
	if err != nil {
		return SkillRelease{}, err
	}
	actorID, err = requireID(actorID, "actor_id")
	if err != nil {
		return SkillRelease{}, err
	}
	minRunner, err := optionalText(input.MinRunnerVersion, "min_runner_version", 100)
	if err != nil {
		return SkillRelease{}, err
	}
	var result SkillRelease
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionSkillWrite); err != nil {
			return err
		}
		now := time.Now().UTC()
		key := projectResourceKey(projectID, id)
		if existing, ok := st.SkillReleases[key]; ok {
			if existing.Name != name ||
				existing.URI != uri || existing.Version != version ||
				existing.SHA256 != checksum ||
				existing.MinRunnerVersion != minRunner {
				return conflict("skill release id %q identifies different immutable content", id)
			}
			if err := validateRegistryTransition(existing.Status, status); err != nil {
				return err
			}
			existing.Status = status
			existing.Metadata = metadata
			existing.UpdatedBy = actorID
			existing.UpdatedAt = now
			st.SkillReleases[key] = existing
			result = existing
			return nil
		}
		result = SkillRelease{
			ID: id, ProjectID: projectID, Name: name, Version: version, URI: uri,
			SHA256: checksum, MinRunnerVersion: minRunner, Status: status,
			Metadata: metadata, CreatedBy: actorID, UpdatedBy: actorID,
			CreatedAt: now, UpdatedAt: now,
		}
		st.SkillReleases[key] = result
		return nil
	})
	return result, err
}

func (s *Service) ListSkillReleases(userID, projectID string) ([]SkillRelease, error) {
	var result []SkillRelease
	err := s.readProject(userID, projectID, ActionSkillRead, func(st state, _ Project) error {
		for _, value := range st.SkillReleases {
			if value.ProjectID == projectID {
				if err := validateStoredSkillRelease(value); err != nil {
					return err
				}
				result = append(result, value)
			}
		}
		slices.SortFunc(result, func(a, b SkillRelease) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) GetSkillRelease(
	userID, projectID, id string,
) (SkillRelease, error) {
	id, err := requireID(id, "skill_id")
	if err != nil {
		return SkillRelease{}, err
	}
	var result SkillRelease
	err = s.readProject(userID, projectID, ActionSkillRead, func(st state, _ Project) error {
		value, ok := st.SkillReleases[projectResourceKey(projectID, id)]
		if !ok {
			return entityNotFound("skill release", id)
		}
		if err := validateStoredSkillRelease(value); err != nil {
			return err
		}
		result = value
		return nil
	})
	return result, err
}

func (s *Service) DeleteSkillRelease(actorID, projectID, id string) error {
	return s.deleteRegistry(
		actorID, projectID, id, ActionSkillWrite, "skill release",
		func(st *state, key string) (RegistryStatus, bool) {
			value, ok := st.SkillReleases[key]
			return value.Status, ok
		},
		func(st *state, key string) { delete(st.SkillReleases, key) },
	)
}

func (s *Service) PutRunnerRelease(
	actorID string,
	input PutRunnerReleaseInput,
) (RunnerRelease, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return RunnerRelease{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return RunnerRelease{}, err
	}
	id, err := normalizeID(input.ID, "runner-release")
	if err != nil {
		return RunnerRelease{}, err
	}
	channel, err := requireKey(strings.ToLower(input.Channel), "channel")
	if err != nil {
		return RunnerRelease{}, err
	}
	version, err := requireText(input.Version, "version", 100)
	if err != nil {
		return RunnerRelease{}, err
	}
	targetOS, err := requireKey(strings.ToLower(input.OS), "os")
	if err != nil {
		return RunnerRelease{}, err
	}
	arch, err := requireKey(strings.ToLower(input.Arch), "arch")
	if err != nil {
		return RunnerRelease{}, err
	}
	uri, err := validateRegistryURI(input.URI, "uri")
	if err != nil {
		return RunnerRelease{}, err
	}
	checksum, err := requireSHA256(input.SHA256)
	if err != nil {
		return RunnerRelease{}, err
	}
	if input.SizeBytes < 0 {
		return RunnerRelease{}, fmt.Errorf("size_bytes cannot be negative")
	}
	minProtocol, err := requireText(input.MinProtocol, "min_protocol", 100)
	if err != nil {
		return RunnerRelease{}, err
	}
	status := input.Status
	if status == "" {
		status = RegistryDraft
	}
	if !validRegistryStatus(status) {
		return RunnerRelease{}, fmt.Errorf("unsupported registry status %q", status)
	}
	var result RunnerRelease
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionRunnerReleaseWrite); err != nil {
			return err
		}
		now := time.Now().UTC()
		key := projectResourceKey(projectID, id)
		if existing, ok := st.RunnerReleases[key]; ok {
			if existing.Channel != channel ||
				existing.Version != version || existing.OS != targetOS ||
				existing.Arch != arch || existing.URI != uri ||
				existing.SHA256 != checksum ||
				existing.SizeBytes != input.SizeBytes ||
				existing.MinProtocol != minProtocol {
				return conflict("runner release id %q identifies different immutable content", id)
			}
			if err := validateRegistryTransition(existing.Status, status); err != nil {
				return err
			}
			existing.Status = status
			existing.UpdatedBy = actorID
			existing.UpdatedAt = now
			st.RunnerReleases[key] = existing
			result = existing
			return nil
		}
		if input.SizeBytes <= 0 {
			return fmt.Errorf("size_bytes must be positive for a new runner release")
		}
		result = RunnerRelease{
			ID: id, ProjectID: projectID, Channel: channel, Version: version,
			OS: targetOS, Arch: arch, URI: uri, SHA256: checksum,
			SizeBytes: input.SizeBytes, MinProtocol: minProtocol,
			Status: status, CreatedBy: actorID,
			UpdatedBy: actorID, CreatedAt: now, UpdatedAt: now,
		}
		st.RunnerReleases[key] = result
		return nil
	})
	return result, err
}

func (s *Service) ListRunnerReleases(
	userID, projectID string,
) ([]RunnerRelease, error) {
	var result []RunnerRelease
	err := s.readProject(userID, projectID, ActionRunnerReleaseRead, func(st state, _ Project) error {
		for _, value := range st.RunnerReleases {
			if value.ProjectID == projectID {
				if err := validateStoredRunnerRelease(value); err != nil {
					return err
				}
				result = append(result, value)
			}
		}
		slices.SortFunc(result, func(a, b RunnerRelease) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func (s *Service) GetRunnerRelease(
	userID, projectID, id string,
) (RunnerRelease, error) {
	id, err := requireID(id, "runner_release_id")
	if err != nil {
		return RunnerRelease{}, err
	}
	var result RunnerRelease
	err = s.readProject(userID, projectID, ActionRunnerReleaseRead, func(st state, _ Project) error {
		value, ok := st.RunnerReleases[projectResourceKey(projectID, id)]
		if !ok {
			return entityNotFound("runner release", id)
		}
		if err := validateStoredRunnerRelease(value); err != nil {
			return err
		}
		result = value
		return nil
	})
	return result, err
}

func (s *Service) DeleteRunnerRelease(actorID, projectID, id string) error {
	return s.deleteRegistry(
		actorID, projectID, id, ActionRunnerReleaseWrite, "runner release",
		func(st *state, key string) (RegistryStatus, bool) {
			value, ok := st.RunnerReleases[key]
			return value.Status, ok
		},
		func(st *state, key string) { delete(st.RunnerReleases, key) },
	)
}

func (s *Service) CompileContext(
	actorID string,
	input CompileContextInput,
) (ContextBundle, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return ContextBundle{}, err
	}
	projectID, err := requireID(input.ProjectID, "project_id")
	if err != nil {
		return ContextBundle{}, err
	}
	repositoryID := strings.TrimSpace(input.RepositoryID)
	if repositoryID != "" {
		if repositoryID, err = requireID(repositoryID, "repository_id"); err != nil {
			return ContextBundle{}, err
		}
	}
	targetUserID := strings.TrimSpace(input.UserID)
	if targetUserID != "" {
		if targetUserID, err = requireID(targetUserID, "user_id"); err != nil {
			return ContextBundle{}, err
		}
	}
	budgetID := strings.TrimSpace(input.BudgetID)
	if budgetID != "" {
		if budgetID, err = requireID(budgetID, "budget_id"); err != nil {
			return ContextBundle{}, err
		}
	}
	knowledgeIDs, err := uniqueIDs(input.KnowledgeIDs, "knowledge_id")
	if err != nil {
		return ContextBundle{}, err
	}
	skillIDs, err := uniqueIDs(input.SkillIDs, "skill_id")
	if err != nil {
		return ContextBundle{}, err
	}
	var result ContextBundle
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionContextCompile); err != nil {
			return err
		}
		project := st.Projects[projectID]
		if repositoryID != "" {
			repository, ok := st.Repositories[repositoryID]
			if !ok || repository.ProjectID != projectID {
				return entityNotFound("repository", repositoryID)
			}
		}
		if targetUserID != "" {
			member := findProjectMembership(st, projectID, targetUserID)
			if member == nil || member.Status != MembershipActive {
				return fmt.Errorf("%w: context user is not an active project member", ErrForbidden)
			}
		}
		budget := ContextBudgetSnapshot{}
		if budgetID != "" {
			value, ok := st.TokenBudgets[projectResourceKey(projectID, budgetID)]
			if !ok {
				return entityNotFound("budget", budgetID)
			}
			if value.UserID != "" && value.UserID != targetUserID {
				return conflict("budget user does not match context user")
			}
			budget = ContextBudgetSnapshot{
				BudgetID: value.ID, BudgetUserID: value.UserID,
				LimitTokens: value.LimitTokens, UsedTokens: value.UsedTokens,
			}
		}
		knowledge := make([]ContextResourceRef, 0, len(knowledgeIDs))
		for _, id := range knowledgeIDs {
			value, ok := st.KnowledgeSources[projectResourceKey(projectID, id)]
			if !ok {
				return entityNotFound("knowledge source", id)
			}
			if err := validateStoredKnowledgeSource(value); err != nil {
				return err
			}
			if value.Status != RegistryApproved || value.SHA256 == "" {
				return conflict("knowledge source %q is not approved with a checksum", id)
			}
			knowledge = append(knowledge, ContextResourceRef{
				ID: value.ID, Name: value.Name, Version: value.Revision,
				URI: value.URI, SHA256: value.SHA256,
			})
		}
		skills := make([]ContextResourceRef, 0, len(skillIDs))
		for _, id := range skillIDs {
			value, ok := st.SkillReleases[projectResourceKey(projectID, id)]
			if !ok {
				return entityNotFound("skill release", id)
			}
			if err := validateStoredSkillRelease(value); err != nil {
				return err
			}
			if value.Status != RegistryApproved || value.SHA256 == "" {
				return conflict("skill release %q is not approved with a checksum", id)
			}
			skills = append(skills, ContextResourceRef{
				ID: value.ID, Name: value.Name, Version: value.Version,
				URI: value.URI, SHA256: value.SHA256,
			})
		}
		policy, err := resolvePolicySnapshot(*st, project, repositoryID)
		if err != nil {
			return err
		}
		material := contextCanonicalMaterial{
			CompilerVersion: ContextCompilerVersion,
			ProjectID:       projectID, RepositoryID: repositoryID,
			TargetUserID: targetUserID,
			Policy:       policy, Budget: budget, Knowledge: knowledge, Skills: skills,
		}
		hash, err := hashContextMaterial(material)
		if err != nil {
			return err
		}
		id := "ctx-" + hash[:32]
		key := projectResourceKey(projectID, id)
		if existing, ok := st.ContextBundles[key]; ok {
			if err := validateStoredContextBundle(existing); err != nil {
				return err
			}
			if existing.Hash != hash {
				return conflict("context bundle id collision")
			}
			result = existing
			return nil
		}
		result = ContextBundle{
			ID: id, ProjectID: projectID, RepositoryID: repositoryID,
			TargetUserID:    targetUserID,
			CompilerVersion: ContextCompilerVersion, Policy: policy, Budget: budget,
			Knowledge: knowledge, Skills: skills, Hash: hash,
			CreatedBy: actorID, CreatedAt: time.Now().UTC(),
		}
		st.ContextBundles[key] = result
		return nil
	})
	return result, err
}

func (s *Service) ListContextBundles(
	userID, projectID string,
) ([]ContextBundle, error) {
	var result []ContextBundle
	err := s.readProject(userID, projectID, ActionContextRead, func(st state, _ Project) error {
		for _, value := range st.ContextBundles {
			if value.ProjectID == projectID {
				if err := validateStoredContextBundle(value); err != nil {
					return err
				}
				result = append(result, value)
			}
		}
		slices.SortFunc(result, func(a, b ContextBundle) int {
			return strings.Compare(a.ID, b.ID)
		})
		return nil
	})
	return result, err
}

func validateRegistryInput(
	projectValue, idValue, idPrefix, nameValue, uriValue, revisionValue,
	checksumValue string,
	status RegistryStatus,
	metadataValue map[string]string,
) (string, string, string, string, string, string, RegistryStatus, map[string]string, error) {
	projectID, err := requireID(projectValue, "project_id")
	if err != nil {
		return "", "", "", "", "", "", "", nil, err
	}
	id, err := normalizeID(idValue, idPrefix)
	if err != nil {
		return "", "", "", "", "", "", "", nil, err
	}
	name, err := requireText(nameValue, "name", 200)
	if err != nil {
		return "", "", "", "", "", "", "", nil, err
	}
	uri, err := validateRegistryURI(uriValue, "uri")
	if err != nil {
		return "", "", "", "", "", "", "", nil, err
	}
	revision, err := requireText(revisionValue, "revision", 200)
	if err != nil {
		return "", "", "", "", "", "", "", nil, err
	}
	checksum, err := requireSHA256(checksumValue)
	if err != nil {
		return "", "", "", "", "", "", "", nil, err
	}
	if status == "" {
		status = RegistryDraft
	}
	if !validRegistryStatus(status) {
		return "", "", "", "", "", "", "", nil,
			fmt.Errorf("unsupported registry status %q", status)
	}
	metadata, err := cleanRegistryMetadata(metadataValue)
	if err != nil {
		return "", "", "", "", "", "", "", nil, err
	}
	return projectID, id, name, uri, revision, checksum, status, metadata, nil
}

func requireSHA256(value string) (string, error) {
	value, err := validateOptionalSHA256(value)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("sha256 is required")
	}
	return value, nil
}

func validateRegistryTransition(from, to RegistryStatus) error {
	if from == to {
		return nil
	}
	switch from {
	case RegistryDraft:
		if to == RegistryApproved || to == RegistryDisabled {
			return nil
		}
	case RegistryApproved:
		if to == RegistryDisabled {
			return nil
		}
	case RegistryDisabled:
		if to == RegistryApproved {
			return nil
		}
	}
	return fmt.Errorf(
		"%w: registry status cannot transition from %q to %q",
		ErrInvalidTransition,
		from,
		to,
	)
}

func resolvePolicySnapshot(
	st state,
	project Project,
	repositoryID string,
) (ResolvedPolicy, error) {
	candidates := latestApplicablePolicies(st, project, repositoryID, "")
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if policyScopeRank(left.Scope) != policyScopeRank(right.Scope) {
			return policyScopeRank(left.Scope) < policyScopeRank(right.Scope)
		}
		if left.Priority != right.Priority {
			return left.Priority < right.Priority
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		if left.Version != right.Version {
			return left.Version < right.Version
		}
		return left.ID < right.ID
	})
	result := ResolvedPolicy{
		ProjectID: project.ID, RepositoryID: repositoryID,
		Rules: make(map[string]json.RawMessage),
	}
	for _, policy := range candidates {
		if err := validateStoredPolicy(policy); err != nil {
			return ResolvedPolicy{}, err
		}
		result.BundleIDs = append(result.BundleIDs, policy.ID)
		result.BundleHashes = append(result.BundleHashes, policy.Hash)
		for key, value := range policy.Rules {
			result.Rules[key] = append(json.RawMessage(nil), value...)
		}
	}
	hash, err := hashResolvedPolicy(result)
	if err != nil {
		return ResolvedPolicy{}, err
	}
	result.Hash = hash
	return result, nil
}

func validateStoredKnowledgeSource(value KnowledgeSource) error {
	_, _, _, _, _, _, _, _, err := validateRegistryInput(
		value.ProjectID, value.ID, "knowledge", value.Name, value.URI,
		value.Revision, value.SHA256, value.Status, value.Metadata,
	)
	if err != nil {
		return fmt.Errorf("stored knowledge source %q failed schema validation: %w", value.ID, err)
	}
	return nil
}

func validateStoredTokenUsageEvent(st state, value TokenUsageEvent) error {
	if _, err := requireID(value.ProjectID, "project_id"); err != nil {
		return fmt.Errorf("stored usage event %q failed schema validation: %w", value.ID, err)
	}
	if _, err := requireID(value.ID, "event_id"); err != nil {
		return fmt.Errorf("stored usage event %q failed schema validation: %w", value.ID, err)
	}
	if _, err := requireID(value.BudgetID, "budget_id"); err != nil {
		return fmt.Errorf("stored usage event %q failed schema validation: %w", value.ID, err)
	}
	if value.Tokens <= 0 {
		return fmt.Errorf("stored usage event %q has non-positive tokens", value.ID)
	}
	if value.TaskID != "" {
		if _, err := requireID(value.TaskID, "task_id"); err != nil {
			return fmt.Errorf("stored usage event %q failed schema validation: %w", value.ID, err)
		}
	}
	if _, ok := st.TokenBudgets[projectResourceKey(value.ProjectID, value.BudgetID)]; !ok {
		return fmt.Errorf("stored usage event %q references a missing project budget", value.ID)
	}
	metadata, err := cleanUsageMetadata(value.Metadata)
	if err != nil || !maps.Equal(metadata, value.Metadata) {
		if err == nil {
			err = fmt.Errorf("metadata is not canonical")
		}
		return fmt.Errorf("stored usage event %q failed metadata schema validation: %w", value.ID, err)
	}
	return nil
}

func validateStoredSkillRelease(value SkillRelease) error {
	_, _, _, _, _, _, _, _, err := validateRegistryInput(
		value.ProjectID, value.ID, "skill", value.Name, value.URI,
		value.Version, value.SHA256, value.Status, value.Metadata,
	)
	if err != nil {
		return fmt.Errorf("stored skill release %q failed schema validation: %w", value.ID, err)
	}
	if _, err := optionalText(value.MinRunnerVersion, "min_runner_version", 100); err != nil {
		return fmt.Errorf("stored skill release %q failed schema validation: %w", value.ID, err)
	}
	return nil
}

func validateStoredRunnerRelease(value RunnerRelease) error {
	if _, err := requireID(value.ProjectID, "project_id"); err != nil {
		return fmt.Errorf("stored runner release %q failed schema validation: %w", value.ID, err)
	}
	if _, err := requireID(value.ID, "runner_release_id"); err != nil {
		return fmt.Errorf("stored runner release %q failed schema validation: %w", value.ID, err)
	}
	validators := []struct {
		value string
		field string
	}{
		{value.Channel, "channel"},
		{value.OS, "os"},
		{value.Arch, "arch"},
	}
	for _, validator := range validators {
		if _, err := requireKey(validator.value, validator.field); err != nil {
			return fmt.Errorf("stored runner release %q failed schema validation: %w", value.ID, err)
		}
	}
	if _, err := requireText(value.Version, "version", 100); err != nil {
		return fmt.Errorf("stored runner release %q failed schema validation: %w", value.ID, err)
	}
	if _, err := validateRegistryURI(value.URI, "uri"); err != nil {
		return fmt.Errorf("stored runner release %q failed schema validation: %w", value.ID, err)
	}
	if _, err := requireSHA256(value.SHA256); err != nil {
		return fmt.Errorf("stored runner release %q failed schema validation: %w", value.ID, err)
	}
	if value.SizeBytes < 0 {
		return fmt.Errorf(
			"stored runner release %q failed schema validation: negative size_bytes",
			value.ID,
		)
	}
	if _, err := requireText(value.MinProtocol, "min_protocol", 100); err != nil {
		return fmt.Errorf("stored runner release %q failed schema validation: %w", value.ID, err)
	}
	if !validRegistryStatus(value.Status) {
		return fmt.Errorf(
			"stored runner release %q failed schema validation: unsupported status",
			value.ID,
		)
	}
	return nil
}

func validateStoredContextBundle(value ContextBundle) error {
	if _, err := requireID(value.ProjectID, "project_id"); err != nil {
		return fmt.Errorf("stored context bundle %q failed schema validation: %w", value.ID, err)
	}
	if _, err := requireID(value.ID, "context_id"); err != nil {
		return fmt.Errorf("stored context bundle %q failed schema validation: %w", value.ID, err)
	}
	if value.CompilerVersion != ContextCompilerVersion {
		return fmt.Errorf("stored context bundle %q uses an unsupported compiler", value.ID)
	}
	for field, id := range map[string]string{
		"repository_id":  value.RepositoryID,
		"target_user_id": value.TargetUserID,
		"budget_id":      value.Budget.BudgetID,
		"budget_user_id": value.Budget.BudgetUserID,
	} {
		if id == "" {
			continue
		}
		if _, err := requireID(id, field); err != nil {
			return fmt.Errorf("stored context bundle %q failed schema validation: %w", value.ID, err)
		}
	}
	if value.Budget.LimitTokens < 0 || value.Budget.UsedTokens < 0 ||
		value.Budget.UsedTokens > value.Budget.LimitTokens {
		return fmt.Errorf("stored context bundle %q has an invalid budget snapshot", value.ID)
	}
	if _, err := canonicalRules(value.Policy.Rules); err != nil {
		return fmt.Errorf("stored context bundle %q failed policy schema validation: %w", value.ID, err)
	}
	policyHash, err := hashResolvedPolicy(value.Policy)
	if err != nil || policyHash != value.Policy.Hash {
		if err == nil {
			err = fmt.Errorf("resolved policy hash mismatch")
		}
		return fmt.Errorf("stored context bundle %q failed policy hash validation: %w", value.ID, err)
	}
	if value.Policy.ProjectID != value.ProjectID ||
		value.Policy.RepositoryID != value.RepositoryID {
		return fmt.Errorf("stored context bundle %q has a cross-scope policy", value.ID)
	}
	for _, resource := range append(
		append([]ContextResourceRef(nil), value.Knowledge...),
		value.Skills...,
	) {
		if _, err := requireID(resource.ID, "context resource id"); err != nil {
			return fmt.Errorf("stored context bundle %q failed schema validation: %w", value.ID, err)
		}
		if _, err := requireText(resource.Name, "context resource name", 200); err != nil {
			return fmt.Errorf("stored context bundle %q failed schema validation: %w", value.ID, err)
		}
		if _, err := requireText(resource.Version, "context resource version", 200); err != nil {
			return fmt.Errorf("stored context bundle %q failed schema validation: %w", value.ID, err)
		}
		if _, err := validateRegistryURI(resource.URI, "context resource uri"); err != nil {
			return fmt.Errorf("stored context bundle %q failed schema validation: %w", value.ID, err)
		}
		if _, err := requireSHA256(resource.SHA256); err != nil {
			return fmt.Errorf("stored context bundle %q failed schema validation: %w", value.ID, err)
		}
	}
	hash, err := hashContextMaterial(contextCanonicalMaterial{
		CompilerVersion: value.CompilerVersion,
		ProjectID:       value.ProjectID,
		RepositoryID:    value.RepositoryID,
		TargetUserID:    value.TargetUserID,
		Policy:          value.Policy,
		Budget:          value.Budget,
		Knowledge:       value.Knowledge,
		Skills:          value.Skills,
	})
	if err != nil {
		return fmt.Errorf("stored context bundle %q failed hash validation: %w", value.ID, err)
	}
	if value.Hash != hash || value.ID != "ctx-"+hash[:32] {
		return fmt.Errorf("stored context bundle %q hash or id does not match canonical content", value.ID)
	}
	return nil
}

func hashContextMaterial(material contextCanonicalMaterial) (string, error) {
	canonical, err := json.Marshal(material)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func (s *Service) deleteRegistry(
	actorID, projectID, id string,
	action Action,
	resourceName string,
	status func(*state, string) (RegistryStatus, bool),
	remove func(*state, string),
) error {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return err
	}
	projectID, err = requireID(projectID, "project_id")
	if err != nil {
		return err
	}
	id, err = requireID(id, resourceName+"_id")
	if err != nil {
		return err
	}
	return s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, action); err != nil {
			return err
		}
		key := projectResourceKey(projectID, id)
		currentStatus, ok := status(st, key)
		if !ok {
			return entityNotFound(resourceName, id)
		}
		if currentStatus == RegistryApproved {
			return conflict("%s %q must be disabled before deletion", resourceName, id)
		}
		remove(st, key)
		return nil
	})
}
