package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (s *Service) ListEvalCases() ([]EvalCase, error) {
	dir := filepath.Join(s.evalsDir(), "cases")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	result := make([]EvalCase, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var evalCase EvalCase
		if ext == ".json" {
			err = json.Unmarshal(data, &evalCase)
		} else {
			err = yaml.Unmarshal(data, &evalCase)
		}
		if err != nil {
			return nil, fmt.Errorf("parse eval %s: %w", entry.Name(), err)
		}
		if evalCase.ID == "" {
			return nil, fmt.Errorf("eval %s has no id", entry.Name())
		}
		result = append(result, evalCase)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (s *Service) ValidateExperiment(ctx context.Context, experimentID string, execute bool) (ValidationReport, error) {
	exp, err := s.GetExperiment(experimentID)
	if err != nil {
		return ValidationReport{}, err
	}
	baseManifest, err := s.LoadManifest(exp.BaseVersion)
	if err != nil {
		return ValidationReport{}, err
	}
	var candidateManifest Manifest
	if err := readYAML(filepath.Join(exp.CandidatePath, "manifest.yaml"), &candidateManifest); err != nil {
		return ValidationReport{}, err
	}
	cases, err := s.ListEvalCases()
	if err != nil {
		return ValidationReport{}, err
	}
	report := ValidationReport{
		SchemaVersion:   SchemaVersion,
		ExperimentID:    exp.ID,
		BaselineVersion: exp.BaseVersion,
		HarnessVersion:  exp.CandidateVersion,
		CreatedAt:       time.Now().UTC(),
	}
	report.Rejection = append(report.Rejection, validateCandidateManifest(exp, baseManifest, candidateManifest)...)
	changes, scopeRejections, err := s.validateCandidateScope(exp, baseManifest)
	if err != nil {
		return ValidationReport{}, err
	}
	report.CandidateChanges = changes
	report.Rejection = append(report.Rejection, scopeRejections...)

	baseline := Experiment{
		BaseVersion:      exp.BaseVersion,
		CandidateVersion: exp.BaseVersion,
		CandidatePath:    s.versionDir(exp.BaseVersion),
	}
	for _, evalCase := range cases {
		baselineResult := s.runEvalCase(ctx, baseline, evalCase, execute)
		report.BaselineResults = append(report.BaselineResults, baselineResult)
		result := s.runEvalCase(ctx, exp, evalCase, execute)
		report.Results = append(report.Results, result)
		if baselineResult.Passed && !result.Passed {
			report.Regressions = append(report.Regressions, EvalRegression{
				CaseID:            result.CaseID,
				BaselinePassed:    true,
				CandidatePassed:   false,
				CandidateFailures: append([]string(nil), result.Failures...),
			})
		}
	}
	report.BaselineScores = scoreEvalResults(report.BaselineResults)
	report.Scores = scoreEvalResults(report.Results)
	report.BaselineTokens, report.BaselineDurationMS = aggregateEvalMetrics(report.BaselineResults)
	report.CandidateTokens, report.CandidateDurationMS = aggregateEvalMetrics(report.Results)
	report.TokenDelta = relativeDelta(report.BaselineTokens, report.CandidateTokens)
	report.LatencyDelta = relativeDelta64(report.BaselineDurationMS, report.CandidateDurationMS)

	golden := report.Scores[SplitGolden]
	holdout := report.Scores[SplitHoldout]
	if golden.Critical > 0 || golden.Rate < baseManifest.MinimumGolden {
		report.Rejection = append(report.Rejection, fmt.Sprintf("golden score %.3f is below %.3f", golden.Rate, baseManifest.MinimumGolden))
	}
	if holdout.Critical > 0 || holdout.Rate < baseManifest.MinimumHoldout {
		report.Rejection = append(report.Rejection, fmt.Sprintf("holdout score %.3f is below %.3f", holdout.Rate, baseManifest.MinimumHoldout))
	}
	for _, result := range report.Results {
		if result.Critical && !result.Passed {
			report.Rejection = append(report.Rejection, "critical eval failed: "+result.CaseID)
		}
	}
	for _, regression := range report.Regressions {
		report.Rejection = append(report.Rejection, "baseline regression: "+regression.CaseID)
	}
	if report.BaselineTokens > 0 && report.TokenDelta > baseManifest.MaxTokenDelta {
		report.Rejection = append(report.Rejection,
			fmt.Sprintf("token delta %.3f exceeds %.3f", report.TokenDelta, baseManifest.MaxTokenDelta))
	}
	if report.BaselineDurationMS > 0 && report.LatencyDelta > baseManifest.MaxLatencyDelta {
		report.Rejection = append(report.Rejection,
			fmt.Sprintf("latency delta %.3f exceeds %.3f", report.LatencyDelta, baseManifest.MaxLatencyDelta))
	}
	report.Accepted = len(report.Rejection) == 0
	reportPath := filepath.Join(s.reportsDir(), exp.ID+".json")
	if err := writeJSONAtomic(reportPath, report, 0o644); err != nil {
		return ValidationReport{}, err
	}
	exp.Status = ExperimentValidated
	if !report.Accepted {
		exp.Status = ExperimentRejected
	}
	exp.ValidationReport = reportPath
	if err := s.updateExperiment(exp); err != nil {
		return ValidationReport{}, err
	}
	return report, nil
}

func (s *Service) runEvalCase(ctx context.Context, exp Experiment, evalCase EvalCase, execute bool) EvalResult {
	start := time.Now()
	result := EvalResult{
		CaseID:         evalCase.ID,
		Split:          evalCase.Split,
		Tags:           evalCase.Tags,
		Critical:       evalCase.Critical,
		Passed:         true,
		HarnessVersion: exp.CandidateVersion,
	}
	var trace *Trace
	if evalCase.TraceFixture != "" {
		path, err := safeJoin(s.evalsDir(), evalCase.TraceFixture)
		if err != nil {
			result.Failures = append(result.Failures, err.Error())
		} else {
			var fixture Trace
			if err := readJSON(path, &fixture); err != nil {
				result.Failures = append(result.Failures, "load trace fixture: "+err.Error())
			} else {
				trace = &fixture
				result.Stdout = fixture.Output
			}
		}
	}
	if len(evalCase.Runner.Command) > 0 {
		if !execute {
			result.Failures = append(result.Failures, "runner requires --execute")
		} else {
			s.runCommandCase(ctx, exp, evalCase, &result)
		}
	}
	result.Trace = trace
	gradeCase(evalCase, trace, &result)
	result.Passed = len(result.Failures) == 0
	result.Duration = time.Since(start)
	if trace != nil {
		result.MetricTokens = trace.TokenUsage.Total
		result.MetricDurationMS = trace.DurationMS
	}
	return result
}

func scoreEvalResults(results []EvalResult) map[EvalSplit]SplitScore {
	scores := make(map[EvalSplit]SplitScore)
	for _, result := range results {
		score := scores[result.Split]
		score.Total++
		if result.Passed {
			score.Passed++
		} else if result.Critical {
			score.Critical++
		}
		scores[result.Split] = score
	}
	for split, score := range scores {
		if score.Total > 0 {
			score.Rate = float64(score.Passed) / float64(score.Total)
		}
		scores[split] = score
	}
	return scores
}

func aggregateEvalMetrics(results []EvalResult) (tokens int, durationMS int64) {
	for _, result := range results {
		tokens += result.MetricTokens
		durationMS += result.MetricDurationMS
	}
	return tokens, durationMS
}

func relativeDelta(baseline, candidate int) float64 {
	if baseline <= 0 {
		return 0
	}
	return float64(candidate-baseline) / float64(baseline)
}

func relativeDelta64(baseline, candidate int64) float64 {
	if baseline <= 0 {
		return 0
	}
	return float64(candidate-baseline) / float64(baseline)
}

func (s *Service) validateCandidateScope(exp Experiment, manifest Manifest) ([]string, []string, error) {
	before, err := snapshotFiles(s.versionDir(exp.BaseVersion))
	if err != nil {
		return nil, nil, err
	}
	after, err := snapshotFiles(exp.CandidatePath)
	if err != nil {
		return nil, nil, err
	}
	changes := changedFiles(before, after)
	sort.Strings(changes)
	var rejections []string
	for _, changed := range changes {
		if changed == "manifest.yaml" {
			continue
		}
		for _, protected := range manifest.ProtectedPaths {
			if anyPathMatches([]string{changed}, protected) {
				rejections = append(rejections, fmt.Sprintf("protected path changed: %s (matches %s)", changed, protected))
			}
		}
		if !matchesAnyPath(changed, exp.ChangeManifest.TargetComponents) {
			rejections = append(rejections, "candidate changed undeclared component: "+changed)
		}
	}
	return changes, rejections, nil
}

func validateCandidateManifest(exp Experiment, base, candidate Manifest) []string {
	var result []string
	if candidate.Version != exp.CandidateVersion {
		result = append(result, "candidate manifest version does not match experiment")
	}
	if candidate.ParentVersion != exp.BaseVersion {
		result = append(result, "candidate manifest parent does not match base version")
	}
	normalized := candidate
	normalized.Version = base.Version
	normalized.ParentVersion = base.ParentVersion
	normalized.ChangeManifest = base.ChangeManifest
	normalized.CreatedAt = base.CreatedAt
	normalized.CreatedBy = base.CreatedBy
	if !reflect.DeepEqual(normalized, base) {
		result = append(result, "candidate manifest changed fields outside experiment metadata")
	}
	return result
}

func matchesAnyPath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if anyPathMatches([]string{path}, pattern) {
			return true
		}
	}
	return false
}

func (s *Service) runCommandCase(ctx context.Context, exp Experiment, evalCase EvalCase, result *EvalResult) {
	spec := evalCase.Runner
	timeout := time.Duration(spec.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	workDir := spec.WorkingDir
	if workDir == "" {
		workDir = "."
	}
	before, beforeErr := snapshotFiles(workDir)
	if beforeErr != nil {
		result.Failures = append(result.Failures, "snapshot before run: "+beforeErr.Error())
		return
	}
	command := exec.CommandContext(runCtx, spec.Command[0], spec.Command[1:]...)
	command.Dir = workDir
	command.Env = append(os.Environ(),
		"GOCLAW_HARNESS_ROOT="+s.cfg.Root,
		"GOCLAW_HARNESS_CANDIDATE="+exp.CandidatePath,
		"GOCLAW_HARNESS_VERSION="+exp.CandidateVersion,
	)
	for key, value := range spec.Environment {
		command.Env = append(command.Env, key+"="+value)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
			result.Failures = append(result.Failures, "execute runner: "+err.Error())
		}
	}
	after, afterErr := snapshotFiles(workDir)
	if afterErr != nil {
		result.Failures = append(result.Failures, "snapshot after run: "+afterErr.Error())
		return
	}
	result.ChangedFiles = changedFiles(before, after)
	sort.Strings(result.ChangedFiles)
}

func gradeCase(evalCase EvalCase, trace *Trace, result *EvalResult) {
	expected := evalCase.Expected
	if expected.ExitCode != nil && result.ExitCode != *expected.ExitCode {
		result.Failures = append(result.Failures, fmt.Sprintf("exit code %d, expected %d", result.ExitCode, *expected.ExitCode))
	}
	output := result.Stdout
	for _, needle := range expected.OutputContains {
		if !strings.Contains(strings.ToLower(output), strings.ToLower(needle)) {
			result.Failures = append(result.Failures, fmt.Sprintf("output does not contain %q", needle))
		}
	}
	for _, needle := range expected.OutputNotContains {
		if strings.Contains(strings.ToLower(output), strings.ToLower(needle)) {
			result.Failures = append(result.Failures, fmt.Sprintf("output contains forbidden text %q", needle))
		}
	}
	if expected.MaxDurationMS > 0 && result.Duration.Milliseconds() > expected.MaxDurationMS {
		result.Failures = append(result.Failures, fmt.Sprintf("duration %dms exceeds %dms", result.Duration.Milliseconds(), expected.MaxDurationMS))
	}
	for _, pattern := range expected.ForbiddenWrites {
		if anyPathMatches(result.ChangedFiles, pattern) {
			result.Failures = append(result.Failures, "forbidden write matched "+pattern)
		}
	}
	for _, pattern := range expected.RequiredWrites {
		if !anyPathMatches(result.ChangedFiles, pattern) {
			result.Failures = append(result.Failures, "required write missing "+pattern)
		}
	}
	if trace == nil {
		if expected.ExpectedProjectID != "" || expected.RequireEvidence || len(expected.RequiredContextFiles) > 0 ||
			len(expected.RequiredToolCalls) > 0 || len(expected.ForbiddenToolCalls) > 0 || expected.MaxTokens > 0 || expected.MaxToolCalls > 0 {
			result.Failures = append(result.Failures, "trace is required for trace-based expectations")
		}
		return
	}
	if expected.ExpectedProjectID != "" && trace.ProjectID != expected.ExpectedProjectID {
		result.Failures = append(result.Failures, fmt.Sprintf("project %q, expected %q", trace.ProjectID, expected.ExpectedProjectID))
	}
	for _, file := range expected.RequiredContextFiles {
		if !containsString(trace.Context.LoadedFiles, file) {
			result.Failures = append(result.Failures, "required context file missing "+file)
		}
	}
	toolNames := make([]string, 0, len(trace.ToolCalls))
	for _, tool := range trace.ToolCalls {
		toolNames = append(toolNames, tool.Name)
	}
	for _, name := range expected.RequiredToolCalls {
		if !containsString(toolNames, name) {
			result.Failures = append(result.Failures, "required tool call missing "+name)
		}
	}
	for _, name := range expected.ForbiddenToolCalls {
		if containsString(toolNames, name) {
			result.Failures = append(result.Failures, "forbidden tool call found "+name)
		}
	}
	if expected.MaxToolCalls > 0 && len(toolNames) > expected.MaxToolCalls {
		result.Failures = append(result.Failures, fmt.Sprintf("tool calls %d exceed %d", len(toolNames), expected.MaxToolCalls))
	}
	if expected.MaxTokens > 0 && trace.TokenUsage.Total > expected.MaxTokens {
		result.Failures = append(result.Failures, fmt.Sprintf("tokens %d exceed %d", trace.TokenUsage.Total, expected.MaxTokens))
	}
	if expected.RequireEvidence {
		evidence, ok := trace.Metadata["evidence"]
		if !ok || isEmptyEvidence(evidence) {
			result.Failures = append(result.Failures, "verification evidence is required")
		}
	}
}

func anyPathMatches(paths []string, pattern string) bool {
	pattern = filepath.ToSlash(pattern)
	for _, path := range paths {
		path = filepath.ToSlash(path)
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
	}
	return false
}

func pathPatternsOverlap(left, right string) bool {
	left = filepath.ToSlash(filepath.Clean(left))
	right = filepath.ToSlash(filepath.Clean(right))
	return anyPathMatches([]string{left}, right) ||
		anyPathMatches([]string{right}, left) ||
		strings.HasPrefix(strings.TrimSuffix(left, "/**")+"/", strings.TrimSuffix(right, "/**")+"/") ||
		strings.HasPrefix(strings.TrimSuffix(right, "/**")+"/", strings.TrimSuffix(left, "/**")+"/")
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func isEmptyEvidence(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	default:
		return false
	}
}
