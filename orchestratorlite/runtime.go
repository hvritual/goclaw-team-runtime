package orchestratorlite

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/governance"
)

func (s *Service) RunTask(ctx context.Context, id, actor string) (Task, error) {
	return s.executeTask(ctx, id, actor, false, false, false)
}

func (s *Service) RepairTask(ctx context.Context, id, actor string) (Task, error) {
	return s.executeTask(ctx, id, actor, true, false, false)
}

func (s *Service) ResumeTask(ctx context.Context, id, actor string, force bool) (Task, error) {
	return s.executeTask(ctx, id, actor, true, true, force)
}

func (s *Service) executeTask(ctx context.Context, id, actor string, repair, resume, force bool) (Task, error) {
	release, err := s.acquireRun(id, force)
	if err != nil {
		return Task{}, err
	}
	defer release()

	task, err := s.GetTask(id)
	if err != nil {
		return Task{}, err
	}
	if err := validateExecutionStatus(task, repair, resume); err != nil {
		return Task{}, err
	}
	if repair && task.RepairCount >= task.Cost.MaxRepairAttempts {
		return Task{}, fmt.Errorf("repair attempts %d reached approved limit %d", task.RepairCount, task.Cost.MaxRepairAttempts)
	}
	if task.Cost.MaxInputTokens > 0 && task.CumulativeUsage.InputTokens >= task.Cost.MaxInputTokens {
		return Task{}, fmt.Errorf("cumulative input-token budget %d is exhausted", task.Cost.MaxInputTokens)
	}
	if task.Cost.MaxOutputTokens > 0 && task.CumulativeUsage.OutputTokens >= task.Cost.MaxOutputTokens {
		return Task{}, fmt.Errorf("cumulative output-token budget %d is exhausted", task.Cost.MaxOutputTokens)
	}

	task, before, err := s.ensureWorktree(ctx, task)
	if err != nil {
		_ = s.recordFailure(id, actor, "worktree", err)
		return Task{}, err
	}
	runID := "run-" + uuid.NewString()
	if resume && task.CurrentRunID != "" {
		runID = task.CurrentRunID + "-resume-" + uuid.NewString()
	}
	task.CurrentRunID = runID
	task.Status = TaskRunning
	setAllWorkItemsStatus(&task, WorkItemRunning)
	if repair {
		task.RepairCount++
	}
	task.UpdatedAt = time.Now().UTC()
	if err := s.recordTaskEvent(task, "run.started", actor, map[string]any{
		"run_id":   runID,
		"repair":   repair,
		"resume":   resume,
		"worktree": task.WorktreePath,
	}); err != nil {
		return Task{}, err
	}

	runDir := s.runDir(task.ID, runID)
	if err := ensureDir(runDir); err != nil {
		_ = s.recordFailure(id, actor, "evidence_directory", err)
		return Task{}, err
	}
	if err := writeJSONAtomic(filepath.Join(runDir, "task-snapshot.json"), task, 0o600); err != nil {
		return Task{}, err
	}
	if err := writeJSONAtomic(filepath.Join(runDir, "repository-before.json"), before, 0o600); err != nil {
		return Task{}, err
	}

	prompt, err := s.buildExecutionPrompt(task, repair)
	if err != nil {
		return Task{}, err
	}
	resumeThreadID := ""
	if resume {
		resumeThreadID = valueOr(task.CodexThreadID, s.recoverThreadID(task))
	}
	handResult, handErr := s.hand.Execute(ctx, HandRequest{
		Task:           task,
		RunID:          runID,
		Prompt:         prompt,
		ResumeThreadID: resumeThreadID,
		EventsPath:     filepath.Join(runDir, "codex-events.jsonl"),
	})
	task.RunCount++
	addCodexUsage(&task.CumulativeUsage, handResult.Usage)
	task.CodexThreadID = handResult.ThreadID
	_ = writeJSONAtomic(filepath.Join(runDir, "codex-result.json"), handResult, 0o600)
	_ = writeBytesAtomic(filepath.Join(runDir, "codex-final.md"), []byte(handResult.FinalText), 0o600)
	if handErr != nil {
		_ = s.recordRunFailure(task, actor, "codex_hand", handErr, runDir)
		failed, _ := s.GetTask(id)
		return failed, handErr
	}

	task.Status = TaskChecking
	setAllWorkItemsStatus(&task, WorkItemVerifying)
	task.UpdatedAt = time.Now().UTC()
	if err := s.recordTaskEvent(task, "run.codex_completed", "codex-hand", map[string]any{
		"run_id":    runID,
		"thread_id": handResult.ThreadID,
		"usage":     handResult.Usage,
	}); err != nil {
		return Task{}, err
	}

	changedFiles, err := collectChangedFiles(ctx, task)
	if err != nil {
		_ = s.recordRunFailure(task, actor, "collect_changed_files", err, runDir)
		failed, _ := s.GetTask(id)
		return failed, err
	}
	diff, err := collectDiff(ctx, task, changedFiles)
	if err != nil {
		_ = s.recordRunFailure(task, actor, "collect_diff", err, runDir)
		failed, _ := s.GetTask(id)
		return failed, err
	}
	diffPath := filepath.Join(runDir, "diff.patch")
	if err := writeBytesAtomic(diffPath, []byte(diff), 0o600); err != nil {
		return Task{}, err
	}
	policy, err := evaluatePolicy(ctx, task, changedFiles)
	if err != nil {
		_ = s.recordRunFailure(task, actor, "policy", err, runDir)
		failed, _ := s.GetTask(id)
		return failed, err
	}
	verification := s.runVerifications(ctx, task, runDir)
	after, err := captureRepositorySnapshot(ctx, task)
	if err != nil {
		_ = s.recordRunFailure(task, actor, "repository_after", err, runDir)
		failed, _ := s.GetTask(id)
		return failed, err
	}
	_ = writeJSONAtomic(filepath.Join(runDir, "repository-after.json"), after, 0o600)
	_ = writeJSONAtomic(filepath.Join(runDir, "policy.json"), policy, 0o600)
	_ = writeJSONAtomic(filepath.Join(runDir, "verification.json"), verification, 0o600)

	review := IndependentReview{Passed: true, Summary: "Independent model review is disabled by the frozen DoneGate."}
	if task.DoneGate.RequireIndependentReview {
		if policy.Passed && verificationsPassed(verification) {
			modelReview, reviewHand, reviewErr := s.hand.Review(ctx, ReviewRequest{
				Task:       task,
				RunID:      runID,
				Diff:       diff,
				Evidence:   verification,
				EventsPath: filepath.Join(runDir, "codex-review-events.jsonl"),
			})
			addCodexUsage(&task.CumulativeUsage, reviewHand.Usage)
			_ = writeJSONAtomic(filepath.Join(runDir, "codex-review-result.json"), reviewHand, 0o600)
			if reviewErr != nil {
				review = IndependentReview{
					Passed:   false,
					Summary:  "Independent review could not complete.",
					Findings: []string{reviewErr.Error()},
				}
			} else {
				review = modelReview
			}
		} else {
			review = IndependentReview{
				Passed:   false,
				Summary:  "Independent review skipped because deterministic checks failed.",
				Findings: []string{"Resolve policy and verification failures before model review."},
			}
		}
	}
	_ = writeJSONAtomic(filepath.Join(runDir, "independent-review.json"), review, 0o600)
	falsifiers := buildFalsifierResults(task, verification, runDir)
	predictions := buildPredictionChecks(task, policy, verification, review, runDir)
	killChecks := buildKillConditionChecks(task, policy)
	attribution, unattributed := attributeChangedFiles(task, changedFiles)
	_ = writeJSONAtomic(filepath.Join(runDir, "falsifier-results.json"), falsifiers, 0o600)
	_ = writeJSONAtomic(filepath.Join(runDir, "prediction-checks.json"), predictions, 0o600)
	_ = writeJSONAtomic(filepath.Join(runDir, "kill-condition-checks.json"), killChecks, 0o600)
	_ = writeJSONAtomic(filepath.Join(runDir, "change-attribution.json"), map[string]any{
		"attribution":  attribution,
		"unattributed": unattributed,
	}, 0o600)

	pack := EvidencePackage{
		SchemaVersion:    SchemaVersion,
		ProjectID:        task.ProjectID,
		RepositoryID:     task.RepositoryID,
		TaskID:           task.ID,
		WorkItemIDs:      allWorkItemIDs(task),
		IssueIDs:         append([]string(nil), task.IssueIDs...),
		RunID:            runID,
		CorrelationID:    task.CorrelationID,
		PolicyBundleHash: task.PolicyBundleHash,
		DocumentRefs:     append([]string(nil), task.DocumentRefs...),
		TaskRevision:     task.Compile.Revision,
		Before:           before,
		After:            after,
		Hand:             handResult,
		Policy:           policy,
		Verification:     verification,
		Review:           review,
		Attribution:      attribution,
		Unattributed:     unattributed,
		Falsifiers:       falsifiers,
		Predictions:      predictions,
		KillChecks:       killChecks,
		DiffPath:         diffPath,
		CreatedAt:        time.Now().UTC(),
	}
	evidencePath := filepath.Join(runDir, "evidence.json")
	if err := writeJSONAtomic(evidencePath, pack, 0o600); err != nil {
		return Task{}, err
	}
	evidenceHash, err := sha256File(evidencePath)
	if err != nil {
		return Task{}, err
	}
	gate := evaluateDoneGate(task, pack, runDir, evidenceHash, worktreeTreeHash(diff, changedFiles))
	if err := writeJSONAtomic(filepath.Join(runDir, "donegate.json"), gate, 0o600); err != nil {
		return Task{}, err
	}
	task.LastGate = &gate
	task.LastEvidence = runDir
	task.UpdatedAt = time.Now().UTC()
	if gate.Passed {
		if task.DoneGate.RequireHumanAcceptance {
			task.Status = TaskAwaitingAcceptance
		} else {
			task.Status = TaskDone
			setAllWorkItemsStatus(&task, WorkItemDone)
		}
	} else {
		if hasTriggeredKill(pack.KillChecks) {
			task.Status = TaskBlocked
		} else {
			task.Status = TaskRepairPending
		}
		if task.Status != TaskBlocked && task.RepairCount >= task.Cost.MaxRepairAttempts {
			task.Status = TaskFailed
		}
		if task.Status == TaskBlocked || task.Status == TaskFailed {
			setAllWorkItemsStatus(&task, WorkItemBlocked)
		} else {
			setAllWorkItemsStatus(&task, WorkItemPending)
		}
	}
	eventType := "donegate.failed"
	if gate.Passed {
		eventType = "donegate.passed"
	}
	if err := s.recordTaskEvent(task, eventType, "go-donegate", map[string]any{
		"run_id":          runID,
		"verdict":         gate.Verdict,
		"reasons":         gate.Reasons,
		"evidence_path":   runDir,
		"evidence_sha256": evidenceHash,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Service) AcceptTask(ctx context.Context, id, reviewer, comment string) (Task, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return Task{}, errors.New("authenticated governance requires AcceptTaskWithReview")
	}
	return s.AcceptTaskWithReview(ctx, id, governance.Review{
		ReviewerID:    reviewer,
		Rationale:     comment,
		Role:          governance.RoleTaskAccept,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) AcceptTaskWithReview(
	ctx context.Context,
	id string,
	review governance.Review,
) (Task, error) {
	release, err := s.acquireRun(id, false)
	if err != nil {
		return Task{}, err
	}
	defer release()

	task, err := s.GetTask(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status != TaskAwaitingAcceptance {
		return Task{}, fmt.Errorf("task %s is not awaiting human acceptance", id)
	}
	if err := governance.ValidateRole(review, governance.RoleTaskAccept); err != nil {
		return Task{}, err
	}
	if err := governance.ValidateApproval(s.governance, review, task.CreatedBy); err != nil {
		return Task{}, err
	}
	if s.governance.Enabled && s.governance.ForbidFinalApproverFromTaskReview {
		for _, prior := range task.Reviews {
			if governance.SameActor(prior.Reviewer, review.ReviewerID) {
				return Task{}, fmt.Errorf(
					"final approver %q participated in task review %q",
					review.ReviewerID,
					prior.Kind,
				)
			}
		}
	}
	if s.governance.Enabled &&
		distinctTaskReviewers(task.Reviews) < maximum(1, s.governance.MinDistinctTaskReviewers) {
		return Task{}, errors.New("task review separation-of-duties quorum is not satisfied")
	}
	if task.LastGate == nil || !task.LastGate.Passed {
		return Task{}, errors.New("task has no passing DoneGate")
	}
	if err := requireRegularFile(filepath.Join(task.LastEvidence, "evidence.json")); err != nil {
		return Task{}, err
	}
	currentEvidenceHash, err := sha256File(filepath.Join(task.LastEvidence, "evidence.json"))
	if err != nil {
		return Task{}, err
	}
	if currentEvidenceHash != task.LastGate.EvidenceSHA256 {
		return Task{}, errors.New("evidence package changed after DoneGate")
	}
	imported, err := verifyImportedEvidenceTree(task)
	if err != nil {
		return Task{}, err
	}
	if !imported {
		changed, collectErr := collectChangedFiles(ctx, task)
		if collectErr != nil {
			return Task{}, collectErr
		}
		diff, diffErr := collectDiff(ctx, task, changed)
		if diffErr != nil {
			return Task{}, diffErr
		}
		if worktreeTreeHash(diff, changed) != task.LastGate.WorktreeTreeSHA {
			return Task{}, errors.New("worktree changed after DoneGate; rerun checks before acceptance")
		}
	}
	task.Status = TaskDone
	setAllWorkItemsStatus(&task, WorkItemDone)
	decision := governance.ToDecision(review, "approved")
	task.AcceptedBy = &decision
	task.UpdatedAt = decision.CreatedAt
	if err := s.recordTaskEvent(task, "task.accepted", decision.ReviewerID, map[string]any{
		"decision":      decision,
		"evidence_path": task.LastEvidence,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

// CommitTask creates a local commit only after the human has accepted an
// unchanged, passing EvidencePackage. It never pushes or opens a pull request.
func (s *Service) CommitTask(ctx context.Context, id, actor, message string) (Task, error) {
	release, err := s.acquireRun(id, false)
	if err != nil {
		return Task{}, err
	}
	defer release()

	task, err := s.GetTask(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status != TaskDone {
		return Task{}, fmt.Errorf("task %s must be done before commit", id)
	}
	if task.CommitSHA != "" {
		return Task{}, fmt.Errorf("task %s is already committed at %s", id, task.CommitSHA)
	}
	if task.LastGate == nil || !task.LastGate.Passed {
		return Task{}, errors.New("task has no passing DoneGate")
	}
	imported, err := verifyImportedEvidenceTree(task)
	if err != nil {
		return Task{}, err
	}
	if imported {
		return Task{}, errors.New(
			"imported workstation evidence cannot be committed by the control plane; commit from the owning workstation after acceptance",
		)
	}
	changed, err := collectChangedFiles(ctx, task)
	if err != nil {
		return Task{}, err
	}
	diff, err := collectDiff(ctx, task, changed)
	if err != nil {
		return Task{}, err
	}
	if worktreeTreeHash(diff, changed) != task.LastGate.WorktreeTreeSHA {
		return Task{}, errors.New("worktree changed after human acceptance; rerun the task checks")
	}
	if len(changed) == 0 {
		return Task{}, errors.New("no changed files to commit")
	}
	addArgs := append([]string{"add", "--"}, changed...)
	if result, err := runGit(ctx, task.WorktreePath, addArgs...); err != nil {
		return Task{}, fmt.Errorf("stage task files: %w: %s", err, strings.TrimSpace(result.Stderr))
	}
	if strings.TrimSpace(message) == "" {
		message = task.Title
	}
	message = commitMessageWithTraceability(task, message)
	if result, err := runGit(ctx, task.WorktreePath, "commit", "-m", message); err != nil {
		return Task{}, fmt.Errorf("commit accepted task: %w: %s", err, strings.TrimSpace(result.Stderr))
	}
	head, err := runGit(ctx, task.WorktreePath, "rev-parse", "HEAD")
	if err != nil {
		return Task{}, err
	}
	task.CommitSHA = strings.TrimSpace(head.Stdout)
	task.UpdatedAt = time.Now().UTC()
	if err := s.recordTaskEvent(task, "task.committed", valueOr(actor, "human"), map[string]any{
		"commit_sha": task.CommitSHA,
		"branch":     task.Branch,
		"message":    message,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Service) CancelTask(id, actor, reason string) (Task, error) {
	if s.governance.Enabled && s.governance.RequireAuthenticatedReviewers {
		return Task{}, errors.New("authenticated governance requires CancelTaskWithReview")
	}
	return s.CancelTaskWithReview(id, reason, governance.Review{
		ReviewerID:    actor,
		Rationale:     valueOr(reason, "cancel the development task"),
		Role:          governance.RoleTaskCancel,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *Service) CancelTaskWithReview(
	id, reason string,
	review governance.Review,
) (Task, error) {
	release, err := s.acquireRun(id, false)
	if err != nil {
		return Task{}, err
	}
	defer release()

	task, err := s.GetTask(id)
	if err != nil {
		return Task{}, err
	}
	if task.Status == TaskDone || task.Status == TaskCancelled {
		return Task{}, fmt.Errorf("task in status %s cannot be cancelled", task.Status)
	}
	if strings.TrimSpace(reason) == "" {
		return Task{}, errors.New("cancellation reason is required")
	}
	if err := governance.ValidateRole(review, governance.RoleTaskCancel); err != nil {
		return Task{}, err
	}
	if err := governance.ValidateApproval(s.governance, review); err != nil {
		return Task{}, err
	}
	decision := governance.ToDecision(review, "cancelled")
	task.Status = TaskCancelled
	task.UpdatedAt = decision.CreatedAt
	if err := s.recordTaskEvent(task, "task.cancelled", decision.ReviewerID, map[string]any{
		"reason":   strings.TrimSpace(reason),
		"decision": decision,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (s *Service) runVerifications(ctx context.Context, task Task, runDir string) []VerificationResult {
	results := make([]VerificationResult, 0, len(task.EvidencePlan.Commands))
	var (
		verificationHome        string
		verificationEnvironment []string
		environmentErr          error
	)
	if len(s.cfg.VerificationSandbox) > 0 {
		verificationHome = filepath.Join(runDir, "verification-home")
		verificationEnvironment, environmentErr = prepareVerificationEnvironment(
			verificationHome,
		)
	}
	for index, spec := range task.EvidencePlan.Commands {
		result := VerificationResult{
			Name:     valueOr(spec.Name, fmt.Sprintf("verification-%d", index+1)),
			Argv:     spec.Argv,
			ExitCode: -1,
		}
		if len(spec.Argv) == 0 {
			result.Stderr = "empty argv"
			results = append(results, result)
			_ = writeJSONAtomic(
				filepath.Join(runDir, fmt.Sprintf("verification-%02d.json", index+1)),
				result,
				0o600,
			)
			continue
		}
		if len(s.cfg.VerificationSandbox) == 0 &&
			!s.cfg.UnsafeHostVerification {
			result.Stderr = strings.Join([]string{
				"verification command was not executed: verification_sandbox is required",
				"configure an absolute trusted wrapper or explicitly opt into",
				"unsafe_host_verification only inside an already isolated VM/container",
			}, " ")
			results = append(results, result)
			_ = writeJSONAtomic(
				filepath.Join(runDir, fmt.Sprintf("verification-%02d.json", index+1)),
				result,
				0o600,
			)
			continue
		}
		if environmentErr != nil {
			result.Stderr = "prepare verification sandbox environment: " +
				environmentErr.Error()
			results = append(results, result)
			_ = writeJSONAtomic(
				filepath.Join(runDir, fmt.Sprintf("verification-%02d.json", index+1)),
				result,
				0o600,
			)
			continue
		}
		timeout := time.Duration(s.cfg.VerifyTimeoutSeconds) * time.Second
		commandCtx, cancel := context.WithTimeout(ctx, timeout)
		var commandResult commandResult
		var err error
		if len(s.cfg.VerificationSandbox) > 0 {
			commandArgs := append(
				append(
					append(
						[]string(nil),
						s.cfg.VerificationSandbox[1:]...,
					),
					task.WorktreePath,
					verificationHome,
					"--",
					spec.Argv[0],
				),
				spec.Argv[1:]...,
			)
			commandResult, err = runCommandWithEnvironment(
				commandCtx,
				task.WorktreePath,
				"",
				verificationEnvironment,
				s.cfg.VerificationSandbox[0],
				commandArgs...,
			)
		} else {
			commandResult, err = runCommand(
				commandCtx,
				task.WorktreePath,
				"",
				nil,
				spec.Argv[0],
				spec.Argv[1:]...,
			)
		}
		cancel()
		result.ExitCode = commandResult.ExitCode
		result.Stdout = commandResult.Stdout
		result.Stderr = commandResult.Stderr
		result.DurationMS = commandResult.DurationMS
		result.TimedOut = commandResult.TimedOut
		result.Passed = err == nil && commandResult.ExitCode == 0
		results = append(results, result)
		_ = writeJSONAtomic(filepath.Join(runDir, fmt.Sprintf("verification-%02d.json", index+1)), result, 0o600)
	}
	return results
}

func prepareVerificationEnvironment(home string) ([]string, error) {
	absoluteHome, err := filepath.Abs(home)
	if err != nil {
		return nil, fmt.Errorf("resolve verification home: %w", err)
	}
	directories := map[string]string{
		"HOME":            absoluteHome,
		"CODEX_HOME":      filepath.Join(absoluteHome, ".codex"),
		"XDG_CONFIG_HOME": filepath.Join(absoluteHome, ".config"),
		"XDG_CACHE_HOME":  filepath.Join(absoluteHome, ".cache"),
		"XDG_DATA_HOME":   filepath.Join(absoluteHome, ".local", "share"),
		"XDG_STATE_HOME":  filepath.Join(absoluteHome, ".local", "state"),
		"XDG_RUNTIME_DIR": filepath.Join(absoluteHome, ".runtime"),
		"TMPDIR":          filepath.Join(absoluteHome, "tmp"),
		"TMP":             filepath.Join(absoluteHome, "tmp"),
		"TEMP":            filepath.Join(absoluteHome, "tmp"),
	}
	created := make(map[string]struct{})
	for _, directory := range directories {
		if _, ok := created[directory]; ok {
			continue
		}
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return nil, fmt.Errorf(
				"create isolated verification directory: %w",
				err,
			)
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			return nil, fmt.Errorf(
				"secure isolated verification directory: %w",
				err,
			)
		}
		created[directory] = struct{}{}
	}
	if strings.TrimSpace(os.Getenv("PATH")) == "" {
		return nil, errors.New("PATH is required")
	}
	environment := inheritedSafeRuntimeEnvironment()
	for _, name := range []string{
		"HOME",
		"CODEX_HOME",
		"XDG_CONFIG_HOME",
		"XDG_CACHE_HOME",
		"XDG_DATA_HOME",
		"XDG_STATE_HOME",
		"XDG_RUNTIME_DIR",
		"TMPDIR",
		"TMP",
		"TEMP",
	} {
		environment = append(environment, name+"="+directories[name])
	}
	environment = append(environment, "GIT_TERMINAL_PROMPT=0")
	return environment, nil
}

func evaluateDoneGate(task Task, pack EvidencePackage, runDir, evidenceHash, treeHash string) DoneGateResult {
	result := DoneGateResult{
		Passed:          true,
		Verdict:         "pass",
		EvidencePath:    runDir,
		EvidenceSHA256:  evidenceHash,
		WorktreeTreeSHA: treeHash,
		EvaluatedAt:     time.Now().UTC(),
		EvaluatedBy:     "go-orchestrator-lite",
	}
	if pack.Before.HeadCommit != task.Compile.BaseCommit {
		result.Reasons = append(
			result.Reasons,
			fmt.Sprintf(
				"repository before HEAD %s does not match frozen base %s",
				pack.Before.HeadCommit,
				task.Compile.BaseCommit,
			),
		)
	}
	if pack.After.HeadCommit != task.Compile.BaseCommit {
		result.Reasons = append(
			result.Reasons,
			fmt.Sprintf(
				"repository after HEAD %s does not match frozen base %s; the Codex Hand must not commit",
				pack.After.HeadCommit,
				task.Compile.BaseCommit,
			),
		)
	}
	if task.DoneGate.RequireChangedFiles && len(pack.Policy.ChangedFiles) == 0 {
		result.Reasons = append(result.Reasons, "no changed files were produced")
	}
	if task.DoneGate.RequirePolicyPass && !pack.Policy.Passed {
		result.Reasons = append(result.Reasons, pack.Policy.Violations...)
	}
	if task.DoneGate.RequireAllVerifications {
		for _, verification := range pack.Verification {
			if !verification.Passed {
				result.Reasons = append(result.Reasons, "verification failed: "+verification.Name)
			}
		}
	}
	if task.DoneGate.RequireIndependentReview && !pack.Review.Passed {
		result.Reasons = append(result.Reasons, append([]string{"independent review failed: " + pack.Review.Summary}, pack.Review.RequiredFix...)...)
	}
	if task.DoneGate.RequireWorkItemTrace && len(pack.Unattributed) > 0 {
		result.Reasons = append(result.Reasons,
			"changed files are not attributed to a work item: "+strings.Join(pack.Unattributed, ", "))
	}
	if task.DoneGate.RequirePolicyBundle && strings.TrimSpace(pack.PolicyBundleHash) == "" {
		result.Reasons = append(result.Reasons, "the frozen task has no policy bundle hash")
	}
	if task.DoneGate.RequireDocumentEvidence && len(pack.DocumentRefs) == 0 {
		result.Reasons = append(result.Reasons, "the frozen task has no required document references")
	}
	for _, falsifier := range pack.Falsifiers {
		if !falsifier.Checked {
			result.Reasons = append(result.Reasons, "falsifier was not checked: "+falsifier.CriterionID)
		} else if falsifier.Triggered {
			result.Reasons = append(result.Reasons, "success was falsified: "+falsifier.CriterionID)
		}
	}
	for _, prediction := range pack.Predictions {
		if prediction.Due && (!prediction.Checked || !prediction.Satisfied) {
			result.Reasons = append(result.Reasons, "due prediction was not satisfied: "+prediction.PredictionID)
		}
	}
	for _, check := range pack.KillChecks {
		if !check.Evaluated {
			result.Reasons = append(result.Reasons, "kill condition could not be evaluated: "+check.ConditionID)
		} else if check.Triggered {
			result.Reasons = append(result.Reasons,
				fmt.Sprintf("kill condition %s triggered action %s", check.ConditionID, check.Action))
		}
	}
	if task.Cost.MaxInputTokens > 0 && task.CumulativeUsage.InputTokens > task.Cost.MaxInputTokens {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("cumulative input tokens %d exceed approved budget %d", task.CumulativeUsage.InputTokens, task.Cost.MaxInputTokens))
	}
	if task.Cost.MaxOutputTokens > 0 && task.CumulativeUsage.OutputTokens > task.Cost.MaxOutputTokens {
		result.Reasons = append(result.Reasons,
			fmt.Sprintf("cumulative output tokens %d exceed approved budget %d", task.CumulativeUsage.OutputTokens, task.Cost.MaxOutputTokens))
	}
	result.Passed = len(result.Reasons) == 0
	if !result.Passed {
		result.Verdict = "fail"
	}
	return result
}

func buildFalsifierResults(
	task Task,
	verifications []VerificationResult,
	runDir string,
) []FalsifierResult {
	results := make([]FalsifierResult, 0, len(task.Goal.Falsifiers))
	for _, falsifier := range task.Goal.Falsifiers {
		result := FalsifierResult{
			CriterionID:      falsifier.CriterionID,
			Condition:        falsifier.Condition,
			EvidenceRequired: falsifier.EvidenceRequired,
			Reason:           "no verification command was mapped to this criterion",
		}
		for index, verification := range verifications {
			if !strings.HasPrefix(verification.Name, falsifier.CriterionID+":") {
				continue
			}
			result.Checked = true
			result.Triggered = result.Triggered || !verification.Passed
			result.EvidenceRefs = append(result.EvidenceRefs,
				filepath.Join(runDir, fmt.Sprintf("verification-%02d.json", index+1)),
			)
		}
		if result.Checked && result.Triggered {
			result.Reason = "mapped deterministic evidence triggered the preregistered falsifier"
		} else if result.Checked {
			result.Reason = "mapped deterministic evidence did not trigger the preregistered falsifier"
		}
		results = append(results, result)
	}
	return results
}

func buildPredictionChecks(
	task Task,
	policy PolicyResult,
	verifications []VerificationResult,
	review IndependentReview,
	runDir string,
) []PredictionCheck {
	results := make([]PredictionCheck, 0, len(task.Goal.Predictions))
	deterministicPass := policy.Passed && verificationsPassed(verifications)
	if task.DoneGate.RequireIndependentReview {
		deterministicPass = deterministicPass && review.Passed
	}
	for _, prediction := range task.Goal.Predictions {
		horizon := strings.ToLower(strings.TrimSpace(prediction.Horizon))
		due := horizon == "before acceptance" || horizon == "before_acceptance"
		check := PredictionCheck{
			PredictionID: prediction.ID,
			Horizon:      prediction.Horizon,
			Due:          due,
			Checked:      due,
			Satisfied:    !due || deterministicPass,
			Observation:  "prediction is not due in this execution window",
		}
		if due {
			check.EvidenceRefs = []string{
				filepath.Join(runDir, "policy.json"),
				filepath.Join(runDir, "verification.json"),
				filepath.Join(runDir, "independent-review.json"),
			}
			if deterministicPass {
				check.Observation = "all deterministic and required independent gates supporting the prediction passed"
			} else {
				check.Observation = "one or more gates supporting the prediction failed"
			}
		}
		results = append(results, check)
	}
	return results
}

func buildKillConditionChecks(task Task, policy PolicyResult) []KillConditionCheck {
	results := make([]KillConditionCheck, 0, len(task.Risk.KillConditions))
	for _, condition := range task.Risk.KillConditions {
		threshold, err := strconv.ParseFloat(strings.TrimSpace(condition.Threshold), 64)
		check := KillConditionCheck{
			ConditionID: condition.ID,
			Metric:      condition.Metric,
			Threshold:   threshold,
			Action:      condition.Action,
			Evaluated:   err == nil,
		}
		switch condition.Metric {
		case "changed_files":
			check.Observed = float64(len(policy.ChangedFiles))
		case "changed_lines":
			check.Observed = float64(policy.AddedLines + policy.DeletedLines)
		case "input_tokens":
			check.Observed = float64(task.CumulativeUsage.InputTokens)
		case "output_tokens":
			check.Observed = float64(task.CumulativeUsage.OutputTokens)
		case "repair_attempts":
			check.Observed = float64(task.RepairCount)
		default:
			check.Evaluated = false
		}
		if check.Evaluated {
			check.Triggered = check.Observed > check.Threshold
			check.Reason = fmt.Sprintf("observed %.0f; approved maximum %.0f", check.Observed, check.Threshold)
		} else {
			check.Reason = "metric or threshold is not machine-evaluable"
		}
		results = append(results, check)
	}
	return results
}

func hasTriggeredKill(checks []KillConditionCheck) bool {
	for _, check := range checks {
		if check.Triggered {
			return true
		}
	}
	return false
}

func addCodexUsage(total *CodexUsage, value CodexUsage) {
	total.InputTokens += value.InputTokens
	total.CachedInputTokens += value.CachedInputTokens
	total.OutputTokens += value.OutputTokens
	total.ReasoningOutputTokens += value.ReasoningOutputTokens
}

func (s *Service) buildExecutionPrompt(task Task, repair bool) (string, error) {
	taskJSON, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return "", err
	}
	lines := []string{
		"You are the controlled Codex development Hand for GoClaw Orchestrator Lite.",
		"Implement only the frozen task below inside the current Git worktree.",
		"Do not commit, push, open a pull request, modify Git configuration, or leave the worktree.",
		"Respect allowed_paths, denied_paths, maximum changed files/lines, dependency policy, non-goals, and risk constraints.",
		"Use repository AGENTS.md and project-local instructions when present.",
		"Run focused checks while working. The Go DoneGate will independently rerun every frozen verification command.",
		"Do not claim completion when checks fail. Leave the worktree in the most reviewable state possible.",
	}
	if repair {
		lines = append(lines,
			"This is a repair run. Inspect the previous EvidencePackage and DoneGate reasons described in the frozen task state, then make the smallest correction.")
		if task.LastGate != nil {
			reasons, _ := json.Marshal(task.LastGate.Reasons)
			lines = append(lines, "PREVIOUS_DONEGATE_REASONS: "+string(reasons))
		}
	}
	if len(task.PolicyInstructions) > 0 {
		lines = append(lines,
			"\nFROZEN_POLICY_INSTRUCTIONS:\n- "+strings.Join(task.PolicyInstructions, "\n- "))
	}
	lines = append(lines, "\nFROZEN_EXECUTION_BUNDLE:\n"+string(taskJSON))
	return strings.Join(lines, "\n"), nil
}

func validateExecutionStatus(task Task, repair, resume bool) error {
	if resume {
		switch task.Status {
		case TaskRunning, TaskChecking, TaskRepairPending, TaskFailed:
			return nil
		default:
			return fmt.Errorf("task %s cannot resume from status %s", task.ID, task.Status)
		}
	}
	if repair {
		if task.Status != TaskRepairPending && task.Status != TaskFailed {
			return fmt.Errorf("task %s cannot repair from status %s", task.ID, task.Status)
		}
		return nil
	}
	if task.Status != TaskFrozen {
		return fmt.Errorf("task %s must be frozen before run (status %s)", task.ID, task.Status)
	}
	return nil
}

func verificationsPassed(results []VerificationResult) bool {
	if len(results) == 0 {
		return false
	}
	for _, result := range results {
		if !result.Passed {
			return false
		}
	}
	return true
}

func (s *Service) recordTaskEvent(task Task, eventType, actor string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.loadTaskUnlocked(task.ID)
	if err != nil {
		return err
	}
	if current.Compile.Revision != task.Compile.Revision {
		return errors.New("task revision changed during execution")
	}
	return s.appendEventUnlocked(task, eventType, actor, data)
}

func (s *Service) recordFailure(id, actor, phase string, failure error) error {
	task, err := s.GetTask(id)
	if err != nil {
		return err
	}
	task.Status = TaskFailed
	task.UpdatedAt = time.Now().UTC()
	return s.recordTaskEvent(task, "run.failed", actor, map[string]any{"phase": phase, "error": failure.Error()})
}

func (s *Service) recordRunFailure(task Task, actor, phase string, failure error, evidencePath string) error {
	task.Status = TaskFailed
	task.LastEvidence = evidencePath
	task.UpdatedAt = time.Now().UTC()
	return s.recordTaskEvent(task, "run.failed", actor, map[string]any{
		"phase": phase, "error": failure.Error(), "evidence_path": evidencePath,
	})
}

func (s *Service) acquireRun(id string, force bool) (func(), error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.busy[id] {
		s.mu.Unlock()
		return nil, fmt.Errorf("task %s is already running in this process", id)
	}
	s.busy[id] = true
	s.mu.Unlock()
	lockPath := filepath.Join(s.locksDir(), id+".lock")
	if force {
		_ = os.Remove(lockPath)
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		s.mu.Lock()
		delete(s.busy, id)
		s.mu.Unlock()
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("task %s has a run lock; use resume --force only after confirming the previous process stopped", id)
		}
		return nil, err
	}
	_, _ = fmt.Fprintf(file, `{"pid":%d,"created_at":%q}`+"\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
	_ = file.Sync()
	_ = file.Close()
	return func() {
		_ = os.Remove(lockPath)
		s.mu.Lock()
		delete(s.busy, id)
		s.mu.Unlock()
	}, nil
}

func (s *Service) recoverThreadID(task Task) string {
	if task.CurrentRunID == "" {
		return ""
	}
	path := filepath.Join(s.runDir(task.ID, task.CurrentRunID), "codex-events.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event map[string]any
		if json.Unmarshal(scanner.Bytes(), &event) == nil && event["type"] == "thread.started" {
			threadID, _ := event["thread_id"].(string)
			return threadID
		}
	}
	return ""
}

func (s *Service) runDir(taskID, runID string) string {
	return filepath.Join(s.taskDir(taskID), "runs", runID)
}
