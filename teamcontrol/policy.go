package teamcontrol

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

func (s *Service) PutPolicyBundle(
	actorID string,
	input PutPolicyBundleInput,
) (PolicyBundle, error) {
	actorID, err := requireID(actorID, "actor_id")
	if err != nil {
		return PolicyBundle{}, err
	}
	id, err := normalizeID(input.ID, "policy")
	if err != nil {
		return PolicyBundle{}, err
	}
	name, err := requireKey(input.Name, "name")
	if err != nil {
		return PolicyBundle{}, err
	}
	if !validPolicyScope(input.Scope) {
		return PolicyBundle{}, fmt.Errorf("unsupported policy scope %q", input.Scope)
	}
	scopeID, err := requireID(input.ScopeID, "scope_id")
	if err != nil {
		return PolicyBundle{}, err
	}
	if input.Version <= 0 {
		return PolicyBundle{}, fmt.Errorf("version must be positive")
	}
	if input.Priority < -1000 || input.Priority > 1000 {
		return PolicyBundle{}, fmt.Errorf("priority must be between -1000 and 1000")
	}
	rules, err := canonicalRules(input.Rules)
	if err != nil {
		return PolicyBundle{}, err
	}
	if len(rules) == 0 {
		return PolicyBundle{}, fmt.Errorf("at least one policy rule is required")
	}
	var created PolicyBundle
	err = s.store.update(func(st *state) error {
		teamID, projectID, err := authorizePolicyScope(st, actorID, input.Scope, scopeID)
		if err != nil {
			return err
		}
		for _, policy := range st.Policies {
			if policy.Scope == input.Scope && policy.ScopeID == scopeID &&
				policy.Name == name && policy.Version == input.Version &&
				policy.ID != id {
				return conflict(
					"policy %q version %d already exists at this scope",
					name,
					input.Version,
				)
			}
		}
		now := time.Now().UTC()
		created = PolicyBundle{
			ID:        id,
			Name:      name,
			Scope:     input.Scope,
			ScopeID:   scopeID,
			TeamID:    teamID,
			ProjectID: projectID,
			Version:   input.Version,
			Priority:  input.Priority,
			Enabled:   input.Enabled,
			Rules:     rules,
			CreatedBy: actorID,
			CreatedAt: now,
			UpdatedAt: now,
		}
		hash, err := hashPolicyBundle(created)
		if err != nil {
			return err
		}
		created.Hash = hash
		if existing, exists := st.Policies[id]; exists {
			if existing.Hash == created.Hash {
				created = existing
				return nil
			}
			return conflict("policy id %q already identifies a different bundle", id)
		}
		st.Policies[id] = created
		return nil
	})
	return created, err
}

func (s *Service) GetPolicyBundle(
	userID, projectID, policyID string,
) (PolicyBundle, error) {
	policyID, err := requireID(policyID, "policy_id")
	if err != nil {
		return PolicyBundle{}, err
	}
	var result PolicyBundle
	err = s.readProject(userID, projectID, ActionPolicyRead, func(st state, project Project) error {
		policy, ok := st.Policies[policyID]
		if !ok || !policyAppliesToProject(&st, policy, project) {
			return entityNotFound("policy", policyID)
		}
		if err := validateStoredPolicy(policy); err != nil {
			return err
		}
		result = policy
		return nil
	})
	return result, err
}

// ResolvePolicy deterministically applies the latest enabled version of each
// named layer in hierarchy order. repositoryID and componentID are optional,
// but when supplied they must belong to projectID.
func (s *Service) ResolvePolicy(
	userID, projectID, repositoryID, componentID string,
) (ResolvedPolicy, error) {
	repositoryID = strings.TrimSpace(repositoryID)
	componentID = strings.TrimSpace(componentID)
	var result ResolvedPolicy
	err := s.readProject(userID, projectID, ActionPolicyRead, func(st state, project Project) error {
		if repositoryID != "" {
			var err error
			repositoryID, err = requireID(repositoryID, "repository_id")
			if err != nil {
				return err
			}
			repository, ok := st.Repositories[repositoryID]
			if !ok || repository.ProjectID != projectID {
				return entityNotFound("repository", repositoryID)
			}
		}
		if componentID != "" {
			var err error
			componentID, err = requireID(componentID, "component_id")
			if err != nil {
				return err
			}
			component, ok := st.Components[componentID]
			if !ok || component.ProjectID != projectID {
				return entityNotFound("component", componentID)
			}
			if repositoryID != "" && component.RepositoryID != "" &&
				component.RepositoryID != repositoryID {
				return conflict("component does not belong to the requested repository")
			}
		}

		candidates := latestApplicablePolicies(
			st,
			project,
			repositoryID,
			componentID,
		)
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

		result = ResolvedPolicy{
			ProjectID:    projectID,
			RepositoryID: repositoryID,
			ComponentID:  componentID,
			Rules:        make(map[string]json.RawMessage),
		}
		for _, policy := range candidates {
			if err := validateStoredPolicy(policy); err != nil {
				return err
			}
			result.BundleIDs = append(result.BundleIDs, policy.ID)
			result.BundleHashes = append(result.BundleHashes, policy.Hash)
			for key, value := range policy.Rules {
				result.Rules[key] = append(json.RawMessage(nil), value...)
			}
		}
		hash, err := hashResolvedPolicy(result)
		if err != nil {
			return err
		}
		result.Hash = hash
		return nil
	})
	return result, err
}

func canonicalRules(input map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	if len(input) > 500 {
		return nil, fmt.Errorf("policy rules exceed 500 entries")
	}
	output := make(map[string]json.RawMessage, len(input))
	for key, raw := range input {
		key, err := requireKey(key, "policy rule key")
		if err != nil {
			return nil, err
		}
		if len(raw) == 0 {
			return nil, fmt.Errorf("policy rule is empty")
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("policy rule contains invalid JSON: %w", err)
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			if err == nil {
				return nil, fmt.Errorf("policy rule contains trailing JSON")
			}
			return nil, fmt.Errorf("policy rule has invalid trailing JSON: %w", err)
		}
		if text, ok := value.(string); ok &&
			(key == "style" || key == "code_style") {
			value = strings.TrimSpace(text)
		}
		if err := validatePolicyRule(key, value); err != nil {
			return nil, err
		}
		canonical, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		output[key] = canonical
	}
	return output, nil
}

func validatePolicyRule(key string, value any) error {
	switch key {
	case "style", "code_style":
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("policy rule %q must be a string", key)
		}
		if _, err := requireText(text, "policy rule "+key, 100); err != nil {
			return err
		}
	case "max_files", "max_changed_lines":
		number, ok := value.(json.Number)
		if !ok {
			return fmt.Errorf("policy rule %q must be an integer", key)
		}
		parsed, err := number.Int64()
		if err != nil || parsed <= 0 || parsed > 1_000_000 {
			return fmt.Errorf("policy rule %q must be an integer between 1 and 1000000", key)
		}
	case "require_race_test", "require_all_verifications",
		"require_independent_review":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("policy rule %q must be a boolean", key)
		}
	default:
		return fmt.Errorf("unsupported policy rule")
	}
	return nil
}

func validateStoredPolicy(policy PolicyBundle) error {
	if _, err := canonicalRules(policy.Rules); err != nil {
		return fmt.Errorf("stored policy %q failed schema validation: %w", policy.ID, err)
	}
	hash, err := hashPolicyBundle(policy)
	if err != nil {
		return fmt.Errorf("stored policy %q failed hash validation: %w", policy.ID, err)
	}
	if hash != policy.Hash {
		return fmt.Errorf("stored policy %q hash does not match canonical content", policy.ID)
	}
	return nil
}

func authorizePolicyScope(
	st *state,
	actorID string,
	scope PolicyScope,
	scopeID string,
) (teamID, projectID string, err error) {
	switch scope {
	case PolicyTeam:
		if _, err := requireTeamAdmin(st, actorID, scopeID); err != nil {
			return "", "", err
		}
		return scopeID, "", nil
	case PolicyProject:
		project, ok := st.Projects[scopeID]
		if !ok {
			return "", "", entityNotFound("project", scopeID)
		}
		if err := authorizeProject(st, actorID, project.ID, ActionPolicyWrite); err != nil {
			return "", "", err
		}
		return project.TeamID, project.ID, nil
	case PolicyRepository:
		repository, ok := st.Repositories[scopeID]
		if !ok {
			return "", "", entityNotFound("repository", scopeID)
		}
		project := st.Projects[repository.ProjectID]
		if err := authorizeProject(st, actorID, project.ID, ActionPolicyWrite); err != nil {
			return "", "", err
		}
		return project.TeamID, project.ID, nil
	case PolicyComponent:
		component, ok := st.Components[scopeID]
		if !ok {
			return "", "", entityNotFound("component", scopeID)
		}
		project := st.Projects[component.ProjectID]
		if err := authorizeProject(st, actorID, project.ID, ActionPolicyWrite); err != nil {
			return "", "", err
		}
		return project.TeamID, project.ID, nil
	default:
		return "", "", fmt.Errorf("unsupported policy scope %q", scope)
	}
}

func latestApplicablePolicies(
	st state,
	project Project,
	repositoryID, componentID string,
) []PolicyBundle {
	latest := make(map[string]PolicyBundle)
	for _, policy := range st.Policies {
		if !policy.Enabled || !policyAppliesToTarget(
			policy,
			project,
			repositoryID,
			componentID,
		) {
			continue
		}
		key := string(policy.Scope) + ":" + policy.ScopeID + ":" + policy.Name
		existing, exists := latest[key]
		if !exists || policy.Version > existing.Version ||
			(policy.Version == existing.Version && policy.ID < existing.ID) {
			latest[key] = policy
		}
	}
	result := make([]PolicyBundle, 0, len(latest))
	for _, policy := range latest {
		result = append(result, policy)
	}
	return result
}

func policyAppliesToProject(st *state, policy PolicyBundle, project Project) bool {
	switch policy.Scope {
	case PolicyTeam:
		return policy.ScopeID == project.TeamID
	case PolicyProject:
		return policy.ScopeID == project.ID
	case PolicyRepository:
		repository, ok := st.Repositories[policy.ScopeID]
		return ok && repository.ProjectID == project.ID
	case PolicyComponent:
		component, ok := st.Components[policy.ScopeID]
		return ok && component.ProjectID == project.ID
	default:
		return false
	}
}

func policyAppliesToTarget(
	policy PolicyBundle,
	project Project,
	repositoryID, componentID string,
) bool {
	switch policy.Scope {
	case PolicyTeam:
		return policy.ScopeID == project.TeamID
	case PolicyProject:
		return policy.ScopeID == project.ID
	case PolicyRepository:
		return repositoryID != "" && policy.ScopeID == repositoryID
	case PolicyComponent:
		return componentID != "" && policy.ScopeID == componentID
	default:
		return false
	}
}

func policyScopeRank(scope PolicyScope) int {
	switch scope {
	case PolicyTeam:
		return 1
	case PolicyProject:
		return 2
	case PolicyRepository:
		return 3
	case PolicyComponent:
		return 4
	default:
		return 99
	}
}

func hashPolicyBundle(policy PolicyBundle) (string, error) {
	payload := struct {
		ID        string                     `json:"id"`
		Name      string                     `json:"name"`
		Scope     PolicyScope                `json:"scope"`
		ScopeID   string                     `json:"scope_id"`
		TeamID    string                     `json:"team_id"`
		ProjectID string                     `json:"project_id,omitempty"`
		Version   int                        `json:"version"`
		Priority  int                        `json:"priority"`
		Enabled   bool                       `json:"enabled"`
		Rules     map[string]json.RawMessage `json:"rules"`
	}{
		ID:        policy.ID,
		Name:      policy.Name,
		Scope:     policy.Scope,
		ScopeID:   policy.ScopeID,
		TeamID:    policy.TeamID,
		ProjectID: policy.ProjectID,
		Version:   policy.Version,
		Priority:  policy.Priority,
		Enabled:   policy.Enabled,
		Rules:     policy.Rules,
	}
	return hashCanonical(payload)
}

func hashResolvedPolicy(policy ResolvedPolicy) (string, error) {
	payload := struct {
		ProjectID    string                     `json:"project_id"`
		RepositoryID string                     `json:"repository_id,omitempty"`
		ComponentID  string                     `json:"component_id,omitempty"`
		BundleIDs    []string                   `json:"bundle_ids"`
		BundleHashes []string                   `json:"bundle_hashes"`
		Rules        map[string]json.RawMessage `json:"rules"`
	}{
		ProjectID:    policy.ProjectID,
		RepositoryID: policy.RepositoryID,
		ComponentID:  policy.ComponentID,
		BundleIDs:    policy.BundleIDs,
		BundleHashes: policy.BundleHashes,
		Rules:        policy.Rules,
	}
	return hashCanonical(payload)
}

func hashCanonical(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
