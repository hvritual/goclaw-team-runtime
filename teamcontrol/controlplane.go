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
		existing, exists := st.TokenBudgets[id]
		if exists {
			if existing.ProjectID != projectID || existing.UserID != userID {
				return conflict("budget id %q belongs to another scope", id)
			}
			if input.LimitTokens < existing.UsedTokens {
				return conflict("limit_tokens cannot be lower than current usage")
			}
			existing.LimitTokens = input.LimitTokens
			existing.UpdatedBy = actorID
			existing.UpdatedAt = now
			st.TokenBudgets[id] = existing
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
		st.TokenBudgets[id] = result
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
	metadata, err := cleanStringMap(input.Metadata)
	if err != nil {
		return TokenUsageEvent{}, err
	}
	var result TokenUsageEvent
	err = s.store.update(func(st *state) error {
		if err := authorizeProject(st, actorID, projectID, ActionBudgetWrite); err != nil {
			return err
		}
		budget, ok := st.TokenBudgets[budgetID]
		if !ok || budget.ProjectID != projectID {
			return entityNotFound("budget", budgetID)
		}
		if existing, exists := st.TokenUsageEvents[id]; exists {
			if existing.ProjectID == projectID &&
				existing.BudgetID == budgetID &&
				existing.Tokens == input.Tokens &&
				existing.TaskID == taskID &&
				maps.Equal(existing.Metadata, metadata) {
				result = existing
				return nil
			}
			return conflict("usage event id %q identifies a different payload", id)
		}
		if budget.UsedTokens > budget.LimitTokens ||
			input.Tokens > budget.LimitTokens-budget.UsedTokens {
			return conflict("token budget %q would be exceeded", budgetID)
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
		st.TokenBudgets[budgetID] = budget
		st.TokenUsageEvents[id] = result
		return nil
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
			budget, ok := st.TokenBudgets[budgetID]
			if !ok || budget.ProjectID != projectID {
				return entityNotFound("budget", budgetID)
			}
		}
		for _, value := range st.TokenUsageEvents {
			if value.ProjectID == projectID &&
				(budgetID == "" || value.BudgetID == budgetID) {
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
		if existing, ok := st.KnowledgeSources[id]; ok {
			if existing.ProjectID != projectID || existing.Name != name ||
				existing.URI != uri || existing.Revision != revision ||
				existing.SHA256 != checksum {
				return conflict("knowledge source id %q identifies different immutable content", id)
			}
			existing.Status = status
			existing.Metadata = metadata
			existing.UpdatedBy = actorID
			existing.UpdatedAt = now
			st.KnowledgeSources[id] = existing
			result = existing
			return nil
		}
		result = KnowledgeSource{
			ID: id, ProjectID: projectID, Name: name, URI: uri,
			Revision: revision, SHA256: checksum, Status: status, Metadata: metadata,
			CreatedBy: actorID, UpdatedBy: actorID, CreatedAt: now, UpdatedAt: now,
		}
		st.KnowledgeSources[id] = result
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
		if existing, ok := st.SkillReleases[id]; ok {
			if existing.ProjectID != projectID || existing.Name != name ||
				existing.URI != uri || existing.Version != version ||
				existing.SHA256 != checksum ||
				existing.MinRunnerVersion != minRunner {
				return conflict("skill release id %q identifies different immutable content", id)
			}
			existing.Status = status
			existing.Metadata = metadata
			existing.UpdatedBy = actorID
			existing.UpdatedAt = now
			st.SkillReleases[id] = existing
			result = existing
			return nil
		}
		result = SkillRelease{
			ID: id, ProjectID: projectID, Name: name, Version: version, URI: uri,
			SHA256: checksum, MinRunnerVersion: minRunner, Status: status,
			Metadata: metadata, CreatedBy: actorID, UpdatedBy: actorID,
			CreatedAt: now, UpdatedAt: now,
		}
		st.SkillReleases[id] = result
		return nil
	})
	return result, err
}

func (s *Service) ListSkillReleases(userID, projectID string) ([]SkillRelease, error) {
	var result []SkillRelease
	err := s.readProject(userID, projectID, ActionSkillRead, func(st state, _ Project) error {
		for _, value := range st.SkillReleases {
			if value.ProjectID == projectID {
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
	uri, err := validateURI(input.URI, "uri")
	if err != nil {
		return RunnerRelease{}, err
	}
	checksum, err := requireSHA256(input.SHA256)
	if err != nil {
		return RunnerRelease{}, err
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
		if existing, ok := st.RunnerReleases[id]; ok {
			if existing.ProjectID != projectID || existing.Channel != channel ||
				existing.Version != version || existing.OS != targetOS ||
				existing.Arch != arch || existing.URI != uri ||
				existing.SHA256 != checksum || existing.MinProtocol != minProtocol {
				return conflict("runner release id %q identifies different immutable content", id)
			}
			existing.Status = status
			existing.UpdatedBy = actorID
			existing.UpdatedAt = now
			st.RunnerReleases[id] = existing
			result = existing
			return nil
		}
		result = RunnerRelease{
			ID: id, ProjectID: projectID, Channel: channel, Version: version,
			OS: targetOS, Arch: arch, URI: uri, SHA256: checksum,
			MinProtocol: minProtocol, Status: status, CreatedBy: actorID,
			UpdatedBy: actorID, CreatedAt: now, UpdatedAt: now,
		}
		st.RunnerReleases[id] = result
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
		budget := ContextBudgetSnapshot{UserID: targetUserID}
		if budgetID != "" {
			value, ok := st.TokenBudgets[budgetID]
			if !ok || value.ProjectID != projectID {
				return entityNotFound("budget", budgetID)
			}
			if value.UserID != "" && value.UserID != targetUserID {
				return conflict("budget user does not match context user")
			}
			budget = ContextBudgetSnapshot{
				BudgetID: value.ID, UserID: value.UserID,
				LimitTokens: value.LimitTokens, UsedTokens: value.UsedTokens,
			}
		}
		knowledge := make([]ContextResourceRef, 0, len(knowledgeIDs))
		for _, id := range knowledgeIDs {
			value, ok := st.KnowledgeSources[id]
			if !ok || value.ProjectID != projectID {
				return entityNotFound("knowledge source", id)
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
			value, ok := st.SkillReleases[id]
			if !ok || value.ProjectID != projectID {
				return entityNotFound("skill release", id)
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
		material := struct {
			CompilerVersion string                `json:"compiler_version"`
			ProjectID       string                `json:"project_id"`
			RepositoryID    string                `json:"repository_id,omitempty"`
			Policy          ResolvedPolicy        `json:"policy"`
			Budget          ContextBudgetSnapshot `json:"budget"`
			Knowledge       []ContextResourceRef  `json:"knowledge"`
			Skills          []ContextResourceRef  `json:"skills"`
		}{
			CompilerVersion: ContextCompilerVersion,
			ProjectID:       projectID, RepositoryID: repositoryID,
			Policy: policy, Budget: budget, Knowledge: knowledge, Skills: skills,
		}
		canonical, err := json.Marshal(material)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(canonical)
		hash := hex.EncodeToString(sum[:])
		id := "ctx-" + hash[:32]
		if existing, ok := st.ContextBundles[id]; ok {
			if existing.Hash != hash {
				return conflict("context bundle id collision")
			}
			result = existing
			return nil
		}
		result = ContextBundle{
			ID: id, ProjectID: projectID, RepositoryID: repositoryID,
			CompilerVersion: ContextCompilerVersion, Policy: policy, Budget: budget,
			Knowledge: knowledge, Skills: skills, Hash: hash,
			CreatedBy: actorID, CreatedAt: time.Now().UTC(),
		}
		st.ContextBundles[id] = result
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
	uri, err := validateURI(uriValue, "uri")
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
	metadata, err := cleanStringMap(metadataValue)
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
