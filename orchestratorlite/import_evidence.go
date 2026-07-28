package orchestratorlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxImportedDiffBytes = 10 * 1024 * 1024

// ImportExecutionEvidence binds a control-plane-verified external execution
// receipt to one immutable frozen task revision. It does not trust the remote
// DoneGate verdict: paths, diff identity, frozen checks, base/head state and
// the local DoneGate are validated again before the task can advance.
func (s *Service) ImportExecutionEvidence(
	ctx context.Context,
	input ImportExecutionEvidenceInput,
) (Task, error) {
	if err := validateID(input.TaskID); err != nil {
		return Task{}, err
	}
	release, err := s.acquireRun(input.TaskID, false)
	if err != nil {
		return Task{}, err
	}
	defer release()

	task, err := s.GetTask(input.TaskID)
	if err != nil {
		return Task{}, err
	}
	normalized, err := validateImportedExecutionEvidence(ctx, task, input)
	if err != nil {
		return Task{}, err
	}
	input.Evidence = normalized

	if importedEvidenceMatches(task, input.Evidence.BundleSHA256) {
		return task, nil
	}
	if task.Status != TaskFrozen {
		return Task{}, fmt.Errorf(
			"task %s must be frozen before evidence import (status %s)",
			task.ID,
			task.Status,
		)
	}
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}

	runID := "import-" + strings.ToLower(input.Evidence.BundleSHA256)
	runDir := s.runDir(task.ID, runID)
	if err := ensureDir(runDir); err != nil {
		return Task{}, err
	}
	diffPath := filepath.Join(runDir, "diff.patch")
	if err := writeBytesAtomic(diffPath, []byte(input.DiffPatch), 0o600); err != nil {
		return Task{}, err
	}

	policy, err := evaluateImportedPolicy(ctx, task, input.Evidence, input.DiffPatch)
	if err != nil {
		return Task{}, err
	}
	verification := importedVerificationResults(input.Evidence.Checks)
	review := IndependentReview{
		Passed:  true,
		Summary: "Independent model review is disabled by the frozen DoneGate.",
	}
	var reviewHand HandResult
	if task.DoneGate.RequireIndependentReview {
		if policy.Passed && verificationsPassed(verification) {
			var reviewErr error
			review, reviewHand, reviewErr = s.hand.Review(ctx, ReviewRequest{
				Task:       task,
				RunID:      runID,
				Diff:       input.DiffPatch,
				Evidence:   verification,
				EventsPath: filepath.Join(runDir, "codex-review-events.jsonl"),
			})
			addCodexUsage(&task.CumulativeUsage, reviewHand.Usage)
			if reviewErr != nil {
				review = IndependentReview{
					Passed:   false,
					Summary:  "Independent review could not complete.",
					Findings: []string{reviewErr.Error()},
				}
			}
		} else {
			review = IndependentReview{
				Passed:   false,
				Summary:  "Independent review skipped because deterministic checks failed.",
				Findings: []string{"Resolve policy and verification failures before model review."},
			}
		}
	}
	attribution, unattributed := attributeChangedFiles(task, policy.ChangedFiles)
	falsifiers := buildFalsifierResults(task, verification, runDir)
	predictions := buildPredictionChecks(task, policy, verification, review, runDir)
	killChecks := buildKillConditionChecks(task, policy)

	before := RepositorySnapshot{
		RepoPath:   task.RepoPath,
		Branch:     input.Evidence.Branch,
		BaseCommit: input.Evidence.BaseCommit,
		HeadCommit: input.Evidence.BaseCommit,
		CapturedAt: input.Evidence.StartedAt,
	}
	after := RepositorySnapshot{
		RepoPath:   task.RepoPath,
		Branch:     input.Evidence.Branch,
		BaseCommit: input.Evidence.BaseCommit,
		HeadCommit: input.Evidence.HeadCommit,
		CapturedAt: input.Evidence.FinishedAt,
	}
	imported := input.Evidence
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
		Hand: HandResult{
			FinalText: input.Evidence.Summary,
			ExitCode:  importedExecutionExitCode(input.Evidence),
		},
		Policy:       policy,
		Verification: verification,
		Review:       review,
		Attribution:  attribution,
		Unattributed: unattributed,
		Falsifiers:   falsifiers,
		Predictions:  predictions,
		KillChecks:   killChecks,
		DiffPath:     diffPath,
		Imported:     &imported,
		CreatedAt:    time.Now().UTC(),
	}
	for path, value := range map[string]any{
		"repository-before.json":     before,
		"repository-after.json":      after,
		"imported-evidence.json":     imported,
		"policy.json":                policy,
		"verification.json":          verification,
		"independent-review.json":    review,
		"codex-review-result.json":   reviewHand,
		"falsifier-results.json":     falsifiers,
		"prediction-checks.json":     predictions,
		"kill-condition-checks.json": killChecks,
		"change-attribution.json": map[string]any{
			"attribution":  attribution,
			"unattributed": unattributed,
		},
	} {
		if err := writeJSONAtomic(filepath.Join(runDir, path), value, 0o600); err != nil {
			return Task{}, err
		}
	}

	evidencePath := filepath.Join(runDir, "evidence.json")
	if err := writeJSONAtomic(evidencePath, pack, 0o600); err != nil {
		return Task{}, err
	}
	evidenceHash, err := sha256File(evidencePath)
	if err != nil {
		return Task{}, err
	}
	treeHash := worktreeTreeHash(input.DiffPatch, policy.ChangedFiles)
	gate := evaluateDoneGate(task, pack, runDir, evidenceHash, treeHash)
	if err := writeJSONAtomic(filepath.Join(runDir, "donegate.json"), gate, 0o600); err != nil {
		return Task{}, err
	}

	task.CurrentRunID = runID
	task.RunCount++
	task.LastGate = &gate
	task.LastEvidence = runDir
	task.UpdatedAt = time.Now().UTC()
	if gate.Passed {
		if task.DoneGate.RequireHumanAcceptance {
			task.Status = TaskAwaitingAcceptance
			setAllWorkItemsStatus(&task, WorkItemVerifying)
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
	if err := s.recordTaskEvent(task, eventType, input.Evidence.VerifiedBy, map[string]any{
		"source":                input.Evidence.Source,
		"run_id":                runID,
		"runner_id":             input.Evidence.RunnerID,
		"lease_id":              input.Evidence.LeaseID,
		"attempt":               input.Evidence.Attempt,
		"bundle_sha256":         input.Evidence.BundleSHA256,
		"execution_pack_sha256": input.Evidence.ExecutionPackSHA256,
		"signature_algorithm":   input.Evidence.SignatureAlgorithm,
		"key_id":                input.Evidence.KeyID,
		"verified_at":           input.Evidence.VerifiedAt,
		"verdict":               gate.Verdict,
		"reasons":               gate.Reasons,
		"evidence_path":         runDir,
		"evidence_sha256":       evidenceHash,
	}); err != nil {
		return Task{}, err
	}
	return task, nil
}

func validateImportedExecutionEvidence(
	ctx context.Context,
	task Task,
	input ImportExecutionEvidenceInput,
) (ImportedExecutionEvidence, error) {
	evidence := input.Evidence
	switch {
	case input.TaskID != task.ID:
		return ImportedExecutionEvidence{}, errors.New("imported evidence task id does not match")
	case strings.TrimSpace(input.ProjectID) == "" || input.ProjectID != task.ProjectID:
		return ImportedExecutionEvidence{}, errors.New("imported evidence project id does not match")
	case input.TaskRevision != task.Compile.Revision:
		return ImportedExecutionEvidence{}, fmt.Errorf(
			"imported evidence revision %d does not match frozen revision %d",
			input.TaskRevision,
			task.Compile.Revision,
		)
	case strings.TrimSpace(input.ExecutionBundleHash) == "" ||
		input.ExecutionBundleHash != task.Compile.ExecutionBundleHash:
		return ImportedExecutionEvidence{}, errors.New("imported evidence execution bundle hash does not match")
	case strings.TrimSpace(evidence.Source) == "":
		return ImportedExecutionEvidence{}, errors.New("imported evidence source is required")
	case strings.TrimSpace(evidence.RunnerID) == "":
		return ImportedExecutionEvidence{}, errors.New("imported evidence runner id is required")
	case strings.TrimSpace(evidence.LeaseID) == "":
		return ImportedExecutionEvidence{}, errors.New("imported evidence lease id is required")
	case evidence.Attempt <= 0:
		return ImportedExecutionEvidence{}, errors.New("imported evidence attempt must be positive")
	case evidence.Outcome != "completed":
		return ImportedExecutionEvidence{}, errors.New("only completed execution evidence can be imported")
	case evidence.StartedAt.IsZero() || evidence.FinishedAt.IsZero():
		return ImportedExecutionEvidence{}, errors.New("imported evidence timestamps are required")
	case evidence.FinishedAt.Before(evidence.StartedAt):
		return ImportedExecutionEvidence{}, errors.New("imported evidence finished_at precedes started_at")
	case evidence.BaseCommit != task.Compile.BaseCommit:
		return ImportedExecutionEvidence{}, errors.New("imported evidence base commit does not match frozen task")
	case evidence.HeadCommit != task.Compile.BaseCommit:
		return ImportedExecutionEvidence{}, errors.New("imported evidence HEAD must equal the frozen base commit")
	case strings.TrimSpace(evidence.CommitSHA) != "":
		return ImportedExecutionEvidence{}, errors.New("imported evidence must not contain an automatic commit")
	case strings.TrimSpace(evidence.SignatureAlgorithm) == "":
		return ImportedExecutionEvidence{}, errors.New("imported evidence signature algorithm is required")
	case strings.TrimSpace(evidence.KeyID) == "":
		return ImportedExecutionEvidence{}, errors.New("imported evidence key id is required")
	case strings.TrimSpace(evidence.Signature) == "":
		return ImportedExecutionEvidence{}, errors.New("imported evidence signature is required")
	case evidence.VerifiedAt.IsZero():
		return ImportedExecutionEvidence{}, errors.New("imported evidence verified_at is required")
	case evidence.VerifiedAt.Before(evidence.FinishedAt):
		return ImportedExecutionEvidence{}, errors.New("imported evidence was marked verified before execution finished")
	case strings.TrimSpace(evidence.VerifiedBy) == "":
		return ImportedExecutionEvidence{}, errors.New("imported evidence verified_by is required")
	}
	for name, value := range map[string]string{
		"execution_pack_sha256": evidence.ExecutionPackSHA256,
		"bundle_sha256":         evidence.BundleSHA256,
	} {
		if err := validateImportedSHA256(name, value); err != nil {
			return ImportedExecutionEvidence{}, err
		}
	}
	if _, err := hex.DecodeString(strings.TrimSpace(evidence.KeyID)); err != nil {
		return ImportedExecutionEvidence{}, errors.New("imported evidence key id must be hexadecimal")
	}
	signature, err := hex.DecodeString(strings.TrimSpace(evidence.Signature))
	if err != nil || len(signature) < sha256.Size {
		return ImportedExecutionEvidence{}, errors.New("imported evidence signature must contain at least 32 hexadecimal bytes")
	}
	if len(input.DiffPatch) > maxImportedDiffBytes {
		return ImportedExecutionEvidence{}, fmt.Errorf(
			"imported diff exceeds %d bytes",
			maxImportedDiffBytes,
		)
	}
	if input.DiffPatch == "" {
		if strings.TrimSpace(evidence.DiffSHA256) != "" {
			return ImportedExecutionEvidence{}, errors.New("imported evidence has a diff digest without a diff")
		}
		if len(evidence.ChangedFiles) > 0 {
			return ImportedExecutionEvidence{}, errors.New("imported changed files require a recoverable diff")
		}
	} else {
		if err := validateImportedSHA256("diff_sha256", evidence.DiffSHA256); err != nil {
			return ImportedExecutionEvidence{}, err
		}
		if sha256Bytes([]byte(input.DiffPatch)) != strings.ToLower(strings.TrimSpace(evidence.DiffSHA256)) {
			return ImportedExecutionEvidence{}, errors.New("imported diff digest does not match diff content")
		}
	}

	changed, err := normalizeImportedChangedFiles(evidence.ChangedFiles)
	if err != nil {
		return ImportedExecutionEvidence{}, err
	}
	_, _, diffFiles, err := inspectImportedDiff(ctx, task.RepoPath, input.DiffPatch)
	if err != nil {
		return ImportedExecutionEvidence{}, err
	}
	if !equalStringSets(changed, diffFiles) {
		return ImportedExecutionEvidence{}, fmt.Errorf(
			"imported changed files do not match diff paths: claimed=%v diff=%v",
			changed,
			diffFiles,
		)
	}
	evidence.ChangedFiles = changed
	evidence.DiffSHA256 = strings.ToLower(strings.TrimSpace(evidence.DiffSHA256))
	evidence.ExecutionPackSHA256 = strings.ToLower(strings.TrimSpace(evidence.ExecutionPackSHA256))
	evidence.BundleSHA256 = strings.ToLower(strings.TrimSpace(evidence.BundleSHA256))
	evidence.KeyID = strings.ToLower(strings.TrimSpace(evidence.KeyID))
	evidence.Signature = strings.ToLower(strings.TrimSpace(evidence.Signature))
	evidence.Source = strings.TrimSpace(evidence.Source)
	evidence.VerifiedBy = strings.TrimSpace(evidence.VerifiedBy)
	evidence.TraceIDs = uniqueStrings(evidence.TraceIDs)

	if err := validateImportedChecks(task, evidence.Source, evidence.Checks); err != nil {
		return ImportedExecutionEvidence{}, err
	}
	for _, artifact := range evidence.Artifacts {
		if strings.TrimSpace(artifact.Name) == "" {
			return ImportedExecutionEvidence{}, errors.New("imported evidence artifact name is required")
		}
		if err := validateImportedSHA256("artifact "+artifact.Name, artifact.SHA256); err != nil {
			return ImportedExecutionEvidence{}, err
		}
		if artifact.SizeBytes < 0 {
			return ImportedExecutionEvidence{}, fmt.Errorf(
				"imported evidence artifact %q has a negative size",
				artifact.Name,
			)
		}
	}
	return evidence, nil
}

func validateImportedChecks(
	task Task,
	source string,
	checks []ImportedEvidenceCheck,
) error {
	expected := make(map[string]int, len(task.EvidencePlan.Commands))
	for index, command := range task.EvidencePlan.Commands {
		name := valueOr(strings.TrimSpace(command.Name), strings.Join(command.Argv, " "))
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("verification-%d", index+1)
		}
		expected[name]++
	}
	if strings.EqualFold(strings.TrimSpace(source), "workstation") {
		for _, name := range []string{
			"runner-setup",
			"codex-exec",
			"scope-policy",
			"no-automatic-commit",
		} {
			expected[name]++
		}
	}
	seen := make(map[string]int, len(checks))
	for _, check := range checks {
		name := strings.TrimSpace(check.Name)
		if name == "" {
			return errors.New("imported evidence check name is required")
		}
		if check.DurationMS < 0 {
			return fmt.Errorf("imported evidence check %q has a negative duration", name)
		}
		seen[name]++
	}
	for name, count := range expected {
		if seen[name] != count {
			return fmt.Errorf(
				"imported evidence requires %d %q check(s), got %d",
				count,
				name,
				seen[name],
			)
		}
	}
	return nil
}

func evaluateImportedPolicy(
	ctx context.Context,
	task Task,
	evidence ImportedExecutionEvidence,
	diff string,
) (PolicyResult, error) {
	result := PolicyResult{
		Passed:       true,
		ChangedFiles: append([]string(nil), evidence.ChangedFiles...),
	}
	for _, path := range evidence.ChangedFiles {
		if matchesAnyPath(path, task.Scope.DeniedPaths) {
			result.Violations = append(result.Violations, "denied path changed: "+path)
		}
		if len(task.Scope.AllowedPaths) > 0 && !matchesAnyPath(path, task.Scope.AllowedPaths) {
			result.Violations = append(result.Violations, "path outside approved scope: "+path)
		}
		if dependencyManifest(path) {
			result.NewDependencies = append(result.NewDependencies, path)
		}
	}
	if task.Scope.MaxChangedFiles > 0 && len(evidence.ChangedFiles) > task.Scope.MaxChangedFiles {
		result.Violations = append(
			result.Violations,
			fmt.Sprintf(
				"changed files %d exceed approved limit %d",
				len(evidence.ChangedFiles),
				task.Scope.MaxChangedFiles,
			),
		)
	}
	added, deleted, _, err := inspectImportedDiff(ctx, task.RepoPath, diff)
	if err != nil {
		return result, err
	}
	result.AddedLines = added
	result.DeletedLines = deleted
	totalLines := added + deleted
	if task.Scope.MaxChangedLines > 0 && totalLines > task.Scope.MaxChangedLines {
		result.Violations = append(
			result.Violations,
			fmt.Sprintf(
				"changed lines %d exceed approved limit %d",
				totalLines,
				task.Scope.MaxChangedLines,
			),
		)
	}
	if len(result.NewDependencies) > 0 && !task.Scope.AllowNewDependency {
		result.Violations = append(
			result.Violations,
			"dependency manifests changed without capacity approval: "+
				strings.Join(result.NewDependencies, ", "),
		)
	}
	for _, check := range evidence.Checks {
		if strings.TrimSpace(check.Name) == "scope-policy" && (!check.Passed || check.ExitCode != 0) {
			detail := strings.TrimSpace(check.Details)
			if detail == "" {
				detail = "external scope-policy check failed"
			}
			result.Violations = append(result.Violations, detail)
		}
	}
	sort.Strings(result.Violations)
	result.Passed = len(result.Violations) == 0
	return result, nil
}

func inspectImportedDiff(
	ctx context.Context,
	repository string,
	diff string,
) (int, int, []string, error) {
	if diff == "" {
		return 0, 0, nil, nil
	}
	result, err := runCommand(
		ctx,
		repository,
		diff,
		[]string{"GIT_CONFIG_NOSYSTEM=1"},
		"git",
		"apply",
		"--numstat",
		"-z",
	)
	if err != nil {
		return 0, 0, nil, fmt.Errorf(
			"inspect imported diff: %w: %s",
			err,
			strings.TrimSpace(result.Stderr),
		)
	}
	var added, deleted int
	var paths []string
	for _, record := range strings.Split(result.Stdout, "\x00") {
		if record == "" {
			continue
		}
		parts := strings.SplitN(record, "\t", 3)
		if len(parts) != 3 {
			return 0, 0, nil, errors.New("inspect imported diff: malformed numstat record")
		}
		if value, parseErr := strconv.Atoi(parts[0]); parseErr == nil {
			added += value
		}
		if value, parseErr := strconv.Atoi(parts[1]); parseErr == nil {
			deleted += value
		}
		path := filepath.ToSlash(strings.TrimSpace(parts[2]))
		if path == "" {
			return 0, 0, nil, errors.New("inspect imported diff: empty changed path")
		}
		paths = append(paths, path)
	}
	normalized, err := normalizeImportedChangedFiles(paths)
	if err != nil {
		return 0, 0, nil, err
	}
	return added, deleted, normalized, nil
}

func normalizeImportedChangedFiles(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
		if clean == "" || clean == "." || clean == ".." ||
			strings.HasPrefix(clean, "../") || filepath.IsAbs(value) {
			return nil, fmt.Errorf("invalid imported changed path %q", value)
		}
		if _, duplicate := seen[clean]; duplicate {
			return nil, fmt.Errorf("duplicate imported changed path %q", clean)
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	sort.Strings(result)
	return result, nil
}

func importedVerificationResults(checks []ImportedEvidenceCheck) []VerificationResult {
	results := make([]VerificationResult, 0, len(checks))
	for _, check := range checks {
		stderr := check.Stderr
		if strings.TrimSpace(check.Details) != "" {
			if strings.TrimSpace(stderr) != "" {
				stderr += "\n"
			}
			stderr += check.Details
		}
		results = append(results, VerificationResult{
			Name:       strings.TrimSpace(check.Name),
			Argv:       append([]string(nil), check.Argv...),
			ExitCode:   check.ExitCode,
			Stdout:     check.Stdout,
			Stderr:     stderr,
			DurationMS: check.DurationMS,
			TimedOut:   check.TimedOut,
			Passed:     check.Passed && check.ExitCode == 0 && !check.TimedOut,
		})
	}
	return results
}

func importedExecutionExitCode(evidence ImportedExecutionEvidence) int {
	if evidence.Outcome != "completed" {
		return 1
	}
	for _, check := range evidence.Checks {
		if !check.Passed || check.ExitCode != 0 || check.TimedOut {
			return 1
		}
	}
	return 0
}

func validateImportedSHA256(name, value string) error {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("imported evidence %s must be a SHA-256 digest", name)
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("imported evidence %s must be hexadecimal", name)
	}
	return nil
}

func equalStringSets(left, right []string) bool {
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

func importedEvidenceMatches(task Task, bundleSHA256 string) bool {
	if strings.TrimSpace(task.LastEvidence) == "" {
		return false
	}
	var pack EvidencePackage
	if err := readJSON(filepath.Join(task.LastEvidence, "evidence.json"), &pack); err != nil {
		return false
	}
	return pack.TaskID == task.ID &&
		pack.TaskRevision == task.Compile.Revision &&
		pack.Imported != nil &&
		strings.EqualFold(pack.Imported.BundleSHA256, strings.TrimSpace(bundleSHA256))
}

func verifyImportedEvidenceTree(task Task) (bool, error) {
	if strings.TrimSpace(task.LastEvidence) == "" {
		return false, nil
	}
	var pack EvidencePackage
	if err := readJSON(filepath.Join(task.LastEvidence, "evidence.json"), &pack); err != nil {
		return false, err
	}
	if pack.Imported == nil {
		return false, nil
	}
	diffPath := filepath.Join(task.LastEvidence, "diff.patch")
	if err := requireRegularFile(diffPath); err != nil {
		return true, err
	}
	diff, err := os.ReadFile(diffPath)
	if err != nil {
		return true, err
	}
	if pack.Imported.DiffSHA256 == "" {
		if len(diff) != 0 {
			return true, errors.New("imported diff was added after DoneGate")
		}
	} else if sha256Bytes(diff) != pack.Imported.DiffSHA256 {
		return true, errors.New("imported diff changed after DoneGate")
	}
	if task.LastGate == nil ||
		worktreeTreeHash(string(diff), pack.Policy.ChangedFiles) != task.LastGate.WorktreeTreeSHA {
		return true, errors.New("imported execution tree changed after DoneGate")
	}
	return true, nil
}
