package orchestratorlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/governance"
)

type Service struct {
	cfg        Config
	mu         sync.Mutex
	hand       Hand
	busy       map[string]bool
	governance governance.Config
}

func NewService(cfg Config) (*Service, error) {
	if strings.TrimSpace(cfg.Root) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		cfg.Root = filepath.Join(home, ".goclaw", "development")
	}
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, err
	}
	cfg.Root = root
	if cfg.WorktreeRoot == "" {
		cfg.WorktreeRoot = filepath.Join(cfg.Root, "worktrees")
	}
	worktreeRoot, err := filepath.Abs(cfg.WorktreeRoot)
	if err != nil {
		return nil, err
	}
	cfg.WorktreeRoot = worktreeRoot
	if cfg.CodexCommand == "" {
		cfg.CodexCommand = "codex"
	}
	if cfg.RunTimeoutSeconds <= 0 {
		cfg.RunTimeoutSeconds = 6 * 60 * 60
	}
	if cfg.VerifyTimeoutSeconds <= 0 {
		cfg.VerifyTimeoutSeconds = 30 * 60
	}
	verificationSandbox, err := validateVerificationSandbox(
		cfg.VerificationSandbox,
	)
	if err != nil {
		return nil, err
	}
	cfg.VerificationSandbox = verificationSandbox
	if len(cfg.VerificationSandbox) > 0 && cfg.UnsafeHostVerification {
		return nil, errors.New(
			"verification_sandbox and unsafe_host_verification are mutually exclusive",
		)
	}
	if cfg.MaxRepairAttempts <= 0 {
		cfg.MaxRepairAttempts = 2
	}
	if cfg.DefaultMaxChangedFiles <= 0 {
		cfg.DefaultMaxChangedFiles = 40
	}
	if cfg.DefaultMaxChangedLines <= 0 {
		cfg.DefaultMaxChangedLines = 2000
	}
	if len(cfg.DeniedPaths) == 0 {
		cfg.DeniedPaths = []string{
			".git/**",
			".codex/orchestrator/**",
			".env",
			".env.*",
			"**/.env",
			"**/.env.*",
			"*credential*",
			"*secret*",
			"**/*credential*",
			"**/*secret*",
		}
	}
	service := &Service{
		cfg:        cfg,
		busy:       make(map[string]bool),
		governance: governance.DefaultConfig(),
	}
	service.hand = NewCodexExecHand(cfg)
	if err := service.Ensure(); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *Service) SetHand(hand Hand) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hand = hand
}

func (s *Service) SetGovernancePolicy(policy governance.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.governance = policy
}

func (s *Service) GovernancePolicy() governance.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.governance
}

func (s *Service) Config() Config {
	cfg := s.cfg
	cfg.VerificationSandbox = append(
		[]string(nil),
		s.cfg.VerificationSandbox...,
	)
	return cfg
}

func validateVerificationSandbox(sandbox []string) ([]string, error) {
	result := append([]string(nil), sandbox...)
	for index := range result {
		result[index] = strings.TrimSpace(result[index])
		if result[index] == "" {
			return nil, errors.New(
				"verification_sandbox entries must not be empty",
			)
		}
	}
	if len(result) == 0 {
		return nil, nil
	}
	if !filepath.IsAbs(result[0]) {
		return nil, errors.New(
			"verification_sandbox executable must be an absolute path",
		)
	}
	result[0] = filepath.Clean(result[0])
	info, err := os.Lstat(result[0])
	if err != nil {
		return nil, fmt.Errorf("verification_sandbox executable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return nil, errors.New(
			"verification_sandbox executable must be a regular executable file",
		)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o022 != 0 {
		return nil, errors.New(
			"verification_sandbox executable must not be writable by group or others",
		)
	}
	return result, nil
}

func (s *Service) Ensure() error {
	for _, dir := range []string{s.tasksDir(), s.cfg.WorktreeRoot, s.locksDir()} {
		if err := ensureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CreateTask(request CreateRequest) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(request.Title) == "" {
		return Task{}, errors.New("title is required")
	}
	if strings.TrimSpace(request.Request.RawRequest) == "" {
		return Task{}, errors.New("request.raw_request is required")
	}
	if request.RepoPath == "" {
		request.RepoPath = s.cfg.RepoPath
	}
	if request.RepoPath == "" {
		return Task{}, errors.New("repo_path is required")
	}
	repoPath, err := filepath.Abs(request.RepoPath)
	if err != nil {
		return Task{}, err
	}
	if request.ID == "" {
		request.ID = "task-" + uuid.NewString()
	}
	if err := validateID(request.ID); err != nil {
		return Task{}, err
	}
	request.RepoPath = repoPath
	requestData, err := json.Marshal(request)
	if err != nil {
		return Task{}, err
	}
	requestSHA256 := sha256Bytes(requestData)
	if exists(s.taskDir(request.ID)) {
		existing, err := s.loadTaskUnlocked(request.ID)
		if err != nil {
			return Task{}, err
		}
		if existing.CreateRequestSHA256 == requestSHA256 {
			return existing, nil
		}
		return Task{}, fmt.Errorf(
			"task already exists with a different create request: %s",
			request.ID,
		)
	}
	now := time.Now().UTC()
	task := Task{
		SchemaVersion:       SchemaVersion,
		ID:                  request.ID,
		TeamID:              strings.TrimSpace(request.TeamID),
		ProjectID:           valueOr(request.ProjectID, "default"),
		RepositoryID:        strings.TrimSpace(request.RepositoryID),
		Module:              strings.TrimSpace(request.Module),
		AssigneeID:          strings.TrimSpace(request.AssigneeID),
		ParentTaskID:        strings.TrimSpace(request.ParentTaskID),
		IssueIDs:            uniqueStrings(request.IssueIDs),
		SpecRefs:            uniqueStrings(request.SpecRefs),
		DocumentRefs:        uniqueStrings(request.DocumentRefs),
		PolicyBundleHash:    strings.TrimSpace(request.PolicyBundleHash),
		PolicyInstructions:  uniqueStrings(request.PolicyInstructions),
		CorrelationID:       valueOr(strings.TrimSpace(request.CorrelationID), request.ID),
		CreateRequestSHA256: requestSHA256,
		Wave:                cloneWaveBinding(request.Wave),
		Title:               request.Title,
		Status:              TaskReviewPending,
		Request:             request.Request,
		Goal:                request.Goal,
		Plan:                request.Plan,
		EvidencePlan:        request.EvidencePlan,
		Scope:               request.Scope,
		Risk:                request.Risk,
		Cost:                request.Cost,
		DoneGate:            request.DoneGate,
		Reviews:             make(map[ReviewKind]ReviewRecord),
		Compile: CompileRecord{
			Revision: 1,
			BaseRef:  valueOr(request.BaseRef, "HEAD"),
		},
		RepoPath:    repoPath,
		CreatedBy:   valueOr(request.CreatedBy, "human"),
		RequestedBy: strings.TrimSpace(request.RequestedBy),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.applyTaskDefaults(&task)
	for _, kind := range RequiredReviewKinds {
		task.Reviews[kind] = ReviewRecord{Kind: kind, Decision: ReviewPending}
	}
	if err := s.appendEventUnlocked(task, "task.created", valueOr(request.CreatedBy, "human"), map[string]any{
		"compile_revision": task.Compile.Revision,
		"requested_by":     task.RequestedBy,
		"wave":             task.Wave,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Service) ReviewTask(id string, kind ReviewKind, decision ReviewDecision, reviewer, comment string) (Task, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return Task{}, errors.New("authenticated governance requires ReviewTaskWithReview")
	}
	return s.ReviewTaskWithReview(id, kind, decision, governance.Review{
		ReviewerID:    reviewer,
		Rationale:     comment,
		Role:          governanceRoleForReview(kind),
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) ReviewTaskWithReview(
	id string,
	kind ReviewKind,
	decision ReviewDecision,
	review governance.Review,
) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, err := s.loadTaskUnlocked(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status != TaskReviewPending && task.Status != TaskReadyToFreeze && task.Status != TaskBlocked {
		return Task{}, fmt.Errorf("task %s cannot be reviewed in status %s", id, task.Status)
	}
	if !containsReviewKind(RequiredReviewKinds, kind) {
		return Task{}, fmt.Errorf("unsupported review kind %q", kind)
	}
	if decision != ReviewApproved && decision != ReviewRejected {
		return Task{}, errors.New("decision must be approved or rejected")
	}
	requiredRole := governanceRoleForReview(kind)
	if err := governance.ValidateRole(review, requiredRole); err != nil {
		return Task{}, err
	}
	if err := governance.ValidateDecision(s.governance, review, string(decision), task.CreatedBy); err != nil {
		return Task{}, err
	}
	record := governance.ToDecision(review, string(decision))
	prior := task.Reviews[kind]
	if prior.Governance != nil &&
		prior.Decision == decision &&
		governance.SameActor(
			prior.Governance.ReviewerID,
			record.ReviewerID,
		) &&
		prior.Governance.Rationale == record.Rationale &&
		prior.Governance.Counterargument == record.Counterargument &&
		strings.Join(prior.Governance.EvidenceRefs, "\x00") ==
			strings.Join(record.EvidenceRefs, "\x00") {
		return task, nil
	}
	if s.governance.Enabled && s.governance.MaxTaskReviewKindsPerReviewer > 0 {
		kinds := 0
		for priorKind, prior := range task.Reviews {
			if priorKind != kind &&
				prior.Decision == ReviewApproved &&
				governance.SameActor(prior.Reviewer, review.ReviewerID) {
				kinds++
			}
		}
		if decision == ReviewApproved && kinds >= s.governance.MaxTaskReviewKindsPerReviewer {
			return Task{}, fmt.Errorf(
				"reviewer %q already holds the maximum %d task review kinds",
				review.ReviewerID,
				s.governance.MaxTaskReviewKindsPerReviewer,
			)
		}
	}
	now := record.CreatedAt
	task.Reviews[kind] = ReviewRecord{
		Kind:       kind,
		Decision:   decision,
		Reviewer:   record.ReviewerID,
		Comment:    record.Rationale,
		Governance: &record,
		DecidedAt:  &now,
	}
	task.Status = reviewStatus(task.Reviews)
	if task.Status == TaskReadyToFreeze && s.governance.Enabled {
		if distinctTaskReviewers(task.Reviews) < maximum(1, s.governance.MinDistinctTaskReviewers) {
			task.Status = TaskReviewPending
		}
	}
	task.UpdatedAt = now
	if err := s.appendEventUnlocked(task, "review.decided", record.ReviewerID, map[string]any{
		"kind": kind, "decision": decision, "governance": record,
		"distinct_reviewers": distinctTaskReviewers(task.Reviews),
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func governanceRoleForReview(kind ReviewKind) string {
	switch kind {
	case ReviewScenario:
		return governance.RoleScenarioReview
	case ReviewCapacity:
		return governance.RoleCapacityReview
	case ReviewRisk:
		return governance.RoleRiskReview
	case ReviewCost:
		return governance.RoleCostReview
	default:
		return ""
	}
}

func distinctTaskReviewers(reviews map[ReviewKind]ReviewRecord) int {
	seen := make(map[string]struct{})
	for _, record := range reviews {
		if record.Decision != ReviewApproved || strings.TrimSpace(record.Reviewer) == "" {
			continue
		}
		seen[strings.ToLower(strings.TrimSpace(record.Reviewer))] = struct{}{}
	}
	return len(seen)
}

func maximum(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func (s *Service) FreezeTask(ctx context.Context, id, actor string) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, err := s.loadTaskUnlocked(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status == TaskFrozen {
		return task, nil
	}
	if task.Status != TaskReadyToFreeze {
		return Task{}, fmt.Errorf("task %s requires all four approved reviews before freeze (status %s)", id, task.Status)
	}
	if err := validateFrozenTask(task); err != nil {
		return Task{}, err
	}
	if !s.cfg.AllowDirtyRepo {
		status, err := runGit(ctx, task.RepoPath, "status", "--porcelain")
		if err != nil {
			return Task{}, fmt.Errorf("inspect repository: %w", err)
		}
		if strings.TrimSpace(status.Stdout) != "" {
			return Task{}, errors.New("repository has uncommitted changes; commit/stash them or enable development.allow_dirty_repo")
		}
	}
	base, err := runGit(ctx, task.RepoPath, "rev-parse", "--verify", task.Compile.BaseRef+"^{commit}")
	if err != nil {
		return Task{}, fmt.Errorf("resolve base ref %s: %w", task.Compile.BaseRef, err)
	}
	task.Compile.BaseCommit = strings.TrimSpace(base.Stdout)
	if task.Wave != nil {
		if err := validateWaveBindingAtBase(ctx, task, task.Compile.BaseCommit); err != nil {
			return Task{}, fmt.Errorf("validate wave binding: %w", err)
		}
	}
	task.Compile.FrozenAt = time.Now().UTC()
	task.Compile.FrozenBy = valueOr(actor, "human")
	task.Status = TaskFrozen
	task.UpdatedAt = task.Compile.FrozenAt
	hashInput := task
	hashInput.Compile.ExecutionBundleHash = ""
	hashInput.UpdatedAt = time.Time{}
	data, err := json.Marshal(hashInput)
	if err != nil {
		return Task{}, err
	}
	task.Compile.ExecutionBundleHash = sha256Bytes(data)
	if err := s.appendEventUnlocked(task, "task.frozen", task.Compile.FrozenBy, map[string]any{
		"base_commit":           task.Compile.BaseCommit,
		"execution_bundle_hash": task.Compile.ExecutionBundleHash,
		"wave":                  task.Wave,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Service) ReviseTask(id, actor, reason string, replacement Task) (Task, error) {
	release, err := s.acquireRun(id, false)
	if err != nil {
		return Task{}, err
	}
	defer release()

	s.mu.Lock()
	defer s.mu.Unlock()
	task, err := s.loadTaskUnlocked(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status == TaskRunning || task.Status == TaskChecking {
		return Task{}, errors.New("cannot revise a running task")
	}
	if task.Status == TaskDone || task.Status == TaskCancelled {
		return Task{}, fmt.Errorf("cannot revise task in status %s", task.Status)
	}
	if strings.TrimSpace(reason) == "" {
		return Task{}, errors.New("change intent reason is required")
	}
	immutableID := task.ID
	createdAt := task.CreatedAt
	if replacement.Title != "" {
		task.Title = replacement.Title
	}
	if replacement.TeamID != "" {
		task.TeamID = replacement.TeamID
	}
	if replacement.ProjectID != "" {
		task.ProjectID = replacement.ProjectID
	}
	if replacement.RepositoryID != "" {
		task.RepositoryID = replacement.RepositoryID
	}
	if replacement.Module != "" {
		task.Module = replacement.Module
	}
	if replacement.AssigneeID != "" {
		task.AssigneeID = replacement.AssigneeID
	}
	if replacement.ParentTaskID != "" {
		task.ParentTaskID = replacement.ParentTaskID
	}
	if replacement.CorrelationID != "" {
		task.CorrelationID = replacement.CorrelationID
	}
	if replacement.PolicyBundleHash != "" {
		task.PolicyBundleHash = replacement.PolicyBundleHash
	}
	if len(replacement.PolicyInstructions) > 0 {
		task.PolicyInstructions = uniqueStrings(replacement.PolicyInstructions)
	}
	if len(replacement.IssueIDs) > 0 {
		task.IssueIDs = uniqueStrings(replacement.IssueIDs)
	}
	if len(replacement.SpecRefs) > 0 {
		task.SpecRefs = uniqueStrings(replacement.SpecRefs)
	}
	if len(replacement.DocumentRefs) > 0 {
		task.DocumentRefs = uniqueStrings(replacement.DocumentRefs)
	}
	if replacement.Request.RawRequest != "" {
		task.Request = replacement.Request
	}
	if replacement.Goal.Objective != "" {
		task.Goal = replacement.Goal
	}
	if len(replacement.Plan.Milestones) > 0 {
		task.Plan = replacement.Plan
	}
	if len(replacement.EvidencePlan.Commands) > 0 {
		task.EvidencePlan = replacement.EvidencePlan
	}
	if replacement.Scope.MaxChangedFiles > 0 {
		task.Scope = replacement.Scope
	}
	if replacement.Risk.Level != "" {
		task.Risk = replacement.Risk
	}
	if replacement.Cost.MaxRepairAttempts > 0 {
		task.Cost = replacement.Cost
	}
	task.ID = immutableID
	task.CreatedAt = createdAt
	task.Compile.Revision++
	task.Compile.BaseCommit = ""
	task.Compile.ExecutionBundleHash = ""
	task.Compile.FrozenAt = time.Time{}
	task.Compile.FrozenBy = ""
	task.WorktreePath = ""
	task.Branch = ""
	task.CurrentRunID = ""
	task.CodexThreadID = ""
	task.LastGate = nil
	task.LastEvidence = ""
	task.RepairCount = 0
	task.Status = TaskReviewPending
	task.UpdatedAt = time.Now().UTC()
	task.Reviews = make(map[ReviewKind]ReviewRecord)
	for _, kind := range RequiredReviewKinds {
		task.Reviews[kind] = ReviewRecord{Kind: kind, Decision: ReviewPending}
	}
	s.applyTaskDefaults(&task)
	if err := s.appendEventUnlocked(task, "task.revised", valueOr(actor, "human"), map[string]any{
		"reason": reason, "compile_revision": task.Compile.Revision,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Service) GetTask(id string) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadTaskUnlocked(id)
}

func (s *Service) ListTasks(projectID string) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.tasksDir())
	if err != nil {
		return nil, err
	}
	var tasks []Task
	for _, entry := range entries {
		if !entry.IsDir() || validateID(entry.Name()) != nil {
			continue
		}
		task, err := s.loadTaskUnlocked(entry.Name())
		if err != nil {
			continue
		}
		if projectID == "" || task.ProjectID == projectID {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})
	return tasks, nil
}

func (s *Service) ListEvents(id string) ([]SessionEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadEventsUnlocked(id)
}

func (s *Service) applyTaskDefaults(task *Task) {
	if task.Goal.Objective == "" {
		task.Goal.Objective = task.Request.RawRequest
	}
	if task.Plan.Summary == "" {
		task.Plan.Summary = "Implement the frozen request and produce verifiable evidence."
	}
	if len(task.Plan.Milestones) == 0 {
		task.Plan.Milestones = []Milestone{{
			ID:    "implementation",
			Title: "Implementation",
			WorkItems: []WorkItem{{
				ID:           "implementation",
				Title:        task.Title,
				Instructions: task.Request.RawRequest,
			}},
		}}
	}
	if task.Scope.MaxChangedFiles <= 0 {
		task.Scope.MaxChangedFiles = s.cfg.DefaultMaxChangedFiles
	}
	if task.Scope.MaxChangedLines <= 0 {
		task.Scope.MaxChangedLines = s.cfg.DefaultMaxChangedLines
	}
	task.Scope.DeniedPaths = uniqueStrings(append(task.Scope.DeniedPaths, s.cfg.DeniedPaths...))
	if task.Cost.MaxRepairAttempts <= 0 {
		task.Cost.MaxRepairAttempts = s.cfg.MaxRepairAttempts
	}
	if task.Risk.Level == "" {
		task.Risk.Level = "medium"
	}
	if task.Risk.Rollback == "" {
		task.Risk.Rollback = "Discard the isolated worktree or reset the task branch to the frozen base commit."
	}
	if len(task.EvidencePlan.Commands) == 0 {
		for _, milestone := range task.Plan.Milestones {
			for _, workItem := range milestone.WorkItems {
				task.EvidencePlan.Commands = append(task.EvidencePlan.Commands, workItem.VerificationCommands...)
			}
		}
	}
	if len(task.EvidencePlan.Required) == 0 {
		task.EvidencePlan.Required = []string{
			"repository_before",
			"repository_after",
			"diff",
			"verification_results",
			"policy_result",
			"donegate_result",
		}
	}
	if !task.DoneGate.RequireAllVerifications && !task.DoneGate.RequirePolicyPass &&
		!task.DoneGate.RequireChangedFiles && !task.DoneGate.RequireIndependentReview {
		task.DoneGate.RequireChangedFiles = true
		task.DoneGate.RequireAllVerifications = true
		task.DoneGate.RequirePolicyPass = true
		task.DoneGate.RequireIndependentReview = s.cfg.IndependentReview
		task.DoneGate.RequireHumanAcceptance = s.cfg.RequireHumanFinalApproval
	}
	capability := CapabilityManifest{
		Executor: "codex-exec",
		Model:    s.cfg.CodexModel,
		Tools:    []string{"edit", "shell", "test"},
		Sandbox:  "workspace-write",
	}
	for milestoneIndex := range task.Plan.Milestones {
		for workItemIndex := range task.Plan.Milestones[milestoneIndex].WorkItems {
			item := &task.Plan.Milestones[milestoneIndex].WorkItems[workItemIndex]
			if item.ID == "" {
				item.ID = fmt.Sprintf("work-%d-%d", milestoneIndex+1, workItemIndex+1)
			}
			if item.Status == "" {
				item.Status = WorkItemPending
			}
			if item.AssigneeID == "" {
				item.AssigneeID = task.AssigneeID
			}
			item.IssueIDs = uniqueStrings(append(item.IssueIDs, task.IssueIDs...))
			item.DocumentRefs = uniqueStrings(item.DocumentRefs)
			item.ScopePaths = uniqueStrings(item.ScopePaths)
			if item.CapabilityManifest.Executor == "" {
				item.CapabilityManifest = capability
			}
		}
	}
}

func validateFrozenTask(task Task) error {
	if task.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported task schema version %d", task.SchemaVersion)
	}
	if strings.TrimSpace(task.ProjectID) == "" {
		return errors.New("project_id is required")
	}
	if strings.TrimSpace(task.AssigneeID) != "" && strings.TrimSpace(task.RepositoryID) == "" {
		return errors.New("repository_id is required for assigned team tasks")
	}
	if task.DoneGate.RequirePolicyBundle && strings.TrimSpace(task.PolicyBundleHash) == "" {
		return errors.New("policy_bundle_hash is required by the DoneGate")
	}
	if task.DoneGate.RequirePolicyBundle && len(task.PolicyInstructions) == 0 {
		return errors.New("policy_instructions are required by the DoneGate")
	}
	if task.DoneGate.RequireDocumentEvidence && len(task.DocumentRefs) == 0 {
		return errors.New("document_refs are required by the DoneGate")
	}
	if task.Goal.Objective == "" {
		return errors.New("goal objective is required")
	}
	if len(task.Plan.Milestones) == 0 {
		return errors.New("at least one milestone is required")
	}
	workItemIDs := make(map[string]struct{})
	for _, milestone := range task.Plan.Milestones {
		for _, item := range milestone.WorkItems {
			if strings.TrimSpace(item.ID) == "" {
				return errors.New("work item id is required")
			}
			if _, exists := workItemIDs[item.ID]; exists {
				return fmt.Errorf("duplicate work item id %q", item.ID)
			}
			workItemIDs[item.ID] = struct{}{}
		}
	}
	for _, milestone := range task.Plan.Milestones {
		for _, item := range milestone.WorkItems {
			for _, dependency := range item.DependsOn {
				if _, exists := workItemIDs[dependency]; !exists {
					return fmt.Errorf("work item %q depends on unknown work item %q", item.ID, dependency)
				}
			}
		}
	}
	if len(task.EvidencePlan.Commands) == 0 {
		return errors.New("at least one deterministic verification command is required")
	}
	for _, command := range task.EvidencePlan.Commands {
		if len(command.Argv) == 0 || strings.TrimSpace(command.Argv[0]) == "" {
			return errors.New("verification commands must use non-empty argv arrays")
		}
	}
	for _, kind := range RequiredReviewKinds {
		if task.Reviews[kind].Decision != ReviewApproved {
			return fmt.Errorf("%s review is not approved", kind)
		}
	}
	return nil
}

func (s *Service) appendEventUnlocked(task Task, eventType, actor string, data any) error {
	events, err := s.loadEventsOptionalUnlocked(task.ID)
	if err != nil {
		return err
	}
	var previousHash string
	var sequence int64 = 1
	if len(events) > 0 {
		previousHash = events[len(events)-1].Hash
		sequence = events[len(events)-1].Sequence + 1
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	event := SessionEvent{
		SchemaVersion: SchemaVersion,
		ID:            "event-" + uuid.NewString(),
		TaskID:        task.ID,
		Sequence:      sequence,
		Type:          eventType,
		Actor:         valueOr(actor, "system"),
		CreatedAt:     time.Now().UTC(),
		PreviousHash:  previousHash,
		Data:          raw,
		Snapshot:      task,
	}
	hashable := event
	hashable.Hash = ""
	encoded, err := json.Marshal(hashable)
	if err != nil {
		return err
	}
	event.Hash = sha256Bytes(encoded)
	if err := appendJSONLine(s.eventsPath(task.ID), event); err != nil {
		return err
	}
	return writeJSONAtomic(s.projectionPath(task.ID), task, 0o600)
}

func (s *Service) loadTaskUnlocked(id string) (Task, error) {
	events, err := s.loadEventsUnlocked(id)
	if err != nil {
		return Task{}, err
	}
	if len(events) == 0 {
		return Task{}, os.ErrNotExist
	}
	task := events[len(events)-1].Snapshot
	if task.ID != id {
		return Task{}, fmt.Errorf("event snapshot task id %q does not match %q", task.ID, id)
	}
	return task, nil
}

func (s *Service) loadEventsOptionalUnlocked(id string) ([]SessionEvent, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	events, err := readEventLines(s.eventsPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := validateEventChain(events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *Service) loadEventsUnlocked(id string) ([]SessionEvent, error) {
	events, err := s.loadEventsOptionalUnlocked(id)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, os.ErrNotExist
	}
	return events, nil
}

func validateEventChain(events []SessionEvent) error {
	var previousHash string
	for index, event := range events {
		if event.Sequence != int64(index+1) {
			return fmt.Errorf("event sequence mismatch at %d", index+1)
		}
		if event.PreviousHash != previousHash {
			return fmt.Errorf("event hash chain mismatch at sequence %d", event.Sequence)
		}
		hashable := event
		hashable.Hash = ""
		encoded, err := json.Marshal(hashable)
		if err != nil {
			return err
		}
		if sha256Bytes(encoded) != event.Hash {
			return fmt.Errorf("event content hash mismatch at sequence %d", event.Sequence)
		}
		previousHash = event.Hash
	}
	return nil
}

func (s *Service) taskDir(id string) string {
	path, err := safeJoin(s.tasksDir(), id)
	if err != nil {
		return filepath.Join(s.tasksDir(), "__invalid__")
	}
	return path
}

func (s *Service) eventsPath(id string) string {
	return filepath.Join(s.taskDir(id), "events.jsonl")
}

func (s *Service) projectionPath(id string) string {
	return filepath.Join(s.taskDir(id), "task.json")
}

func (s *Service) tasksDir() string { return filepath.Join(s.cfg.Root, "tasks") }
func (s *Service) locksDir() string { return filepath.Join(s.cfg.Root, "locks") }

func reviewStatus(reviews map[ReviewKind]ReviewRecord) TaskStatus {
	allApproved := true
	for _, kind := range RequiredReviewKinds {
		switch reviews[kind].Decision {
		case ReviewRejected:
			return TaskBlocked
		case ReviewApproved:
		default:
			allApproved = false
		}
	}
	if allApproved {
		return TaskReadyToFreeze
	}
	return TaskReviewPending
}

func containsReviewKind(values []ReviewKind, expected ReviewKind) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.ToSlash(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func cloneWaveBinding(binding *WaveBinding) *WaveBinding {
	if binding == nil {
		return nil
	}
	copy := *binding
	return &copy
}
