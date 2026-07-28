package orchestratorlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const waveRegistryPath = "docs/waves/wave-registry.json"

type waveRegistry struct {
	SchemaVersion int                  `json:"schema_version"`
	ActiveWave    string               `json:"active_wave"`
	Waves         []waveRegistryRecord `json:"waves"`
}

type waveRegistryRecord struct {
	ID                        string   `json:"id"`
	Status                    string   `json:"status"`
	Document                  string   `json:"document"`
	DependsOn                 []string `json:"depends_on"`
	AllowedChangeScope        []string `json:"allowed_change_scope"`
	ProductCodeChangesAllowed bool     `json:"product_code_changes_allowed"`
}

type wavePlanFrontMatter struct {
	Schema                    string   `yaml:"schema"`
	WaveID                    string   `yaml:"wave_id"`
	Revision                  int      `yaml:"revision"`
	PlanStatus                string   `yaml:"plan_status"`
	WaveState                 string   `yaml:"wave_state"`
	ApprovedBy                []string `yaml:"approved_by"`
	DependsOn                 []string `yaml:"depends_on"`
	Steps                     []string `yaml:"steps"`
	AllowedChangeScope        []string `yaml:"allowed_change_scope"`
	ProductCodeChangesAllowed bool     `yaml:"product_code_changes_allowed"`
}

// ResolveWaveBinding resolves baseRef to an exact commit and derives the
// immutable Wave binding from repository-owned governance files at that
// commit. Callers provide only the intended step ID; all other authority is
// reconstructed from Git and must not be accepted from an untrusted client.
func ResolveWaveBinding(
	ctx context.Context,
	repoPath string,
	baseRef string,
	stepID string,
) (WaveBinding, string, error) {
	baseRef = strings.TrimSpace(baseRef)
	if baseRef == "" {
		return WaveBinding{}, "", errors.New("base_ref is required")
	}
	stepID = strings.TrimSpace(stepID)
	if err := validateID(stepID); err != nil {
		return WaveBinding{}, "", fmt.Errorf("wave step_id: %w", err)
	}
	base, err := runGit(
		ctx,
		repoPath,
		"rev-parse",
		"--verify",
		baseRef+"^{commit}",
	)
	if err != nil {
		return WaveBinding{}, "", fmt.Errorf(
			"resolve base ref %q: %w: %s",
			baseRef,
			err,
			strings.TrimSpace(base.Stderr),
		)
	}
	baseCommit := strings.ToLower(strings.TrimSpace(base.Stdout))
	if !validGitCommitReference(baseCommit) {
		return WaveBinding{}, "", errors.New(
			"resolved base commit is not a hexadecimal Git commit id",
		)
	}

	registryData, err := gitBlobAtBase(
		ctx,
		repoPath,
		baseCommit,
		waveRegistryPath,
	)
	if err != nil {
		return WaveBinding{}, "", fmt.Errorf(
			"read wave registry at resolved base: %w",
			err,
		)
	}
	var registry waveRegistry
	if err := json.Unmarshal(registryData, &registry); err != nil {
		return WaveBinding{}, "", fmt.Errorf("decode wave registry: %w", err)
	}
	record, err := validateActiveWaveRegistry(registry, registry.ActiveWave)
	if err != nil {
		return WaveBinding{}, "", err
	}
	planPath, err := registryPlanPath(record.Document)
	if err != nil {
		return WaveBinding{}, "", err
	}
	planData, err := gitBlobAtBase(ctx, repoPath, baseCommit, planPath)
	if err != nil {
		return WaveBinding{}, "", fmt.Errorf(
			"read wave plan at resolved base: %w",
			err,
		)
	}
	plan, err := decodeWavePlanFrontMatter(planData)
	if err != nil {
		return WaveBinding{}, "", err
	}
	binding := WaveBinding{
		WaveID:         record.ID,
		PlanRevision:   plan.Revision,
		StepID:         stepID,
		PlanPath:       planPath,
		RegistrySHA256: sha256Bytes(registryData),
		PlanSHA256:     sha256Bytes(planData),
	}
	if err := validateWaveBindingShape(binding); err != nil {
		return WaveBinding{}, "", err
	}
	if err := validateWavePlan(plan, binding, record); err != nil {
		return WaveBinding{}, "", err
	}
	if err := validateWaveDependencies(registry, record.DependsOn); err != nil {
		return WaveBinding{}, "", err
	}
	return binding, baseCommit, nil
}

// ValidateWaveBinding revalidates a frozen task against the exact registry and
// plan blobs in its frozen Git base. Gate callers should invoke this again
// immediately before enqueue so a stored task cannot bypass the freeze-time
// decision.
func (s *Service) ValidateWaveBinding(ctx context.Context, task Task) error {
	if strings.TrimSpace(task.Compile.BaseCommit) == "" {
		return errors.New("task has no frozen base commit")
	}
	return validateWaveBindingAtBase(ctx, task, task.Compile.BaseCommit)
}

func validateWaveBindingAtBase(
	ctx context.Context,
	task Task,
	baseCommit string,
) error {
	baseCommit = strings.ToLower(strings.TrimSpace(baseCommit))
	if !validGitCommitReference(baseCommit) {
		return errors.New("frozen base commit must be a hexadecimal Git commit id")
	}
	if task.Wave == nil {
		return errors.New("wave binding is required")
	}
	binding := *task.Wave
	if err := validateWaveBindingShape(binding); err != nil {
		return err
	}
	if len(task.IssueIDs) == 0 {
		return errors.New("at least one issue_id is required by the wave gate")
	}
	if len(task.Scope.AllowedPaths) == 0 {
		return errors.New("allowed_paths is required by the wave gate")
	}

	registryData, err := gitBlobAtBase(
		ctx,
		task.RepoPath,
		baseCommit,
		waveRegistryPath,
	)
	if err != nil {
		return fmt.Errorf("read wave registry at frozen base: %w", err)
	}
	if actual := sha256Bytes(registryData); actual != binding.RegistrySHA256 {
		return fmt.Errorf(
			"wave registry hash mismatch: binding=%s frozen_base=%s",
			binding.RegistrySHA256,
			actual,
		)
	}
	var registry waveRegistry
	if err := json.Unmarshal(registryData, &registry); err != nil {
		return fmt.Errorf("decode wave registry: %w", err)
	}
	record, err := validateActiveWaveRegistry(registry, binding.WaveID)
	if err != nil {
		return err
	}

	expectedPlanPath, err := registryPlanPath(record.Document)
	if err != nil {
		return err
	}
	if binding.PlanPath != expectedPlanPath {
		return fmt.Errorf(
			"wave plan path mismatch: binding=%q registry=%q",
			binding.PlanPath,
			expectedPlanPath,
		)
	}
	planData, err := gitBlobAtBase(
		ctx,
		task.RepoPath,
		baseCommit,
		expectedPlanPath,
	)
	if err != nil {
		return fmt.Errorf("read wave plan at frozen base: %w", err)
	}
	if actual := sha256Bytes(planData); actual != binding.PlanSHA256 {
		return fmt.Errorf(
			"wave plan hash mismatch: binding=%s frozen_base=%s",
			binding.PlanSHA256,
			actual,
		)
	}
	frontMatter, err := decodeWavePlanFrontMatter(planData)
	if err != nil {
		return err
	}
	if err := validateWavePlan(frontMatter, binding, record); err != nil {
		return err
	}
	if err := validateWaveDependencies(registry, record.DependsOn); err != nil {
		return err
	}
	if err := validateTaskWaveScope(task, record, frontMatter); err != nil {
		return err
	}
	return nil
}

func validateWaveBindingShape(binding WaveBinding) error {
	if err := validateID(strings.TrimSpace(binding.WaveID)); err != nil {
		return fmt.Errorf("wave_id: %w", err)
	}
	if binding.PlanRevision <= 0 {
		return errors.New("wave plan_revision must be positive")
	}
	if err := validateID(strings.TrimSpace(binding.StepID)); err != nil {
		return fmt.Errorf("wave step_id: %w", err)
	}
	cleanPlanPath, err := cleanRepositoryPath(binding.PlanPath)
	if err != nil {
		return fmt.Errorf("wave plan_path: %w", err)
	}
	if cleanPlanPath != binding.PlanPath ||
		!strings.HasPrefix(cleanPlanPath, "docs/waves/") {
		return errors.New("wave plan_path must be a canonical path under docs/waves")
	}
	for label, digest := range map[string]string{
		"registry_sha256": binding.RegistrySHA256,
		"plan_sha256":     binding.PlanSHA256,
	} {
		if !validSHA256(digest) {
			return fmt.Errorf("wave %s must be a lowercase SHA-256 digest", label)
		}
	}
	return nil
}

func validateActiveWaveRegistry(
	registry waveRegistry,
	expectedWaveID string,
) (waveRegistryRecord, error) {
	if registry.SchemaVersion != 1 {
		return waveRegistryRecord{}, fmt.Errorf(
			"unsupported wave registry schema version %d",
			registry.SchemaVersion,
		)
	}
	records := make(map[string]waveRegistryRecord, len(registry.Waves))
	var active []waveRegistryRecord
	for _, record := range registry.Waves {
		if strings.TrimSpace(record.ID) == "" {
			return waveRegistryRecord{}, errors.New("wave registry contains an empty wave id")
		}
		if _, duplicate := records[record.ID]; duplicate {
			return waveRegistryRecord{}, fmt.Errorf(
				"wave registry contains duplicate wave %q",
				record.ID,
			)
		}
		records[record.ID] = record
		if record.Status == "active" {
			active = append(active, record)
		}
	}
	if len(active) != 1 {
		return waveRegistryRecord{}, fmt.Errorf(
			"wave registry requires exactly one active wave, got %d",
			len(active),
		)
	}
	if registry.ActiveWave != active[0].ID {
		return waveRegistryRecord{}, fmt.Errorf(
			"wave registry active_wave %q does not match active record %q",
			registry.ActiveWave,
			active[0].ID,
		)
	}
	if active[0].ID != expectedWaveID {
		return waveRegistryRecord{}, fmt.Errorf(
			"task wave %q is not the active wave %q",
			expectedWaveID,
			active[0].ID,
		)
	}
	if !active[0].ProductCodeChangesAllowed {
		return waveRegistryRecord{}, fmt.Errorf(
			"active wave %q does not allow product code changes",
			active[0].ID,
		)
	}
	return active[0], nil
}

func validateWavePlan(
	plan wavePlanFrontMatter,
	binding WaveBinding,
	record waveRegistryRecord,
) error {
	if plan.Schema != "goclaw.wave/v1" {
		return fmt.Errorf("unsupported wave plan schema %q", plan.Schema)
	}
	if plan.WaveID != binding.WaveID {
		return fmt.Errorf(
			"wave plan id %q does not match binding %q",
			plan.WaveID,
			binding.WaveID,
		)
	}
	if plan.Revision != binding.PlanRevision {
		return fmt.Errorf(
			"wave plan revision %d does not match binding %d",
			plan.Revision,
			binding.PlanRevision,
		)
	}
	if plan.PlanStatus != "approved" || len(plan.ApprovedBy) == 0 {
		return errors.New("wave plan is not approved")
	}
	if plan.WaveState != "active" {
		return fmt.Errorf("wave plan state %q is not active", plan.WaveState)
	}
	if !plan.ProductCodeChangesAllowed {
		return fmt.Errorf(
			"wave plan %q does not allow product code changes",
			plan.WaveID,
		)
	}
	if !slices.Contains(plan.Steps, binding.StepID) {
		return fmt.Errorf(
			"wave step %q is not declared by plan revision %d",
			binding.StepID,
			binding.PlanRevision,
		)
	}
	if !sameStringSet(plan.DependsOn, record.DependsOn) {
		return errors.New("wave plan dependencies do not match the registry")
	}
	return nil
}

func validateWaveDependencies(registry waveRegistry, dependencyIDs []string) error {
	statuses := make(map[string]string, len(registry.Waves))
	for _, record := range registry.Waves {
		statuses[record.ID] = record.Status
	}
	for _, dependencyID := range dependencyIDs {
		status, exists := statuses[dependencyID]
		if !exists {
			return fmt.Errorf("wave dependency %q is missing", dependencyID)
		}
		if status != "complete" {
			return fmt.Errorf(
				"wave dependency %q is not complete (status %q)",
				dependencyID,
				status,
			)
		}
	}
	return nil
}

func validateTaskWaveScope(
	task Task,
	record waveRegistryRecord,
	plan wavePlanFrontMatter,
) error {
	for _, candidate := range task.Scope.AllowedPaths {
		if err := validateScopeSubset(candidate, record.AllowedChangeScope); err != nil {
			return fmt.Errorf("task allowed path %q exceeds registry scope: %w", candidate, err)
		}
		if err := validateScopeSubset(candidate, plan.AllowedChangeScope); err != nil {
			return fmt.Errorf("task allowed path %q exceeds plan scope: %w", candidate, err)
		}
	}
	for _, milestone := range task.Plan.Milestones {
		for _, item := range milestone.WorkItems {
			for _, candidate := range item.ScopePaths {
				if err := validateScopeSubset(candidate, task.Scope.AllowedPaths); err != nil {
					return fmt.Errorf(
						"work item %q scope %q exceeds task scope: %w",
						item.ID,
						candidate,
						err,
					)
				}
			}
		}
	}
	return nil
}

func validateScopeSubset(candidate string, permitted []string) error {
	cleanCandidate, err := cleanRepositoryPattern(candidate)
	if err != nil {
		return err
	}
	for _, allowed := range permitted {
		cleanAllowed, cleanErr := cleanRepositoryPattern(allowed)
		if cleanErr != nil {
			return fmt.Errorf("invalid authority scope %q: %w", allowed, cleanErr)
		}
		if scopePatternWithin(cleanCandidate, cleanAllowed) {
			return nil
		}
	}
	return errors.New("no enclosing allowed scope")
}

func scopePatternWithin(candidate, allowed string) bool {
	if allowed == "**" || allowed == "*" || candidate == allowed {
		return true
	}
	candidatePrefix := repositoryPatternPrefix(candidate)
	if strings.HasSuffix(allowed, "/**") {
		root := strings.TrimSuffix(allowed, "/**")
		return candidatePrefix == root ||
			strings.HasPrefix(candidatePrefix, root+"/")
	}
	if !strings.ContainsAny(allowed, "*?[") {
		return candidatePrefix == allowed ||
			strings.HasPrefix(candidatePrefix, strings.TrimSuffix(allowed, "/")+"/")
	}
	if !strings.ContainsAny(candidate, "*?[") {
		return matchesAnyPath(candidate, []string{allowed})
	}
	// Proving one arbitrary glob is a subset of another requires a full glob
	// automaton. Reject uncertain glob-on-glob cases rather than widening the
	// frozen authority. Exact equality and enclosing /** were handled above.
	return false
}

func repositoryPatternPrefix(pattern string) string {
	index := strings.IndexAny(pattern, "*?[")
	if index >= 0 {
		pattern = pattern[:index]
	}
	return strings.TrimSuffix(pattern, "/")
}

func cleanRepositoryPattern(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("scope path is empty")
	}
	if strings.Contains(value, "\\") || strings.ContainsRune(value, '\x00') {
		return "", errors.New("scope path must use repository slash syntax")
	}
	prefix := repositoryPatternPrefix(value)
	if prefix == "" && value != "*" && value != "**" {
		return "", errors.New("scope pattern has no stable repository prefix")
	}
	if prefix != "" {
		clean, err := cleanRepositoryPath(prefix)
		if err != nil {
			return "", err
		}
		if clean != strings.TrimSuffix(prefix, "/") {
			return "", errors.New("scope path is not canonical")
		}
	}
	return value, nil
}

func cleanRepositoryPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsRune(value, '\x00') {
		return "", errors.New("repository path is empty or contains NUL")
	}
	if strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return "", errors.New("repository path must be relative and use slash separators")
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("repository path escapes the repository")
	}
	return clean, nil
}

func registryPlanPath(document string) (string, error) {
	document = strings.TrimSpace(document)
	clean, err := cleanRepositoryPath(document)
	if err != nil {
		return "", fmt.Errorf("wave registry document: %w", err)
	}
	if clean != document || strings.HasPrefix(clean, "docs/waves/") {
		return "", errors.New("wave registry document must be canonical and relative to docs/waves")
	}
	return "docs/waves/" + clean, nil
}

func decodeWavePlanFrontMatter(data []byte) (wavePlanFrontMatter, error) {
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return wavePlanFrontMatter{}, errors.New("wave plan is missing YAML front matter")
	}
	remainder := content[len("---\n"):]
	end := strings.Index(remainder, "\n---")
	if end < 0 {
		return wavePlanFrontMatter{}, errors.New("wave plan front matter is not terminated")
	}
	var result wavePlanFrontMatter
	if err := yaml.Unmarshal([]byte(remainder[:end]), &result); err != nil {
		return wavePlanFrontMatter{}, fmt.Errorf("decode wave plan front matter: %w", err)
	}
	return result, nil
}

func gitBlobAtBase(
	ctx context.Context,
	repoPath string,
	baseCommit string,
	repositoryPath string,
) ([]byte, error) {
	clean, err := cleanRepositoryPath(repositoryPath)
	if err != nil {
		return nil, err
	}
	if clean != repositoryPath {
		return nil, errors.New("repository blob path is not canonical")
	}
	result, err := runGit(
		ctx,
		repoPath,
		"show",
		strings.TrimSpace(baseCommit)+":"+repositoryPath,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(result.Stderr))
	}
	if len(result.Stdout) >= maxCapturedCommandBytes {
		return nil, fmt.Errorf(
			"repository blob %q exceeds the %d-byte integrity limit",
			repositoryPath,
			maxCapturedCommandBytes,
		)
	}
	return []byte(result.Stdout), nil
}

func validSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') &&
			(character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func sameStringSet(left, right []string) bool {
	left = uniqueStrings(left)
	right = uniqueStrings(right)
	return slices.Equal(left, right)
}
